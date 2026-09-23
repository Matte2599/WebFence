#!/usr/bin/env python3
"""Attach verified MSYS2 source archives to private Windows package staging.

Trusted build inputs only; no network or recipe execution. Original archives
retain nested upstream sources, patches and notices. This is not a declaration
of source/license completeness or a reproducible binary rebuild.
"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import stat
import tempfile

import collect_windows_sources as sources
from msys2_binary_metadata import sha


def read_json(path):
    if path.stat().st_size > 16 * 1024 * 1024:
        raise ValueError('Manifest size budget exceeded')
    raw = path.read_bytes()
    return raw, json.loads(raw)


def regular_file(root, relative):
    path = root
    parts = sources.download.safe_path(relative).parts
    for index, part in enumerate(parts):
        path = path / part
        mode = path.lstat().st_mode
        expected = stat.S_ISREG if index == len(parts) - 1 else stat.S_ISDIR
        if not expected(mode):
            raise ValueError('Expected regular source material without links')
    return path


def attach(native_build, materials, output, zstd='zstd'):
    native_build, materials, output = Path(native_build), Path(materials).resolve(), Path(output)
    if output.exists() or output.is_symlink():
        raise FileExistsError('Refusing to replace existing Windows source materials')
    if (materials / 'INCOMPLETE').exists() or (materials / 'INCOMPLETE').is_symlink():
        raise ValueError('Windows source collection is incomplete')
    current_raw, current = read_json(native_build)
    original_raw, original = read_json(regular_file(materials, 'native-build.input.json'))
    manifest_raw, manifest = read_json(regular_file(materials, 'source-materials.json'))
    plan = sources.source_plan(current)
    if sources.source_plan(original) != plan:
        raise ValueError('Source package identities or recipe hashes differ from current build')
    if (manifest.get('schema') != 1
            or manifest.get('distribution_ready') is not False
            or manifest.get('corresponding_sources_complete') is not False
            or manifest.get('input_manifest_sha256') != hashlib.sha256(original_raw).hexdigest()):
        raise ValueError('Invalid source collection provenance or status')
    records = manifest.get('archives', [])
    if not isinstance(records, list) or len(records) != len(plan):
        raise ValueError('Source archive list differs from inventory')
    total = 0
    # Caller owns the staging directory; concurrent writers are unsupported.
    # Never overwrite a previous attachment or publish partially checked data.
    with tempfile.TemporaryDirectory(prefix='.webfence-windows-sources-', dir=output.parent) as temporary:
        stage = Path(temporary) / 'sources'
        stage.mkdir()
        for item, record in zip(plan, records):
            if any(record.get(key) != value for key, value in item.items()):
                raise ValueError('Source record differs from inventory')
            identity = item['base'] + '-' + item['version']
            relative = 'archives/' + identity + '.src.tar.zst'
            recipe_directory = 'recipes/' + identity
            url = 'https://repo.msys2.org/mingw/sources/' + identity + '.src.tar.zst'
            if (record.get('archive') != relative or record.get('recipe_directory') != recipe_directory
                    or record.get('url') != url):
                raise ValueError('Unexpected source material location')
            original_archive = regular_file(materials, relative)
            size = original_archive.stat().st_size
            total += size
            if size > sources.MAX_ARCHIVE or total > sources.MAX_TOTAL or size != record.get('size_bytes'):
                raise ValueError('Source archive size budget or metadata mismatch')
            archive = stage / relative
            archive.parent.mkdir(exist_ok=True)
            shutil.copyfile(original_archive, archive)
            # Check copied bytes, then regenerate recipes from that exact copy;
            # mutable loose recipe/metadata files are never attachment inputs.
            if sha(archive) != record.get('archive_sha256'):
                raise ValueError('Source archive checksum mismatch')
            inspected, metadata = sources.inspect_source(archive, item, zstd)
            if any(record.get(key) != value for key, value in inspected.items()):
                raise ValueError('Regenerated source evidence differs from collection manifest')
            destination = stage / recipe_directory
            destination.mkdir(parents=True)
            for name, raw in metadata.items():
                (destination / name).write_bytes(raw)
        (stage / 'source-materials.json').write_bytes(manifest_raw)
        (stage / 'native-build.input.json').write_bytes(original_raw)
        (stage / 'native-build.current.json').write_bytes(current_raw)
        result = {'schema': 1, 'distribution_ready': False, 'corresponding_sources_complete': False,
                  'current_native_build_sha256': hashlib.sha256(current_raw).hexdigest(),
                  'collection_input_sha256': hashlib.sha256(original_raw).hexdigest(),
                  'collection_manifest_sha256': hashlib.sha256(manifest_raw).hexdigest(),
                  'archive_count': len(records), 'archive_bytes': total,
                  'scope': 'Original MSYS2 archives with nested upstream sources, patches and notices; recipes regenerated from reverified archives.',
                  'provenance': 'native-build.current.json identifies this package; native-build.input.json identifies collection. Source manifest archive paths are relative to this directory.',
                  'open_items': ['Detached signatures, VCS and embedded-component/notice review',
                                 'Build environment, rebuild/replacement instructions and source/license completeness']}
        (stage / 'attachment.json').write_text(json.dumps(result, indent=2) + '\n', encoding='utf-8')
        (stage / 'README.md').write_text(
            '# Materiali sorgente MSYS2 / MSYS2 source materials\n\n'
            '**IT:** Archivi originali e ricette collegate alle DLL tramite inventario e hash. '
            'Gli archivi includono sorgenti, patch e avvisi annidati; non vengono eseguiti. '
            'Manifest di acquisizione e inventario corrente sono distinti. '
            'Firme, completezza, licenze e ricompilazione richiedono ancora revisione; nessuna approvazione della distribuzione.\n\n'
            '**EN:** Original archives and recipes bound to DLLs through inventory and hashes. '
            'Archives include nested sources, patches and notices; none are executed. '
            'Collection manifest and current inventory are separate. '
            'Signatures, completeness, licenses and rebuilding still require review; no distribution approval.\n', encoding='utf-8')
        if output.exists() or output.is_symlink():
            raise FileExistsError('Refusing to replace existing Windows source materials')
        stage.rename(output)
    return result


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('native_build_json')
    parser.add_argument('source_materials_directory')
    parser.add_argument('new_output_directory')
    parser.add_argument('--zstd', default='zstd')
    args = parser.parse_args()
    print(json.dumps(attach(args.native_build_json, args.source_materials_directory,
                            args.new_output_directory, args.zstd), indent=2))
