#!/usr/bin/env python3
"""Collect provenance/notices for a trusted staged Homebrew-based app, before signing."""
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess
import sys

MACHO = {b'\xcf\xfa\xed\xfe', b'\xfe\xed\xfa\xcf', b'\xce\xfa\xed\xfe',
         b'\xfe\xed\xfa\xce', b'\xca\xfe\xba\xbe', b'\xbe\xba\xfe\xca',
         b'\xca\xfe\xba\xbf', b'\xbf\xba\xfe\xca'}


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def is_macho(path):
    with path.open('rb') as stream:
        return stream.read(4) in MACHO


def arm_uuid(path):
    output = subprocess.check_output(['dwarfdump', '--uuid', str(path)], text=True)
    entries = re.findall(r'UUID: ([0-9A-Fa-f-]+) \(([^)]+)\)', output)
    if len(entries) != 1 or entries[0][1] != 'arm64':
        raise ValueError('Expected one arm64 Mach-O slice: ' + str(path))
    return entries[0][0].upper()


def collect(bundle, qt_prefix):
    if sys.platform != 'darwin':
        raise ValueError('macOS collector requires native Apple developer tools')
    bundle, qt_keg = Path(bundle).resolve(), Path(qt_prefix).resolve()
    repo = Path(__file__).resolve().parent.parent
    if qt_keg.parent.name != 'qtbase':
        raise ValueError('Expected an installed qtbase keg')
    receipt = json.loads((qt_keg / 'INSTALL_RECEIPT.json').read_text())
    cellar = qt_keg.parent.parent
    kegs = {qt_keg}
    for dependency in receipt['runtime_dependencies']:
        name, version = dependency['full_name'].split('/')[-1], dependency['pkg_version']
        if not re.fullmatch(r'[A-Za-z0-9@+._-]+', name) or not re.fullmatch(r'[A-Za-z0-9+._-]+', version):
            raise ValueError('Invalid Homebrew dependency identity')
        # Receipt closure also lists unused platforms/dependencies and can
        # predate a runtime upgrade. Match actual copied UUIDs against installed kegs.
        formula = cellar / name
        if formula.is_dir():
            kegs.update(keg.resolve() for keg in formula.iterdir() if (keg / 'INSTALL_RECEIPT.json').is_file())
    binaries = sorted(p for p in bundle.rglob('*') if p.is_file() and not p.is_symlink() and is_macho(p))
    executable = bundle / 'Contents/MacOS/webfence'
    cocoa = bundle / 'Contents/PlugIns/platforms/libqcocoa.dylib'
    if executable not in binaries or cocoa not in binaries:
        raise ValueError('Expected WebFence executable and rebuilt Cocoa plugin')
    names = {p.name for p in binaries if p not in {executable, cocoa}}
    candidates = {}
    for keg in sorted(kegs):
        for entry in keg.rglob('*'):
            if entry.name not in names or not entry.is_file():
                continue
            source = entry.resolve()
            if not source.is_relative_to(keg) or not is_macho(source):
                continue
            candidates.setdefault((entry.name, arm_uuid(source)), set()).add((keg, source))
    output = bundle / 'Contents/Resources/notices'
    output.mkdir(parents=True, exist_ok=False)
    packages, files = {}, []
    for binary in binaries:
        uuid = arm_uuid(binary)
        item = {'path': binary.relative_to(bundle).as_posix(), 'uuid': uuid,
                'sha256_before_final_signing': sha(binary)}
        if binary == executable:
            item['origin'] = 'webfence-go-build'
        elif binary == cocoa:
            item['origin'] = 'rebuilt-qt-cocoa'
            item['materials'] = 'Contents/Resources/notices/scripts/qt-cocoa'
        else:
            matches = candidates.get((binary.name, uuid), set())
            if len(matches) != 1:
                raise ValueError('Unmapped or ambiguous native binary: ' + str(binary))
            keg, source = next(iter(matches))
            key = keg.parent.name + '@' + keg.name
            item.update(origin='homebrew', package=key, source_path=source.relative_to(keg).as_posix(), source_sha256=sha(source))
            if key not in packages:
                destination = output / 'homebrew' / key
                destination.mkdir(parents=True)
                notices = []
                metadata = []
                for entry in keg.rglob('*'):
                    if not entry.is_file() or not entry.resolve().is_relative_to(keg):
                        continue
                    relative = entry.relative_to(keg)
                    is_notice = bool(re.fullmatch(r'(?:licen[cs]e|copying|copyright|notice)(?:[._-].*)?', entry.name, re.I))
                    is_metadata = entry.name in {'INSTALL_RECEIPT.json', 'sbom.spdx.json'} or relative.parts[0] == '.brew' or relative.as_posix().startswith('share/qt/sbom/')
                    if not (is_notice or is_metadata):
                        continue
                    target = destination / relative
                    target.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copyfile(entry, target)
                    (notices if is_notice else metadata).append(relative.as_posix())
                sbom = json.loads((keg / 'sbom.spdx.json').read_text())
                archives = [p for p in sbom['packages'] if p['SPDXID'].startswith('SPDXRef-Archive-')]
                packages[key] = {'formula': keg.parent.name, 'version': keg.name,
                    'notice_files': sorted(notices), 'metadata_files': sorted(metadata),
                    'notice_status': 'installed_only_requires_review' if notices else 'no_installed_notices',
                    'upstream_archives': archives,
                    'review': 'installed notices only; upstream/embedded components and corresponding sources require review'}
        files.append(item)
    subprocess.run([sys.executable, str(repo / 'scripts/package-go-notices.py'), str(executable), str(output / 'go')], check=True)
    go_record = output / 'go/go-build.json'
    record = json.loads(go_record.read_text())
    record['executable_hash_stage'] = 'before final bundle signing; not the final signed executable hash'
    go_record.write_text(json.dumps(record, indent=2) + '\n')
    shutil.copytree(repo / 'scripts/qt-cocoa', output / 'scripts/qt-cocoa')
    shutil.copyfile(repo / 'scripts/build-qt-cocoa.sh', output / 'scripts/build-qt-cocoa.sh')
    for name in ['LICENSE', 'README.md', 'README.en.md']:
        shutil.copyfile(repo / name, output / name)
    # Preserve linked documentation instead of shipping READMEs with missing paths.
    shutil.copytree(repo / 'DOCS', output / 'DOCS', ignore=shutil.ignore_patterns('.DS_Store'))
    for name in ['AGENTS.md', 'MEMORY.md', 'CONTRIBUTING.md', 'SECURITY.md']:
        shutil.copyfile(repo / name, output / name)
    archive_readme = output / 'experiments/qt/README.md'
    archive_readme.parent.mkdir(parents=True)
    shutil.copyfile(repo / 'experiments/qt/README.md', archive_readme)
    build = json.loads(subprocess.check_output(['go', 'version', '-m', '-json', str(executable)], text=True))
    provenance = {item['Key']: item['Value'] for item in build.get('Settings', []) if item['Key'] in
                  {'vcs.revision', 'vcs.time', 'vcs.modified', 'GOOS', 'GOARCH', 'CGO_CXXFLAGS'}}
    inventory = {'schema': 1, 'distribution_ready': False, 'webfence_build': provenance, 'scope': 'Mach-O origin matching by name/arm64/UUID and installed notices; not a complete product SBOM or distribution approval',
        'hash_stage': 'before final bundle signing; UUID is a provenance hint, not an integrity proof',
        'files': files, 'packages': packages,
        'open_items': ['Complete embedded-component attribution', 'Qt and GLib upstream notices beyond installed build-tool notices',
                       'Corresponding source archives, patches and replacement/rebuild materials', 'Qualified legal review and clean-machine trials']}
    (output / 'native-build.json').write_text(json.dumps(inventory, indent=2) + '\n')
    print(f'Collected {len(files)} Mach-O records and {len(packages)} Homebrew packages; distribution review remains open')


if __name__ == '__main__':
    if len(sys.argv) != 3:
        raise SystemExit('Usage: package-macos-notices.py STAGED_APP QT_PREFIX')
    collect(*sys.argv[1:])
