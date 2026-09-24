#!/usr/bin/env python3
"""Collect MSYS2 source packages bound to a trusted Windows native inventory.

Recipes are compared by hash, never executed. Internal checksums are checked
where declared; pinned winpthreads Git and eight detached signatures are
verified offline. Key identity and license/source completeness remain review items.
"""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path, PurePosixPath
import re
import shutil
from urllib.parse import urlsplit

from msys2_binary_metadata import fields, sha, single, tar_stream
from verify_windows_vcs import load_lock as load_vcs_lock, verify_git_archive
from verify_windows_signatures import (EXPECTED_PACKAGES as SIGNED_PACKAGES,
                                       load_lock as load_signature_lock, verify_signature)

spec = importlib.util.spec_from_file_location('native_download', Path(__file__).with_name('collect-native-sources.py'))
download = importlib.util.module_from_spec(spec)
spec.loader.exec_module(download)

MAX_ARCHIVE = 256 * 1024 * 1024
MAX_TOTAL = 1024 * 1024 * 1024
MAX_UNPACKED = 2 * 1024 * 1024 * 1024
MAX_METADATA = 1024 * 1024
ALGORITHMS = {'sha256sums': 'sha256', 'sha512sums': 'sha512', 'sha1sums': 'sha1',
              'md5sums': 'md5', 'b2sums': 'blake2b'}


def source_plan(inventory):
    if inventory.get('schema') != 1 or not isinstance(inventory.get('packages'), dict):
        raise ValueError('Expected Windows native inventory schema 1')
    result = {}
    for owner, record in inventory['packages'].items():
        binary = record.get('binary_package', {})
        base, version, recipe = (binary.get(key, '') for key in ('source_package', 'version', 'pkgbuild_sha256'))
        if (not re.fullmatch(r'mingw-w64-ucrt-x86_64-[a-z0-9+_.-]+', owner)
                or not re.fullmatch(r'mingw-w64-[a-z0-9+_.-]+', base)
                or not re.fullmatch(r'[A-Za-z0-9._+~-]+', version)
                or record.get('version') != version or not re.fullmatch(r'[0-9a-f]{64}', recipe)):
            raise ValueError('Invalid source identity/recipe in native inventory')
        key = (base, version)
        if key in result and result[key]['pkgbuild_sha256'] != recipe:
            raise ValueError('Conflicting recipe hashes for one source package version')
        if key not in result:
            result[key] = {'base': base, 'version': version, 'pkgbuild_sha256': recipe, 'binary_packages': []}
        result[key]['binary_packages'].append(owner)
    if not 1 <= len(result) <= 64:
        raise ValueError('Expected 1–64 source packages')
    return [dict(result[key], binary_packages=sorted(result[key]['binary_packages'])) for key in sorted(result)]


def verify_inputs(srcinfo, files, links):
    checks = []
    if not srcinfo.get('source') and not srcinfo.get('source_x86_64'):
        raise ValueError('No source inputs declared for this target')
    # The requested Windows target is x86-64. Other architecture groups stay
    # in the retained SRCINFO, and are not claimed as validated here.
    for suffix in ('', '_x86_64'):
        values = srcinfo.get('source' + suffix, [])
        sums = {field: srcinfo[field + suffix] for field in ALGORITHMS if field + suffix in srcinfo}
        if any(len(entries) != len(values) for entries in sums.values()):
            raise ValueError('Source/checksum count mismatch')
        for index, value in enumerate(values):
            alias, separator, url = value.partition('::')
            if not separator:
                url = alias
            name = alias if separator else urlsplit(url).path.rstrip('/').rsplit('/', 1)[-1]
            if urlsplit(url).scheme.split('+')[0] in {'git', 'hg', 'svn', 'bzr', 'fossil'}:
                checks.append({'source': value, 'status': 'vcs_unverified', 'checksums_verified': []})
                continue
            if name not in files and name not in links:
                raise ValueError('Missing listed source input: ' + name)
            verified = []
            for field, entries in sums.items():
                expected = entries[index]
                if expected == 'SKIP':
                    continue
                algorithm = ALGORITHMS[field]
                if name not in files or files[name]['hashes'][algorithm] != expected.lower():
                    raise ValueError('Source checksum mismatch: ' + name)
                verified.append(algorithm)
            strong = set(verified) & {'sha256', 'sha512', 'blake2b'}
            checks.append({'source': value, 'path': name, 'status': 'verified' if strong else 'no_strong_checksum',
                           'checksums_verified': verified})
    return checks


def inspect_source(archive_path, item, zstd='zstd'):
    files, links, metadata = {}, {}, {}
    total = 0
    with tar_stream(archive_path, zstd) as archive:
        for number, member in enumerate(archive, 1):
            total += member.size
            if number > 100000 or member.size < 0 or total > MAX_UNPACKED:
                raise ValueError('Source member/size budget exceeded')
            path = member.name.removeprefix('./').rstrip('/')
            parts = PurePosixPath(path).parts
            if (not parts or parts[0] != item['base'] or '\\' in path or ':' in path
                    or any(ord(c) < 32 for c in path)
                    or any(p in {'', '.', '..'} for p in path.split('/'))):
                raise ValueError('Unsafe source package path')
            relative = '/'.join(parts[1:])
            if member.isdir():
                continue
            if not relative or relative in files or relative in links:
                raise ValueError('Duplicate or empty source file path')
            if member.issym() or member.islnk():
                if relative in {'PKGBUILD', '.SRCINFO'}:
                    raise ValueError('Recipe/metadata must be regular files')
                links[relative] = member.linkname
                continue
            if not member.isfile():
                raise ValueError('Unsupported source member type')
            is_metadata = relative in {'PKGBUILD', '.SRCINFO'}
            if is_metadata and member.size > MAX_METADATA:
                raise ValueError('Source metadata size budget exceeded')
            hashes = {algorithm: hashlib.new(algorithm, usedforsecurity=False) for algorithm in ALGORITHMS.values()}
            raw = bytearray()
            with archive.extractfile(member) as stream:
                for chunk in iter(lambda: stream.read(1024 * 1024), b''):
                    for digest in hashes.values():
                        digest.update(chunk)
                    if is_metadata:
                        raw.extend(chunk)
            files[relative] = {'size_bytes': member.size, 'hashes': {name: h.hexdigest() for name, h in hashes.items()}}
            if is_metadata:
                metadata[relative] = bytes(raw)
    if set(metadata) != {'PKGBUILD', '.SRCINFO'}:
        raise ValueError('Missing PKGBUILD or SRCINFO')
    if files['PKGBUILD']['hashes']['sha256'] != item['pkgbuild_sha256']:
        raise ValueError('Recipe differs from binary BUILDINFO')
    # Only the pkgbase section describes source inputs. Package-specific
    # sections below the first pkgname do not redefine the source plan.
    lines = []
    for line in metadata['.SRCINFO'].decode('utf-8').splitlines():
        if line.strip().startswith('pkgname = '):
            break
        lines.append(line)
    srcinfo = fields('\n'.join(lines).encode())
    version = single(srcinfo, 'pkgver') + '-' + single(srcinfo, 'pkgrel')
    if srcinfo.get('epoch', ['0']) != ['0']:
        version = single(srcinfo, 'epoch') + '~' + version
    if single(srcinfo, 'pkgbase') != item['base'] or version != item['version']:
        raise ValueError('SRCINFO identity differs from binary inventory')
    checks = verify_inputs(srcinfo, files, links)
    if item['base'] == 'mingw-w64-winpthreads':
        lock = load_vcs_lock(Path(__file__).resolve().parent.parent /
                             'packaging/windows/winpthreads-vcs-lock.json')
        evidence = verify_git_archive(archive_path, item, files, srcinfo, checks, lock, zstd)
        record = next(entry for entry in checks if entry['source'] == lock['vcs_source'])
        record.update(status='verified', checksums_verified=['sha256'], verification=evidence)
    if item['base'] in SIGNED_PACKAGES:
        key_root = Path(__file__).resolve().parent.parent / 'packaging/windows'
        entry = load_signature_lock(key_root / 'source-signature-lock.json')[item['base']]
        evidence = verify_signature(archive_path, item, files, srcinfo, checks, entry,
                                    key_root=key_root, zstd=zstd)
        record = next(check for check in checks if check['source'] == entry['signature_source'])
        record.update(status='verified', signature_verified=True, verification=evidence)
    recorded = {path: {'size_bytes': record['size_bytes'], 'sha256': record['hashes']['sha256']}
                for path, record in files.items()}
    return {'files': recorded, 'archive_links': links, 'source_checks': checks,
            'unverified_inputs': [entry for entry in checks if entry['status'] != 'verified']}, metadata


def collect(manifest, output, reuse_directory=None, zstd='zstd'):
    manifest, output = Path(manifest), Path(output)
    if manifest.stat().st_size > 16 * 1024 * 1024:
        raise ValueError('Native inventory size budget exceeded')
    raw = manifest.read_bytes()
    plan = source_plan(json.loads(raw))
    output.mkdir(parents=True, exist_ok=False)
    marker = output / 'INCOMPLETE'
    marker.write_text('Source collection incomplete; do not distribute.\n')
    (output / 'native-build.input.json').write_bytes(raw)
    (output / 'archives').mkdir()
    (output / 'pending').mkdir()
    total, results = 0, []
    for item in plan:
        filename = item['base'] + '-' + item['version'] + '.src.tar.zst'
        pending = output / 'pending' / filename
        url = 'https://repo.msys2.org/mingw/sources/' + filename
        limit = min(MAX_ARCHIVE, MAX_TOTAL - total)
        if limit <= 0:
            raise ValueError('Combined archive budget exceeded')
        reusable = Path(reuse_directory) / filename if reuse_directory else None
        if reusable and reusable.is_file():
            if reusable.is_symlink() or reusable.stat().st_size > limit:
                raise ValueError('Invalid reusable source archive')
            shutil.copyfile(reusable, pending)
        else:
            download.fetch(url, pending, limit)
        total += pending.stat().st_size
        if pending.stat().st_size > limit:
            raise ValueError('Source archive size budget exceeded')
        inspected, metadata = inspect_source(pending, item, zstd)
        archive = output / 'archives' / filename
        pending.rename(archive)
        directory = output / 'recipes' / (item['base'] + '-' + item['version'])
        directory.mkdir(parents=True)
        for name, data in metadata.items():
            (directory / name).write_bytes(data)
        results.append(dict(item, url=url, archive=archive.relative_to(output).as_posix(),
                            archive_sha256=sha(archive), size_bytes=archive.stat().st_size,
                            recipe_directory=directory.relative_to(output).as_posix(), **inspected))
        print('Matched recipe and source inputs:', item['base'], item['version'],
              'unverified inputs:', len(inspected['unverified_inputs']), flush=True)
    result = {'schema': 1, 'distribution_ready': False, 'corresponding_sources_complete': False,
              'input_manifest_sha256': hashlib.sha256(raw).hexdigest(), 'archives': results,
              'scope': 'MSYS2 source archives with binary-bound recipe hashes, declared checksums, pinned Git and offline detached-signature verification; no recipe execution.',
              'open_items': ['External signer identity/trust and source completeness review',
                             'Build environment, embedded-component mapping and applicable licenses',
                             'Rebuild/replacement instructions and distribution assembly']}
    (output / 'source-materials.json').write_text(json.dumps(result, indent=2) + '\n')
    marker.unlink()
    return result


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('native_build_json')
    parser.add_argument('new_output_directory')
    parser.add_argument('--reuse-directory')
    parser.add_argument('--zstd', default='zstd')
    args = parser.parse_args()
    collect(args.native_build_json, args.new_output_directory, args.reuse_directory, args.zstd)
