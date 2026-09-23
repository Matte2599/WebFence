"""Inspect trusted cached MSYS2 packages without installing or extracting code."""
from contextlib import contextmanager
import hashlib
from pathlib import Path, PurePosixPath
import re
import subprocess
import tarfile

MAX_ARCHIVE = 512 * 1024 * 1024
MAX_UNPACKED = 2 * 1024 * 1024 * 1024
MAX_METADATA = 1024 * 1024


def sha(path):
    digest = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def fields(raw):
    result = {}
    for line in raw.decode('utf-8').splitlines():
        line = line.strip()
        if not line or line.startswith('#'):
            continue
        if ' = ' not in line:
            raise ValueError('Invalid package metadata line')
        key, value = line.split(' = ', 1)
        result.setdefault(key, []).append(value)
    return result


def single(record, key):
    values = record.get(key, [])
    if len(values) != 1 or not values[0]:
        raise ValueError('Expected one package metadata value: ' + key)
    return values[0]


@contextmanager
def tar_stream(path, zstd):
    if not str(path).endswith('.zst'):
        with tarfile.open(path, mode='r|*') as archive:
            yield archive
        return
    # zstd is a build tool, not code from the package. No shell is involved.
    process = subprocess.Popen([zstd, '-dc', '--', str(path)], stdout=subprocess.PIPE)
    try:
        with tarfile.open(fileobj=process.stdout, mode='r|') as archive:
            yield archive
        # Drain normal tar padding, while bounding unexpected trailing output.
        if len(process.stdout.read(1024 * 1024 + 1)) > 1024 * 1024:
            raise ValueError('Excess trailing package data')
        if process.wait(timeout=10) != 0:
            raise ValueError('Package decompression failed')
    finally:
        process.stdout.close()
        if process.poll() is None:
            process.terminate()
        process.wait(timeout=10)


def inspect(archive_path, name, version, required_files, zstd='zstd'):
    archive_path = Path(archive_path)
    if (not archive_path.is_file() or archive_path.is_symlink()
            or archive_path.stat().st_size > MAX_ARCHIVE):
        raise ValueError('Invalid cached binary archive')
    if not required_files:
        raise ValueError('Expected package DLLs to verify')
    metadata, matched = {}, set()
    total = 0
    with tar_stream(archive_path, zstd) as archive:
        for count, member in enumerate(archive, 1):
            total += member.size
            if count > 100000 or member.size < 0 or total > MAX_UNPACKED:
                raise ValueError('Package entry/size budget exceeded')
            path = member.name.removeprefix('./').rstrip('/')
            if not path:  # Root directory in some tar implementations.
                if member.isdir():
                    continue
                raise ValueError('Empty package entry path')
            if (PurePosixPath(path).is_absolute() or '\\' in path or ':' in path
                    or any(p in {'', '.', '..'} for p in path.split('/'))):
                raise ValueError('Unsafe package path')
            if path not in {'.PKGINFO', '.BUILDINFO'} and path not in required_files:
                continue
            if not member.isfile() or path in metadata or path in matched:
                raise ValueError('Duplicate or nonregular package evidence')
            with archive.extractfile(member) as stream:
                if path in {'.PKGINFO', '.BUILDINFO'}:
                    if member.size > MAX_METADATA:
                        raise ValueError('Package metadata size budget exceeded')
                    metadata[path] = stream.read()
                else:
                    digest = hashlib.sha256()
                    for chunk in iter(lambda: stream.read(1024 * 1024), b''):
                        digest.update(chunk)
                    if digest.hexdigest() != required_files[path]:
                        raise ValueError('Installed DLL differs from cached package: ' + path)
                    matched.add(path)
    if matched != set(required_files) or set(metadata) != {'.PKGINFO', '.BUILDINFO'}:
        raise ValueError('Missing DLL or build metadata in cached package')
    package, build = fields(metadata['.PKGINFO']), fields(metadata['.BUILDINFO'])
    base = single(package, 'pkgbase')
    if not re.fullmatch(r'mingw-w64-[a-z0-9+_.-]+', base):
        raise ValueError('Unexpected MSYS2 source package identity')
    if (single(package, 'pkgname') != name or single(package, 'pkgver') != version
            or name not in build.get('pkgname', []) or single(build, 'pkgver') != version
            or single(build, 'pkgbase') != base
            or single(package, 'arch') != single(build, 'pkgarch')):
        raise ValueError('Cached package/build identity differs from installed package')
    recipe = single(build, 'pkgbuild_sha256sum')
    if single(build, 'format') != '2' or not re.fullmatch(r'[0-9a-f]{64}', recipe):
        raise ValueError('Expected BUILDINFO v2 with PKGBUILD SHA-256')
    return {'filename': archive_path.name, 'sha256': sha(archive_path),
            'size_bytes': archive_path.stat().st_size, 'source_package': base,
            'version': version, 'pkgbuild_sha256': recipe, 'verified_dll_count': len(matched),
            'signature_verification': 'not performed by this collector; cached package and installed DLL correspondence only'}, metadata
