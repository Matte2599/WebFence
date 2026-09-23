#!/usr/bin/env python3
"""Build a development ZIP from an existing WebFence UCRT64 executable."""
import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import shutil
import subprocess
import sys
import tempfile
from msys2_binary_metadata import inspect, sha
from attach_windows_sources import attach
from collect_windows_sources import collect


def run(*args):
    return subprocess.check_output([str(a) for a in args], text=True).strip()


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def notice_relative(path):
    """Select package-owned notices, including ICU's versioned share directory."""
    location = PurePosixPath(path)
    if path.endswith("/") or ".." in location.parts or not path.startswith("/ucrt64/share/"):
        return None
    relative = location.relative_to("/ucrt64/share")
    if relative.parts[0] == "licenses" or re.fullmatch(
            r"(?:licen[cs]e|copying|copyright|notice)(?:[._-].*)?", location.name, flags=re.I):
        return relative
    return None


def package(prefix, executable, source_materials=None, collect_sources=None):
    if sys.platform != "win32":
        raise ValueError("Native Windows UCRT64 Python required")
    if source_materials and collect_sources:
        raise ValueError('Select existing source materials or new collection, not both')
    prefix, executable = Path(prefix).resolve(), Path(executable).resolve()
    repo = Path(__file__).resolve().parent.parent
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
        seen, system_imports = set(), set()
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
            for name in re.findall(r"DLL Name:\s*(\S+)", description):
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
                licenses, license_records = [], []
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
                            licenses.append(published)
                            license_records.append({'path': published, 'sha256': notice_hash,
                                                    'binary_package_member': member})
                if not licenses:
                    raise ValueError("Missing packaged license notices for " + owner)
                owners[owner] = {"pacman_metadata": metadata, "version": identity[1],
                                 "license_files": licenses, "license_file_records": license_records}
            member = source.relative_to(prefix.parent).as_posix()
            required.setdefault(owner, {})[member] = sha(source)
            files.append({"path": binary.relative_to(bundle).as_posix(), "sha256": digest(binary),
                          "source_sha256": required[owner][member], "package": owner, "binary_package_member": member})
        cache = Path(run('cygpath', '-w', '/var/cache/pacman/pkg'))
        for owner, record in owners.items():
            candidates = [p for p in cache.glob(owner + '-' + record['version'] + '-*.pkg.tar.*')
                          if p.name.endswith(('.pkg.tar.zst', '.pkg.tar.xz', '.pkg.tar.gz'))]
            if len(candidates) != 1:
                raise ValueError('Retain exactly one cached binary package for ' + owner + ' ' + record['version'])
            binary_record, metadata = inspect(candidates[0], owner, record['version'], required[owner],
                                               str(prefix / 'bin/zstd.exe'),
                                               required_notices=required_notices[owner])
            destination = notices / 'native' / owner / 'build'
            destination.mkdir()
            binary_record['metadata_files'] = []
            for name, raw in metadata.items():
                target = destination / name
                target.write_bytes(raw)
                binary_record['metadata_files'].append({'path': target.relative_to(bundle).as_posix(), 'sha256': sha(target)})
            record['binary_package'] = binary_record
            print(f"Verified {owner} {record['version']}: {len(required[owner])} DLLs, "
                  f"{binary_record['verified_notice_count']} notices; source {binary_record['source_package']}; "
                  f"PKGBUILD {binary_record['pkgbuild_sha256']}")
        (bundle / "native-build.json").write_text(json.dumps({"schema": 1, "files": files,
            "distribution_ready": False,
            "packages": owners, "system_imports": sorted(system_imports),
            "scope": "PE import closure, installed notices and DLL/build metadata matched to cached MSYS2 archives; not full source compliance, signature verification or dynamic-load coverage"}, indent=2) + "\n")
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
