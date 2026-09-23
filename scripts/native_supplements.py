#!/usr/bin/env python3
"""Collect a reviewed Homebrew supplement plan bound to current recipe hashes.

Trusted build metadata only. No recipe, patch or archive content is executed.
Publish all verified files together; never replace an existing attachment.
"""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import shutil
import stat
import tempfile
from urllib.parse import urlsplit

spec = importlib.util.spec_from_file_location('supplement_download', Path(__file__).with_name('collect-native-sources.py'))
download = importlib.util.module_from_spec(spec)
spec.loader.exec_module(download)
MAX_FILE = 16 * 1024 * 1024
MAX_TOTAL = 64 * 1024 * 1024


def read_json(path):
    if path.stat().st_size > 16 * 1024 * 1024:
        raise ValueError('Supplement manifest size budget exceeded')
    raw = path.read_bytes()
    return raw, json.loads(raw)


def regular_file(root, relative):
    parts = download.safe_path(relative).parts
    path = root
    for index, part in enumerate(parts):
        path /= part
        expected = stat.S_ISREG if index == len(parts) - 1 else stat.S_ISDIR
        if not expected(path.lstat().st_mode):
            raise ValueError('Expected regular supplement material without links')
    return path


def validate(native_build, inventory, plan):
    if inventory.get('schema') != 1 or plan.get('schema') != 1:
        raise ValueError('Expected schema 1 inventory and supplement plan')
    packages = plan.get('packages', {})
    if not isinstance(packages, dict) or not 1 <= len(packages) <= 16:
        raise ValueError('Expected 1–16 supplement packages')
    result, total, count = [], 0, 0
    for package, record in sorted(packages.items()):
        if not re.fullmatch(r'[a-z0-9+_.-]+@[A-Za-z0-9+_.-]+', package):
            raise ValueError('Invalid supplement package identity')
        current = inventory.get('packages', {}).get(package)
        if not current or package != current.get('formula', '') + '@' + current.get('version', ''):
            raise ValueError('Supplement package differs from current inventory')
        recipe = '.brew/' + current['formula'] + '.rb'
        if recipe not in current.get('metadata_files', []):
            raise ValueError('Expected installed recipe in native inventory')
        recipe_file = regular_file(native_build.parent, 'homebrew/' + package + '/' + recipe)
        recipe_hash = record.get('installed_recipe_sha256', '')
        if (not re.fullmatch(r'[a-f0-9]{64}', recipe_hash) or recipe_file.stat().st_size > MAX_FILE
                or download.sha(recipe_file) != recipe_hash):
            raise ValueError('Installed recipe differs from reviewed supplement plan')
        files = record.get('files', [])
        if not isinstance(files, list) or not files:
            raise ValueError('Expected supplement files')
        seen = set()
        for entry in files:
            name, url, size, digest = (entry.get(k) for k in ('path', 'url', 'size_bytes', 'sha256'))
            if (not isinstance(name, str) or not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._+-]*', name)
                    or name in seen or name == 'installed-recipe.rb'):
                raise ValueError('Invalid or duplicate supplement filename')
            seen.add(name)
            parts = urlsplit(url)
            if (parts.scheme != 'https' or not parts.hostname or parts.username or parts.password
                    or parts.fragment or any(ord(c) < 32 for c in url)):
                raise ValueError('Expected credential-free HTTPS supplement URL')
            if (type(size) is not int or not 0 < size <= MAX_FILE
                    or not isinstance(digest, str) or not re.fullmatch(r'[a-f0-9]{64}', digest)):
                raise ValueError('Invalid supplement size or checksum')
            count += 1
            total += size
            if count > 64 or total > MAX_TOTAL:
                raise ValueError('Combined supplement budget exceeded')
        result.append((package, recipe_file, record))
    return result


def collect(native_build, plan_path, output, reuse_directory=None):
    native_build, plan_path, output = Path(native_build), Path(plan_path), Path(output)
    if output.exists() or output.is_symlink():
        raise FileExistsError('Refusing to replace existing native supplements')
    current_raw, current = read_json(native_build)
    plan_raw, plan = read_json(plan_path)
    reviewed = validate(native_build, current, plan)
    records, total = [], 0
    # Private builder staging, no concurrent writers. All output is transient
    # until checks pass; a failure cannot modify the previous application.
    with tempfile.TemporaryDirectory(prefix='.webfence-supplements-', dir=output.parent) as temporary:
        stage = Path(temporary) / 'supplements'
        stage.mkdir()
        for package, recipe, record in reviewed:
            directory = stage / package
            directory.mkdir()
            copied_recipe = directory / 'installed-recipe.rb'
            shutil.copyfile(recipe, copied_recipe)
            if download.sha(copied_recipe) != record['installed_recipe_sha256']:
                raise ValueError('Installed recipe changed while collecting supplements')
            for entry in record['files']:
                relative = package + '/' + entry['path']
                target = stage / relative
                if reuse_directory:
                    source = regular_file(Path(reuse_directory).resolve(), relative)
                    if source.stat().st_size != entry['size_bytes']:
                        raise ValueError('Reusable supplement size mismatch')
                    shutil.copyfile(source, target)
                else:
                    download.fetch(entry['url'], target, entry['size_bytes'])
                if target.stat().st_size != entry['size_bytes'] or download.sha(target) != entry['sha256']:
                    raise ValueError('Supplement size/checksum mismatch')
                total += entry['size_bytes']
                records.append(dict(entry, path=relative, package=package))
        (stage / 'plan.json').write_bytes(plan_raw)
        (stage / 'native-build.current.json').write_bytes(current_raw)
        result = {'schema': 1, 'distribution_ready': False, 'corresponding_sources_complete': False,
                  'native_build_sha256': hashlib.sha256(current_raw).hexdigest(),
                  'plan_sha256': hashlib.sha256(plan_raw).hexdigest(), 'files': records,
                  'file_count': len(records), 'supplement_bytes': total,
                  'installed_recipes': [{'package': p, 'path': p + '/installed-recipe.rb',
                                         'sha256': r['installed_recipe_sha256']} for p, _, r in reviewed],
                  'scope': 'Reviewed recipe-bound supplements only; no patch/recipe execution or automatic license approval.',
                  'open_items': ['Other package resources and exact build environment',
                                 'Embedded notices, rebuild/replacement and distribution review']}
        (stage / 'attachment.json').write_text(json.dumps(result, indent=2) + '\n')
        if output.exists() or output.is_symlink():
            raise FileExistsError('Refusing to replace existing native supplements')
        stage.rename(output)
    return result


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('native_build_json')
    parser.add_argument('reviewed_plan_json')
    parser.add_argument('new_output_directory')
    parser.add_argument('--reuse-directory', help='Use package/filename layout; verify all files without network')
    args = parser.parse_args()
    print(json.dumps(collect(args.native_build_json, args.reviewed_plan_json,
                             args.new_output_directory, args.reuse_directory), indent=2))
