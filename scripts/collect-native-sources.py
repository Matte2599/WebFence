#!/usr/bin/env python3
"""Collect upstream archives/notices from trusted macOS packaging metadata.

No archive code is executed. This is a source-material sidecar, not a claim
that all corresponding sources, embedded attributions or legal duties are met.
"""
import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import posixpath
import re
import shutil
import subprocess
import tarfile
from urllib.parse import urlsplit

MAX_ARCHIVE = 128 * 1024 * 1024
MAX_TOTAL_ARCHIVES = 512 * 1024 * 1024
MAX_UNPACKED = 2 * 1024 * 1024 * 1024
MAX_NOTICE = 4 * 1024 * 1024
MAX_TOTAL_NOTICES = 32 * 1024 * 1024
MAX_MEMBERS = 100000
MAX_QT_REFERENCES = 4096
# References explicitly named by FreeType's root LICENSE.TXT. Keep whole files
# containing these notices; the source archive remains the canonical material.
FREETYPE_REFERENCES = {'docs/FTL.TXT', 'docs/GPLv2.TXT', 'src/bdf/README', 'src/pcf/README',
                       'src/base/fthash.c', 'include/freetype/internal/fthash.h',
                       'src/gzip/zlib.h', 'src/autofit/ft-hb-ft.c', 'src/autofit/ft-hb-decls.h',
                       'src/autofit/ft-hb-types.h', 'src/autofit/hb-script-list.h'}


def sha(path):
    digest = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def safe_path(value):
    path = PurePosixPath(value)
    if (not value or path.is_absolute() or '\\' in value or ':' in value
            or any(part in {'', '.', '..'} for part in value.split('/'))
            or any(ord(c) < 32 for c in value)):
        raise ValueError('Unsafe archive or package path')
    return path


def archive_plan(inventory):
    if inventory.get('schema') != 1 or not isinstance(inventory.get('packages'), dict):
        raise ValueError('Expected native-build schema 1')
    plan = []
    for package, data in sorted(inventory['packages'].items()):
        if not re.fullmatch(r'[A-Za-z0-9@+._-]+', package) or package in {'.', '..'}:
            raise ValueError('Invalid package identity')
        archives = data.get('upstream_archives', [])
        if not archives:
            raise ValueError('Missing upstream archive: ' + package)
        for archive in archives:
            url = archive.get('downloadLocation', '')
            parts = urlsplit(url)
            if (parts.scheme != 'https' or not parts.hostname or parts.username
                    or parts.password or parts.fragment or any(ord(c) < 32 for c in url)):
                raise ValueError('Expected a credential-free HTTPS source URL')
            hashes = [c.get('checksumValue', '') for c in archive.get('checksums', [])
                      if c.get('algorithm') == 'SHA256']
            if len(hashes) != 1 or not re.fullmatch(r'[0-9a-fA-F]{64}', hashes[0]):
                raise ValueError('Expected one SHA-256 source checksum')
            plan.append({'package': package, 'url': url, 'sha256': hashes[0].lower()})
    if not 1 <= len(plan) <= 32:
        raise ValueError('Expected 1–32 upstream archives')
    return plan


def fetch(url, destination, max_bytes):
    # Metadata is a trusted build input, never scanner/AI input. curl limits
    # redirects to HTTPS and bounds transfer size, stalls and total duration.
    version = subprocess.check_output(['curl', '--disable', '--version'], text=True).splitlines()[0]
    match = re.match(r'curl (\d+)\.(\d+)\.(\d+)', version)
    if not match or tuple(map(int, match.groups())) < (8, 4, 0):
        raise ValueError('curl 8.4+ required to bound transfers without Content-Length')
    candidates = [url]
    mirror_prefix = 'https://ftpmirror.gnu.org/gnu/'
    if url.startswith(mirror_prefix):
        candidates.append('https://ftp.gnu.org/gnu/' + url[len(mirror_prefix):])
    for index, candidate in enumerate(candidates):
        try:
            subprocess.run(['curl', '--disable', '--fail', '--location', '--silent', '--show-error',
                            '--proto', '=https', '--proto-redir', '=https', '--max-redirs', '5',
                            '--connect-timeout', '20', '--max-time', '180',
                            '--speed-limit', '1024', '--speed-time', '30',
                            '--max-filesize', str(max_bytes), '--output', str(destination), candidate],
                           check=True, timeout=190)
            return candidate
        except subprocess.CalledProcessError:
            if index + 1 == len(candidates):
                raise


def is_notice(path):
    return (bool(re.fullmatch(r'(?:licen[cs]e|copying|copyright|notice)(?:[._-].*)?', path.name, re.I))
            or 'LICENSES' in path.parts or path.name == 'qt_attribution.json')


def qt_license_references(archive):
    """Read Qt attribution references before streaming files in arbitrary order.

    Qt metadata contains literal newlines in strings, hence strict=False. This
    only relaxes JSON string parsing; reference paths still have strict bounds.
    Never resolve filesystem links or extract metadata as executable content.
    """
    references, seen = set(), set()
    unpacked = metadata_bytes = 0
    with tarfile.open(archive, mode='r|*') as source:
        for number, member in enumerate(source, 1):
            if number > MAX_MEMBERS or member.size < 0:
                raise ValueError('Archive entry budget exceeded')
            unpacked += member.size
            if unpacked > MAX_UNPACKED:
                raise ValueError('Uncompressed source budget exceeded')
            path = safe_path(member.name.rstrip('/'))
            if path.name != 'qt_attribution.json':
                continue
            if member.name in seen or not member.isfile() or len(path.parts) < 2:
                raise ValueError('Invalid or duplicate Qt attribution metadata')
            seen.add(member.name)
            metadata_bytes += member.size
            if member.size > MAX_NOTICE or metadata_bytes > MAX_TOTAL_NOTICES:
                raise ValueError('Qt attribution size budget exceeded')
            with source.extractfile(member) as stream:
                data = json.loads(stream.read(), strict=False)
            entries = data if isinstance(data, list) else [data]
            for entry in entries:
                if not isinstance(entry, dict):
                    raise ValueError('Expected Qt attribution object')
                for field in ('LicenseFile', 'LicenseFiles'):
                    values = entry.get(field, [])
                    values = [values] if isinstance(values, str) else values
                    if not isinstance(values, list):
                        raise ValueError('Expected Qt license file string or list')
                    for value in values:
                        if (not isinstance(value, str) or not value or value.startswith('/')
                                or '\\' in value or ':' in value or '//' in value
                                or any(ord(c) < 32 for c in value)):
                            raise ValueError('Unsafe Qt license reference')
                        target = safe_path(posixpath.normpath(posixpath.join(str(path.parent), value)))
                        if len(target.parts) < 2 or target.parts[0] != path.parts[0]:
                            raise ValueError('Qt license reference escapes archive root')
                        references.add((path.as_posix(), target.as_posix()))
                        if len(references) > MAX_QT_REFERENCES:
                            raise ValueError('Qt license reference budget exceeded')
    return [{'attribution': attribution, 'path': path} for attribution, path in sorted(references)]


def collect_notices(archive, destination, references=()):
    files, links, seen = [], [], set()
    references, found_references = set(references), set()
    qt_references = qt_license_references(archive)
    qt_paths = {item['path'] for item in qt_references}
    found_qt = set()
    unpacked = notice_bytes = 0
    # Stream rather than extractall: do not materialize links, devices, code or
    # arbitrary paths. The original archive preserves all source files.
    with tarfile.open(archive, mode='r|*') as source:
        for number, member in enumerate(source, 1):
            if number > MAX_MEMBERS or member.size < 0:
                raise ValueError('Archive entry budget exceeded')
            unpacked += member.size
            if unpacked > MAX_UNPACKED:
                raise ValueError('Uncompressed source budget exceeded')
            path = safe_path(member.name.rstrip('/'))
            relative = '/'.join(path.parts[1:])
            if not (is_notice(path) or relative in references or path.as_posix() in qt_paths) or member.isdir():
                continue
            if member.name in seen:
                raise ValueError('Duplicate notice path')
            seen.add(member.name)
            if member.issym() or member.islnk():
                links.append({'path': member.name, 'target': member.linkname,
                              'handling': 'retained only in original archive; never followed'})
                continue
            if not member.isfile():
                raise ValueError('Unsupported notice entry type')
            if relative in references:
                found_references.add(relative)
            if path.as_posix() in qt_paths:
                found_qt.add(path.as_posix())
            notice_bytes += member.size
            if member.size > MAX_NOTICE or notice_bytes > MAX_TOTAL_NOTICES:
                raise ValueError('Notice size budget exceeded')
            target = destination.joinpath(*path.parts)
            target.parent.mkdir(parents=True, exist_ok=True)
            with source.extractfile(member) as stream, target.open('xb') as output:
                shutil.copyfileobj(stream, output, 65536)
            files.append({'path': path.as_posix(), 'size_bytes': member.size, 'sha256': sha(target)})
    if not files:
        raise ValueError('No regular source notices found')
    if references != found_references:
        raise ValueError('Missing referenced source notices; review package changes')
    if qt_paths != found_qt:
        raise ValueError('Missing regular Qt license references; review package changes')
    result = {'files': files, 'archive_links': links, 'scope': 'source-tree notices, possibly a superset of shipped code'}
    if qt_references:
        result['qt_license_references'] = qt_references
    return result


def collect(manifest, output, reuse=()):
    manifest, output = Path(manifest), Path(output)
    if manifest.stat().st_size > 16 * 1024 * 1024:
        raise ValueError('Input manifest too large')
    raw = manifest.read_bytes()
    plan = archive_plan(json.loads(raw))
    available = {}
    for entry in reuse:
        path = Path(entry)
        if not path.is_file() or path.stat().st_size > MAX_ARCHIVE:
            raise ValueError('Invalid reusable archive')
        available[sha(path)] = path
    output.mkdir(parents=True, exist_ok=False)
    marker = output / 'INCOMPLETE'
    marker.write_text('Collection is incomplete; do not distribute this sidecar.\n')
    (output / 'native-build.input.json').write_bytes(raw)
    (output / 'archives').mkdir()
    results, downloaded, acquired_from, total = [], {}, {}, 0
    for item in plan:
        digest = item['sha256']
        archive = output / 'archives' / (digest + '.archive')
        if digest not in downloaded:
            partial = archive.with_suffix('.part')
            limit = min(MAX_ARCHIVE, MAX_TOTAL_ARCHIVES - total)
            if limit <= 0:
                raise ValueError('Total source archive budget exceeded')
            if digest in available:
                if available[digest].stat().st_size > limit:
                    raise ValueError('Reusable archive exceeds source budget')
                shutil.copyfile(available[digest], partial)
            else:
                acquired_from[digest] = fetch(item['url'], partial, limit)
            size = partial.stat().st_size
            total += size
            if size > MAX_ARCHIVE or total > MAX_TOTAL_ARCHIVES or sha(partial) != digest:
                raise ValueError('Source size/checksum verification failed: ' + item['package'])
            partial.rename(archive)
            downloaded[digest] = size
        notice_root = output / 'source-notices' / item['package'] / digest
        notice_root.mkdir(parents=True, exist_ok=False)
        references = FREETYPE_REFERENCES if item['package'].startswith('freetype@') else ()
        notices = collect_notices(archive, notice_root, references)
        result = dict(item, archive=archive.relative_to(output).as_posix(),
                      size_bytes=downloaded[digest],
                      notice_root=notice_root.relative_to(output).as_posix(), **notices)
        if digest in acquired_from:
            result['acquisition_url'] = acquired_from[digest]
        results.append(result)
        print('Verified source and notices:', item['package'], flush=True)
    result = {'schema': 1, 'distribution_ready': False, 'corresponding_sources_complete': False,
              'input_manifest_sha256': hashlib.sha256(raw).hexdigest(), 'archives': results,
              'scope': 'Exact upstream archives and source-tree notices for the supplied native inventory. Not a product SBOM or complete corresponding sources.',
              'open_items': ['Package-manager patches/resources and exact build environment',
                             'Embedded-component mapping to the actual binary and license review',
                             'Replacement/rebuild instructions and distribution review']}
    (output / 'source-materials.json').write_text(json.dumps(result, indent=2) + '\n')
    marker.unlink()
    return result


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('native_build_json')
    parser.add_argument('output_directory')
    parser.add_argument('--reuse-archive', action='append', default=[])
    args = parser.parse_args()
    collect(args.native_build_json, args.output_directory, args.reuse_archive)
