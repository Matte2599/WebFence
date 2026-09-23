#!/usr/bin/env python3
"""Assemble available native source materials beside a trusted signed macOS app.

No downloads, recipe execution or app modification. This is a review package,
not certification of complete corresponding sources or distribution approval.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import sys
import tempfile
import zipfile


def module(name, filename):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(filename))
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


attachment = module('source_attachment', 'attach-native-sources.py')
supplements = module('source_supplements', 'native_supplements.py')
sha = attachment.sources.sha
MAX_NOTICE_BYTES = 128 * 1024 * 1024
MAX_FILES = 10000


def copy_notices(source, target):
    total = count = 0
    for path in sorted(source.rglob('*')):
        relative = path.relative_to(source)
        # Regenerate these from their verified inputs, not loose notice files.
        if relative.parts[0] in {'upstream-source', 'homebrew-supplements'}:
            continue
        mode = path.lstat().st_mode
        if stat.S_ISDIR(mode):
            continue
        if not stat.S_ISREG(mode):
            raise ValueError('Packaged notices must contain only regular files and directories')
        count += 1
        total += path.stat().st_size
        if count > MAX_FILES or total > MAX_NOTICE_BYTES:
            raise ValueError('Packaged notice budget exceeded')
        destination = target / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(path, destination)


def assemble(bundle, materials, output):
    bundle, materials, output = Path(bundle).resolve(), Path(materials).resolve(), Path(output).absolute()
    if output.exists() or output.is_symlink():
        raise FileExistsError('Refusing to replace an existing source package')
    if any(output.resolve().is_relative_to(root) for root in (bundle, materials)):
        raise ValueError('Output must be outside the bundle and source collection')
    # Validate the whole app before trusting its packaged recipes and metadata.
    subprocess.run(['codesign', '--verify', '--deep', '--strict', str(bundle)], check=True, timeout=120)
    notices = bundle / 'Contents/Resources/notices'
    inventory_path = attachment.regular_file(notices, 'native-build.json')
    _, inventory = attachment.read_json(inventory_path)
    if inventory.get('distribution_ready') is not False:
        raise ValueError('Expected a development native inventory')
    plan = attachment.regular_file(notices, 'homebrew-supplements/plan.json')
    required = ('LICENSE', 'README.md', 'README.en.md', 'DOCS/ROADMAP.md',
                'scripts/build-qt-cocoa.sh', 'scripts/qt-cocoa/accessibility.patch')
    for name in required:
        attachment.regular_file(notices, name)
    signed_files = []
    for item in inventory.get('files', []):
        path = attachment.regular_file(bundle, item['path'])
        signed_files.append({'path': item['path'], 'sha256': sha(path)})
    if (not signed_files or len({f['path'] for f in signed_files}) != len(signed_files)
            or 'Contents/MacOS/webfence' not in {f['path'] for f in signed_files}):
        raise ValueError('Missing or duplicate bundle binary identity')
    # Temporary output shares the destination filesystem, enabling an atomic
    # no-clobber hard-link publication. Inputs must have no concurrent writers.
    with tempfile.TemporaryDirectory(prefix='.webfence-source-package-', dir=output.parent) as temporary:
        stage = Path(temporary) / 'WebFence-native-sources'
        packaged = stage / 'notices'
        packaged.mkdir(parents=True)
        copy_notices(notices, packaged)
        source_record = attachment.attach(packaged / 'native-build.json', materials, packaged / 'upstream-source')
        supplement_record = supplements.collect(packaged / 'native-build.json', plan,
                                                packaged / 'homebrew-supplements',
                                                reuse_directory=notices / 'homebrew-supplements')
        _, collected = attachment.read_json(materials / 'source-materials.json')
        for record in collected['archives']:
            original = attachment.regular_file(materials, record['archive'])
            destination = packaged / 'upstream-source' / record['archive']
            if destination.exists():
                continue  # Identical archives shared by multiple package records.
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(original, destination)
            if destination.stat().st_size != record['size_bytes'] or sha(destination) != record['sha256']:
                raise ValueError('Source archive changed while copying')
        # The attachment's archive paths now resolve inside this ZIP.
        source_record['archives_location'] = 'Included here; paths in source-materials.json are relative to this upstream-source directory.'
        (packaged / 'upstream-source/attachment.json').write_text(json.dumps(source_record, indent=2) + '\n')
        files = []
        for path in sorted(packaged.rglob('*')):
            if path.is_file():
                files.append({'path': path.relative_to(stage).as_posix(),
                              'size_bytes': path.stat().st_size, 'sha256': sha(path)})
        record = {'schema': 1, 'distribution_ready': False, 'corresponding_sources_complete': False,
                  'scope': 'Available native upstream archives, reviewed supplements and packaged notices; not full WebFence/Go sources or a complete product SBOM.',
                  'native_inventory_sha256': sha(inventory_path), 'webfence_build': inventory.get('webfence_build'),
                  'signed_bundle_files': signed_files, 'signature_check': 'codesign --verify --deep --strict passed; not publisher authentication',
                  'archive_count': source_record['archive_count'], 'archive_bytes': source_record['archive_bytes'],
                  'notice_count': source_record['notice_count'], 'supplement_file_count': supplement_record['file_count'],
                  'files': files,
                  'open_items': ['Remaining recipe resources and exact build environments',
                                 'Embedded component mapping and legal review',
                                 'Complete WebFence/Go source delivery and cross-platform replacement procedures']}
        (stage / 'source-package.json').write_text(json.dumps(record, indent=2) + '\n')
        archive = Path(temporary) / 'materials.zip'
        with zipfile.ZipFile(archive, 'w', compression=zipfile.ZIP_DEFLATED) as zipped:
            for path in sorted(stage.rglob('*')):
                if path.is_file():
                    zipped.write(path, path.relative_to(stage.parent).as_posix(),
                                 compress_type=zipfile.ZIP_STORED if path.suffix == '.archive' else zipfile.ZIP_DEFLATED)
        # Read back exact bytes before publication, not only ZIP CRCs.
        with zipfile.ZipFile(archive) as zipped:
            if len(zipped.namelist()) != len(files) + 1 or zipped.testzip() is not None:
                raise ValueError('Source ZIP structure/CRC verification failed')
            for item in files:
                with zipped.open('WebFence-native-sources/' + item['path']) as source:
                    digest = hashlib.sha256()
                    for chunk in iter(lambda: source.read(1024 * 1024), b''):
                        digest.update(chunk)
                    if digest.hexdigest() != item['sha256']:
                        raise ValueError('Source ZIP file checksum verification failed')
        if (sha(inventory_path) != record['native_inventory_sha256']
                or any(sha(attachment.regular_file(bundle, f['path'])) != f['sha256'] for f in signed_files)):
            raise ValueError('Bundle inputs changed during assembly')
        subprocess.run(['codesign', '--verify', '--deep', '--strict', str(bundle)], check=True, timeout=120)
        os.link(archive, output)
    print(f'Created native source package: {output} ({output.stat().st_size} bytes)')
    return record


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('signed_bundle')
    parser.add_argument('source_materials_directory')
    parser.add_argument('new_output_zip')
    args = parser.parse_args()
    if sys.platform != 'darwin':
        parser.error('Requires macOS codesign verification')
    assemble(args.signed_bundle, args.source_materials_directory, args.new_output_zip)
