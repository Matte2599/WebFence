#!/usr/bin/env python3
"""Attach verified upstream notices to a staged package before final signing.

Trusted build inputs only. Original source archives stay in the source sidecar.
Notices are regenerated from verified archives, not copied from mutable loose
files. This does not establish complete corresponding sources or legal approval.
"""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import stat
import tempfile

spec = importlib.util.spec_from_file_location('native_sources', Path(__file__).with_name('collect-native-sources.py'))
sources = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sources)


def read_json(path):
    if path.stat().st_size > 16 * 1024 * 1024:
        raise ValueError('Manifest size budget exceeded')
    raw = path.read_bytes()
    return raw, json.loads(raw)


def regular_file(root, relative):
    path = root
    parts = sources.safe_path(relative).parts
    for index, part in enumerate(parts):
        path = path / part
        mode = path.lstat().st_mode
        expected = stat.S_ISREG if index == len(parts) - 1 else stat.S_ISDIR
        if not expected(mode):
            raise ValueError('Expected regular source material without links')
    return path


def attach(native_build, materials, output):
    native_build, materials, output = Path(native_build), Path(materials).resolve(), Path(output)
    if output.exists() or output.is_symlink():
        raise FileExistsError('Refusing to replace existing source notices')
    if (materials / 'INCOMPLETE').exists():
        raise ValueError('Source collection is incomplete')
    current_raw, current = read_json(native_build)
    input_raw, original = read_json(regular_file(materials, 'native-build.input.json'))
    source_raw, manifest = read_json(regular_file(materials, 'source-materials.json'))
    plan = sources.archive_plan(current)
    if sources.archive_plan(original) != plan:
        raise ValueError('Source package identities, URLs or hashes differ from current build')
    if (manifest.get('schema') != 1
            or manifest.get('distribution_ready') is not False
            or manifest.get('corresponding_sources_complete') is not False
            or manifest.get('input_manifest_sha256') != hashlib.sha256(input_raw).hexdigest()):
        raise ValueError('Invalid source collection provenance or status')
    records = manifest.get('archives', [])
    if not isinstance(records, list) or len(records) != len(plan):
        raise ValueError('Source archive list differs from inventory')
    seen, total = set(), 0
    # The package builder owns this staging directory. Concurrent writers are
    # unsupported; no existing artifact is modified on verification failure.
    with tempfile.TemporaryDirectory(prefix='.webfence-notices-', dir=output.parent) as temporary:
        stage = Path(temporary) / 'upstream'
        stage.mkdir()
        for item, record in zip(plan, records):
            if any(record.get(key) != value for key, value in item.items()):
                raise ValueError('Source record differs from inventory')
            digest = item['sha256']
            relative = 'archives/' + digest + '.archive'
            notice_root = 'source-notices/' + item['package'] + '/' + digest
            if record.get('archive') != relative or record.get('notice_root') != notice_root:
                raise ValueError('Unexpected source material path')
            archive = regular_file(materials, relative)
            size = archive.stat().st_size
            if digest not in seen:
                total += size
                seen.add(digest)
            if (size > sources.MAX_ARCHIVE or total > sources.MAX_TOTAL_ARCHIVES
                    or size != record.get('size_bytes') or sources.sha(archive) != digest):
                raise ValueError('Source archive size/checksum verification failed')
            target = stage / notice_root
            target.mkdir(parents=True)
            references = sources.FREETYPE_REFERENCES if item['package'].startswith('freetype@') else ()
            extracted = sources.collect_notices(archive, target, references)
            if any(record.get(key) != value for key, value in extracted.items()):
                raise ValueError('Regenerated source notices differ from collection manifest')
        (stage / 'source-materials.json').write_bytes(source_raw)
        (stage / 'native-build.input.json').write_bytes(input_raw)
        result = {'schema': 1, 'distribution_ready': False, 'corresponding_sources_complete': False,
                  'current_native_build_sha256': hashlib.sha256(current_raw).hexdigest(),
                  'collection_manifest_sha256': hashlib.sha256(source_raw).hexdigest(),
                  'collection_input_sha256': hashlib.sha256(input_raw).hexdigest(),
                  'archive_count': len(records), 'archive_bytes': total,
                  'notice_count': sum(len(r['files']) for r in records),
                  'scope': 'Source-tree notices regenerated from verified upstream archives; may include unshipped components.',
                  'archives_location': 'Separate source-material sidecar; archives paths in source-materials.json are relative to that sidecar.',
                  'provenance': 'The collection input describes acquisition, not this executable. Current build provenance remains in ../native-build.json.',
                  'open_items': ['Package-manager patches/resources and exact build environment',
                                 'Embedded-component mapping, license review and rebuild/replacement instructions']}
        (stage / 'attachment.json').write_text(json.dumps(result, indent=2) + '\n')
        if output.exists() or output.is_symlink():
            raise FileExistsError('Refusing to replace existing source notices')
        stage.rename(output)
    return result


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('native_build_json')
    parser.add_argument('source_materials_directory')
    parser.add_argument('new_output_directory')
    args = parser.parse_args()
    print(json.dumps(attach(args.native_build_json, args.source_materials_directory, args.new_output_directory), indent=2))
