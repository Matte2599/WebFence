#!/usr/bin/env python3
"""Build a development ZIP from an existing WebFence UCRT64 executable."""
import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import posixpath
import re
import shutil
import subprocess
import sys
import tempfile
from msys2_binary_metadata import inspect, load_binary_lock, sha, verify_binary_lock
from verify_windows_binary_signatures import load_lock as load_signature_lock, verify_signature as verify_binary_signature
from attach_windows_sources import attach
from collect_windows_sources import collect


def run(*args):
    return subprocess.check_output([str(a) for a in args], text=True).strip()


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def notice_relative(path):
    """Select installed notice and attribution sidecars, including ICU's versioned share directory."""
    location = PurePosixPath(path)
    if (path.endswith("/") or "\\" in path or ".." in location.parts
            or not path.startswith("/ucrt64/share/")):
        return None
    relative = location.relative_to("/ucrt64/share")
    if relative.parts[0] == "licenses" or re.fullmatch(
            r"(?:licen[cs]e|copying|copyright|notice|authors)(?:[._-].*)?", location.name, flags=re.I):
        return relative
    if relative.parts[:3] == ("qt6", "wayland", "protocols") and (
            location.name in {"qt_attribution.json", "REUSE.toml", "README"}
            or re.fullmatch(r"(?:[A-Za-z0-9-]+_)?licen[cs]e(?:[._-].*)?", location.name, flags=re.I)
            or re.fullmatch(r"(?:LGPL|GPL|AGPL)-[A-Za-z0-9.+-]+\.txt", location.name, flags=re.I)):
        return relative
    return None


def qt_license_references(notice_sources):
    """Require every selected Qt attribution to retain its referenced license text."""
    root = 'ucrt64/share/qt6/wayland/protocols/'
    count = 0
    for member, source in notice_sources.items():
        if not (member.startswith(root) and member.endswith('/qt_attribution.json')):
            continue
        if source.stat().st_size > 1024 * 1024:
            raise ValueError('Qt attribution metadata exceeds size budget')
        data = json.loads(source.read_text(encoding='utf-8'))
        entries = data if isinstance(data, list) else [data]
        if not entries or any(not isinstance(entry, dict) for entry in entries):
            raise ValueError('Invalid Qt attribution metadata: ' + member)
        for entry in entries:
            references = []
            for field in ('LicenseFile', 'LicenseFiles'):
                if field not in entry:
                    continue
                value = entry[field]
                references.extend(value if isinstance(value, list) else [value])
            if not references:
                raise ValueError('Qt attribution has no license text reference: ' + member)
            for reference in references:
                if (not isinstance(reference, str) or not 1 <= len(reference) <= 512
                        or reference.startswith('/') or '\\' in reference or ':' in reference
                        or any(ord(char) < 32 for char in reference)):
                    raise ValueError('Unsafe Qt license text reference: ' + member)
                target = posixpath.normpath(posixpath.join(posixpath.dirname(member), reference))
                if not target.startswith(root) or target not in notice_sources:
                    raise ValueError('Missing selected Qt license text: ' + member + ' -> ' + reference)
                count += 1
    return count


def package(prefix, executable, source_materials=None, collect_sources=None):
    if sys.platform != "win32":
        raise ValueError("Native Windows UCRT64 Python required")
    if source_materials and collect_sources:
        raise ValueError('Select existing source materials or new collection, not both')
    prefix, executable = Path(prefix).resolve(), Path(executable).resolve()
    repo = Path(__file__).resolve().parent.parent
    binary_lock_path = repo / 'packaging/windows/msys2-binary-lock.json'
    binary_lock = load_binary_lock(binary_lock_path)
    signature_lock_path = repo / 'packaging/windows/msys2-binary-signature-lock.json'
    signing_key, signature_lock = load_signature_lock(signature_lock_path, binary_lock)
    dist = repo / "dist"
    dist.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix=".webfence-windows-", dir=dist) as temporary:
        stage = Path(temporary)
        bundle = stage / "WebFence"
        bundle.mkdir()
        shutil.copyfile(executable, bundle / "webfence.exe")
        subprocess.run([str(prefix / "bin/windeployqt6.exe"), "--release", "--no-translations",
                        "--no-system-d3d-compiler", "--no-opengl-sw", str(bundle / "webfence.exe")], check=True)
        sources = {}
        for source in prefix.rglob("*.dll"):
            sources.setdefault(source.name.lower(), []).append(source)

        def source_for(name):
            matches = sources.get(name.lower(), [])
            if len(matches) != 1:
                raise ValueError("Missing or ambiguous UCRT64 DLL: " + name)
            return matches[0]

        (bundle / "platforms").mkdir(exist_ok=True)
        shutil.copyfile(source_for("qoffscreen.dll"), bundle / "platforms/qoffscreen.dll")
        (bundle / "qt.conf").write_text("[Paths]\nPrefix=.\nPlugins=.\n", encoding="utf-8")
        queue = [bundle / "webfence.exe", *bundle.rglob("*.dll")]
        seen, system_imports, pe_imports = set(), set(), {}
        system = Path(os.environ["SystemRoot"]) / "System32"
        while queue:
            binary = queue.pop()
            if binary in seen:
                continue
            seen.add(binary)
            description = run(prefix / "bin/objdump.exe", "-p", binary)
            if "file format pei-x86-64" not in description:
                raise ValueError("Expected an x86-64 PE file: " + str(binary))
            if binary.name == "webfence.exe" and "(Windows GUI)" not in description:
                raise ValueError("Build the desktop executable with -ldflags=-H=windowsgui")
            imports = re.findall(r"DLL Name:\s*(\S+)", description)
            for name in imports:
                if not re.fullmatch(r"[a-zA-Z0-9_.+-]+\.dll", name, flags=re.I):
                    raise ValueError("Invalid imported DLL name")
                lower = name.lower()
                if lower.startswith(("api-ms-win-", "ext-ms-win-")) or (system / name).is_file():
                    system_imports.add(lower)
                    continue
                target = bundle / name
                if not target.exists():
                    shutil.copyfile(source_for(name), target)
                queue.append(target)
            pe_imports[binary.relative_to(bundle).as_posix()] = sorted({name.lower() for name in imports})

        notices = bundle / "notices"
        subprocess.run([sys.executable, str(repo / "scripts/package-go-notices.py"),
                        str(bundle / "webfence.exe"), str(notices / "go")], check=True)
        owners, files, required, required_notices = {}, [], {}, {}
        for binary in sorted(bundle.rglob("*.dll")):
            source = source_for(binary.name)
            owner = run("pacman", "-Qoq", run("cygpath", "-u", source))
            if not re.fullmatch(r'mingw-w64-ucrt-x86_64-[a-z0-9+_.-]+', owner):
                raise ValueError('Expected a UCRT64 DLL owner')
            if owner not in owners:
                metadata = run("pacman", "-Qi", owner)
                identity = run('pacman', '-Q', owner).split()
                if (len(identity) != 2 or identity[0] != owner
                        or not re.fullmatch(r'[A-Za-z0-9._+~-]+', identity[1])):
                    raise ValueError('Invalid installed UCRT64 package version')
                licenses, license_records, notice_sources = [], [], {}
                for line in run("pacman", "-Ql", owner).splitlines():
                    _, path = line.split(" ", 1)
                    relative = notice_relative(path)
                    if relative is not None:
                        local = Path(run("cygpath", "-w", path))
                        if local.is_file():
                            target = notices / "native" / owner / relative
                            if target.exists():
                                raise ValueError('Duplicate package notice: ' + path)
                            target.parent.mkdir(parents=True, exist_ok=True)
                            shutil.copyfile(local, target)
                            published = target.relative_to(bundle).as_posix()
                            member = local.relative_to(prefix.parent).as_posix()
                            notice_hash = sha(target)
                            required_notices.setdefault(owner, {})[member] = notice_hash
                            notice_sources[member] = target
                            licenses.append(published)
                            license_records.append({'path': published, 'sha256': notice_hash,
                                                    'binary_package_member': member})
                if not licenses:
                    raise ValueError("Missing packaged license notices for " + owner)
                owners[owner] = {"pacman_metadata": metadata, "version": identity[1],
                                 "license_files": licenses, "license_file_records": license_records,
                                 "qt_license_reference_count": qt_license_references(notice_sources)}
            member = source.relative_to(prefix.parent).as_posix()
            required.setdefault(owner, {})[member] = sha(source)
            relative = binary.relative_to(bundle).as_posix()
            files.append({"path": relative, "sha256": digest(binary),
                          "source_sha256": required[owner][member], "package": owner,
                          "binary_package_member": member, "imports": pe_imports[relative]})
        if set(pe_imports) != {'webfence.exe', *(item['path'] for item in files)}:
            raise ValueError('PE import inventory differs from packaged executable and DLLs')
        cache = Path(run('cygpath', '-w', '/var/cache/pacman/pkg'))
        for owner, record in owners.items():
            candidates = [p for p in cache.glob(owner + '-' + record['version'] + '-*.pkg.tar.*')
                          if p.name.endswith(('.pkg.tar.zst', '.pkg.tar.xz', '.pkg.tar.gz'))]
            if len(candidates) != 1:
                raise ValueError('Retain exactly one cached binary package for ' + owner + ' ' + record['version'])
            binary_record, metadata = inspect(candidates[0], owner, record['version'], required[owner],
                                               str(prefix / 'bin/zstd.exe'),
                                               required_notices=required_notices[owner],
                                               notice_selector=lambda member: notice_relative('/' + member) is not None)
            binary_record['published_sha256_source'] = verify_binary_lock(
                binary_lock, owner, record['version'], binary_record['sha256'])
            binary_record['signature_verification'] = verify_binary_signature(
                candidates[0], signature_lock[owner], signing_key,
                key_root=repo / 'packaging/windows')
            destination = notices / 'native' / owner / 'build'
            destination.mkdir()
            binary_record['metadata_files'] = []
            for name, raw in metadata.items():
                target = destination / name
                target.write_bytes(raw)
                binary_record['metadata_files'].append({'path': target.relative_to(bundle).as_posix(), 'sha256': sha(target)})
            signature_target = destination / 'package.sig'
            shutil.copyfile(repo / 'packaging/windows' / signature_lock[owner]['signature_file'],
                            signature_target)
            if sha(signature_target) != signature_lock[owner]['signature_sha256']:
                raise ValueError('Copied Windows binary package signature differs from lock')
            binary_record['signature_file'] = signature_target.relative_to(bundle).as_posix()
            record['binary_package'] = binary_record
            print(f"Verified {owner} {record['version']}: {len(required[owner])} DLLs, "
                  f"{binary_record['verified_notice_count']} notices; source {binary_record['source_package']}; "
                  f"PKGBUILD {binary_record['pkgbuild_sha256']}")
        if set(owners) != set(binary_lock):
            raise ValueError('Windows binary package set differs from reviewed checksum lock')
        published_lock = notices / 'native/msys2-binary-lock.json'
        shutil.copyfile(binary_lock_path, published_lock)
        published_signature_lock = notices / 'native/msys2-binary-signature-lock.json'
        shutil.copyfile(signature_lock_path, published_signature_lock)
        published_signing_key = notices / 'native/msys2-binary-signing-key.asc'
        shutil.copyfile(repo / 'packaging/windows' / signing_key['public_key_file'],
                        published_signing_key)
        if sha(published_signing_key) != signing_key['public_key_sha256']:
            raise ValueError('Copied Windows binary signing key differs from lock')
        (bundle / "native-build.json").write_text(json.dumps({"schema": 1, "files": files,
            "distribution_ready": False,
            "binary_lock_sha256": sha(published_lock),
            "binary_signature_lock_sha256": sha(published_signature_lock),
            "packages": owners, "system_imports": sorted(system_imports),
            "executable_imports": pe_imports['webfence.exe'],
            "scope": "PE import closure and recorded static imports, installed notice/attribution sidecars and DLL/build metadata matched to cached MSYS2 archives, reviewed SHA-256 checksums and offline package signatures; not full source compliance, independent signer identity or dynamic-load coverage"}, indent=2) + "\n")
        if collect_sources:
            collect(bundle / 'native-build.json', collect_sources, zstd=str(prefix / 'bin/zstd.exe'))
            source_materials = collect_sources
        if source_materials:
            result = attach(bundle / 'native-build.json', source_materials, bundle / 'msys2-sources',
                            zstd=str(prefix / 'bin/zstd.exe'))
            print(f"Attached Windows sources: {result['archive_count']} archives, {result['archive_bytes']} bytes")
        subprocess.run([sys.executable, str(repo / "scripts/package-project-docs.py"), str(bundle)], check=True)
        archive = shutil.make_archive(str(stage / "webfence-windows-amd64"), "zip", stage, "WebFence")
        final = dist / "webfence-windows-amd64.zip"
        os.replace(archive, final)
        print(str(final), digest(final))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('ucrt64_prefix')
    parser.add_argument('webfence_exe')
    selection = parser.add_mutually_exclusive_group()
    selection.add_argument('--source-materials', help='Attach an existing verified source collection; no download')
    selection.add_argument('--collect-sources', help='Download into this new directory, verify and attach sources')
    args = parser.parse_args()
    package(args.ucrt64_prefix, args.webfence_exe, args.source_materials, args.collect_sources)
