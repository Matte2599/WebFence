#!/usr/bin/env python3
"""Reject native macOS load references that require libraries outside the app.

Runs on a trusted, staged Apple Silicon development bundle. This checks Mach-O
load commands, not runtime dlopen calls or compatibility with older macOS.
"""

import json
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys

MACHO = {b'\xcf\xfa\xed\xfe', b'\xfe\xed\xfa\xcf', b'\xce\xfa\xed\xfe',
         b'\xfe\xed\xfa\xce', b'\xca\xfe\xba\xbe', b'\xbe\xba\xfe\xca',
         b'\xca\xfe\xba\xbf', b'\xbf\xba\xfe\xca'}


def otool(flag, binary):
    return subprocess.check_output(['otool', '-arch', 'arm64', flag, str(binary)], text=True)


def dependencies(output):
    result = []
    for line in output.splitlines()[1:]:
        match = re.fullmatch(r'\s+(.+?) \(compatibility version [^)]+, current version [^)]+\)\s*', line)
        if not match:
            raise ValueError('Unrecognized otool -L dependency')
        result.append(match.group(1))
    return result


def install_name(output):
    lines = output.splitlines()
    if len(lines) > 2:
        raise ValueError('Ambiguous otool -D output')
    return lines[1].strip() if len(lines) == 2 else None


def rpaths(output):
    lines = output.splitlines()
    result = []
    for index, line in enumerate(lines):
        if line.strip() != 'cmd LC_RPATH':
            continue
        if index + 2 >= len(lines):
            raise ValueError('Truncated LC_RPATH')
        match = re.fullmatch(r'\s*path (.+) \(offset [0-9]+\)\s*', lines[index + 2])
        if not match:
            raise ValueError('Unrecognized LC_RPATH')
        result.append(match.group(1))
    return result


def resolve_token(value, binary, app):
    bases = {'@loader_path': binary.parent, '@executable_path': app / 'Contents/MacOS'}
    for token, base in bases.items():
        if value == token or value.startswith(token + '/'):
            return (base / value[len(token):].lstrip('/')).resolve()
    raise ValueError('Unsupported or external Mach-O path: ' + value)


def rpath_target(value, binary, app):
    if value.startswith('/'):
        return Path(value).resolve()
    return resolve_token(value, binary, app)


def is_macho(path):
    with path.open('rb') as stream:
        return stream.read(4) in MACHO


def sanitize(app, inspect=otool, edit=subprocess.run):
    """Delete search paths outside a private bundle before inventory/signing.

    This mutates only Mach-O files under the given staging app. Run check()
    afterwards: an @rpath dependency that needed a deleted path must then fail.
    """
    app = Path(app).resolve()
    binaries = sorted(path for path in app.rglob('*') if path.is_file() and not path.is_symlink()
                      and is_macho(path))
    if not binaries:
        raise ValueError('No Mach-O files in staged app')
    removed = 0
    for binary in binaries:
        if not binary.resolve().is_relative_to(app):
            raise ValueError('Native binary escapes staged app')
        if binary.stat().st_nlink != 1:
            raise ValueError('Refusing to edit a shared Mach-O inode')
        for value in rpaths(inspect('-l', binary)):
            if not rpath_target(value, binary, app).is_relative_to(app):
                edit(['install_name_tool', '-delete_rpath', value, str(binary)], check=True)
                removed += 1
        if any(not rpath_target(value, binary, app).is_relative_to(app)
               for value in rpaths(inspect('-l', binary))):
            raise ValueError('External LC_RPATH remains after cleanup')
    return {'binary_count': len(binaries), 'removed_external_rpaths': removed}


def check(app, inspect=otool):
    app = Path(app).resolve()
    inventory = json.loads((app / 'Contents/Resources/notices/native-build.json').read_text())
    files = inventory['files']
    if not files:
        raise ValueError('Empty native inventory')
    seen = set()
    total = system = bundled = 0
    for entry in files:
        path = entry['path']
        if not isinstance(path, str):
            raise ValueError('Invalid native inventory path')
        posix = PurePosixPath(path)
        if (not posix.parts or posix.is_absolute() or '..' in posix.parts
                or posix.parts[0] != 'Contents' or path in seen):
            raise ValueError('Invalid or duplicate native inventory path')
        seen.add(path)
        binary = (app / path).resolve()
        if not binary.is_file() or not binary.is_relative_to(app):
            raise ValueError('Missing or external native binary: ' + path)
        deps = dependencies(inspect('-L', binary))
        own = install_name(inspect('-D', binary))
        if own:
            if not deps or deps[0] != own:
                raise ValueError('Install name does not match first dependency: ' + path)
            deps.pop(0)
        search = [rpath_target(value, binary, app) for value in rpaths(inspect('-l', binary))]
        if any(not directory.is_relative_to(app) for directory in search):
            raise ValueError('External LC_RPATH: ' + path)
        for dep in deps:
            total += 1
            if dep.startswith(('/System/Library/', '/usr/lib/')):
                system += 1
                continue
            if dep.startswith('@rpath/'):
                name = dep.removeprefix('@rpath/')
                if not name or '..' in PurePosixPath(name).parts:
                    raise ValueError('Invalid @rpath dependency: ' + dep)
                targets = [(base / name).resolve() for base in search if (base / name).is_file()]
                if len(targets) != 1:
                    raise ValueError('Unresolved or ambiguous @rpath dependency: ' + dep)
                target = targets[0]
            else:
                target = resolve_token(dep, binary, app)
            if not target.is_file() or not target.is_relative_to(app):
                raise ValueError('Dependency escapes or is absent from bundle: ' + dep)
            bundled += 1
    return {'binary_count': len(files), 'load_references': total,
            'apple_system_references': system, 'bundle_references': bundled}


if __name__ == '__main__':
    if len(sys.argv) == 3 and sys.argv[1] == '--sanitize':
        result = sanitize(sys.argv[2])
        print('PASS macOS staging rpaths: {binary_count} binaries, '
              '{removed_external_rpaths} external paths removed'.format(**result))
    elif len(sys.argv) == 2:
        result = check(sys.argv[1])
        print('PASS macOS Mach-O linkage: {binary_count} binaries, {load_references} load references, '
              '{apple_system_references} Apple system, {bundle_references} resolved in bundle'.format(**result))
    else:
        raise SystemExit('usage: check-macos-linkage.py [--sanitize] WebFence.app')
