"""Verify pinned MSYS2 binary package signatures without network access.

The key is bound to the reviewed MSYS2 keyring package and official fingerprint.
Cryptographic validity alone does not establish independent signer identity.
"""
import json
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile

from msys2_binary_metadata import MAX_ARCHIVE, sha
from verify_windows_signatures import _gpg_environment, _key_fingerprints, _run_gpg, _verified_signer


HEX_SHA = re.compile(r'[0-9a-f]{64}\Z')
HEX_FINGERPRINT = re.compile(r'[0-9A-F]{40}\Z')
MAX_LOCK = 64 * 1024
MAX_KEY = 64 * 1024
MAX_SIGNATURE = 16 * 1024


def load_lock(path, binary_lock):
    path = Path(path)
    if not path.is_file() or path.is_symlink() or path.stat().st_size > MAX_LOCK:
        raise ValueError('Invalid Windows binary signature lock')
    data = json.loads(path.read_text(encoding='utf-8'))
    if (not isinstance(data, dict) or set(data) != {'schema', 'reviewed_on', 'key', 'packages'}
            or type(data['schema']) is not int or data['schema'] != 1
            or not isinstance(data['reviewed_on'], str)
            or not re.fullmatch(r'20\d\d-\d\d-\d\d', data['reviewed_on'])
            or not isinstance(data['packages'], list) or not isinstance(data['key'], dict)):
        raise ValueError('Invalid Windows binary signature lock schema')
    key = data['key']
    fields = {'fingerprint', 'public_key_file', 'public_key_sha256', 'official_key_page',
              'keyring_package_url', 'keyring_package_sha256', 'keyring_member',
              'keyring_member_sha256'}
    if set(key) != fields:
        raise ValueError('Invalid Windows binary signing key record')
    fingerprint = key['fingerprint']
    if (not isinstance(fingerprint, str) or not HEX_FINGERPRINT.fullmatch(fingerprint)
            or key['public_key_file'] != 'binary-signing-keys/' + fingerprint + '.asc'
            or not isinstance(key['public_key_sha256'], str)
            or not HEX_SHA.fullmatch(key['public_key_sha256'])
            or key['official_key_page'] != 'https://www.msys2.org/dev/keyring/'
            or not isinstance(key['keyring_package_url'], str)
            or not re.fullmatch(
                r'https://mirror\.msys2\.org/msys/x86_64/msys2-keyring-[A-Za-z0-9.~_-]+-any\.pkg\.tar\.zst',
                key['keyring_package_url'])
            or key['keyring_member'] != 'usr/share/pacman/keyrings/msys2.gpg'
            or any(not isinstance(key[field], str) or not HEX_SHA.fullmatch(key[field])
                   for field in ('keyring_package_sha256', 'keyring_member_sha256'))):
        raise ValueError('Invalid Windows binary signing key provenance')
    result = {}
    item_fields = {'name', 'version', 'archive_sha256', 'signature_file',
                   'signature_sha256', 'signature_size_bytes', 'signature_source'}
    for item in data['packages']:
        if not isinstance(item, dict) or set(item) != item_fields:
            raise ValueError('Invalid Windows binary signature entry')
        name, version = item['name'], item['version']
        if (not isinstance(name, str) or name not in binary_lock or name in result
                or not isinstance(version, str) or version != binary_lock[name]['version']
                or item['archive_sha256'] != binary_lock[name]['sha256']
                or not isinstance(item['signature_sha256'], str)
                or not HEX_SHA.fullmatch(item['signature_sha256'])
                or type(item['signature_size_bytes']) is not int
                or not 1 <= item['signature_size_bytes'] <= MAX_SIGNATURE):
            raise ValueError('Unreviewed Windows binary signature entry')
        filename = name + '-' + version + '-any.pkg.tar.zst.sig'
        if (item['signature_file'] != 'binary-package-signatures/' + filename
                or item['signature_source'] != 'https://mirror.msys2.org/mingw/ucrt64/' + filename):
            raise ValueError('Invalid Windows binary signature location')
        result[name] = item
    if set(result) != set(binary_lock):
        raise ValueError('Incomplete Windows binary signature lock')
    return key, result


def verify_signature(archive_path, entry, key, *, key_root, gpg='gpg'):
    """Check exact archive/key/signature bytes, then GnuPG detached signature."""
    archive_path, key_root = Path(archive_path), Path(key_root)
    if (not archive_path.is_file() or archive_path.is_symlink()
            or archive_path.stat().st_size > MAX_ARCHIVE
            or sha(archive_path) != entry['archive_sha256']):
        raise ValueError('Windows binary archive differs from signature lock')
    for path, expected, limit, message in (
            (key_root / key['public_key_file'], key['public_key_sha256'], MAX_KEY, 'key'),
            (key_root / entry['signature_file'], entry['signature_sha256'], MAX_SIGNATURE, 'signature')):
        if (not path.is_file() or path.is_symlink() or path.stat().st_size > limit
                or sha(path) != expected):
            raise ValueError('Missing or changed Windows binary ' + message)
    signature = key_root / entry['signature_file']
    if signature.stat().st_size != entry['signature_size_bytes']:
        raise ValueError('Windows binary signature size differs from lock')
    if shutil.which(gpg) is None:
        raise ValueError('GnuPG is required for Windows binary signature verification')
    short_temp = '/tmp' if sys.platform == 'darwin' else None
    with tempfile.TemporaryDirectory(prefix='wf-pkg-gpg-', dir=short_temp) as temporary:
        root = Path(temporary)
        home = root / 'gnupg'
        home.mkdir(mode=0o700)
        env = _gpg_environment(root)
        shutil.copyfile(key_root / key['public_key_file'], root / 'key.asc')
        shutil.copyfile(signature, root / 'signature.sig')
        if sha(root / 'key.asc') != key['public_key_sha256'] or sha(root / 'signature.sig') != entry['signature_sha256']:
            raise ValueError('Copied Windows binary signature material differs from lock')
        _run_gpg(gpg, home, env, '--import', 'key.asc')
        listing = _run_gpg(gpg, home, env, '--with-colons', '--fingerprint',
                           '--fingerprint', '--list-keys').stdout
        primaries, _ = _key_fingerprints(listing)
        if primaries != [key['fingerprint']]:
            raise ValueError('Windows binary signing key fingerprint differs from lock')
        try:
            with archive_path.open('rb') as payload:
                result = subprocess.run(
                    [gpg, '--homedir', home.name, '--no-options', '--batch',
                     '--no-auto-key-retrieve', '--status-fd', '1', '--verify',
                     'signature.sig', '-'], cwd=root, env=env, stdin=payload,
                    capture_output=True, text=True, check=True, timeout=120)
        except subprocess.CalledProcessError as error:
            detail = ' | '.join(line.strip() for line in (error.stderr or '').splitlines())
            detail = detail.encode('unicode_escape').decode('ascii')[-600:]
            raise ValueError('Offline Windows binary signature verification failed: ' + detail) from error
        except (OSError, subprocess.SubprocessError) as error:
            raise ValueError('Offline Windows binary signature verification failed') from error
        _verified_signer(result.stdout, key['fingerprint'], key['fingerprint'])
    return {'method': 'offline_openpgp_detached_signature',
            'signer_fingerprint': key['fingerprint'],
            'public_key_sha256': key['public_key_sha256'],
            'signature_sha256': entry['signature_sha256'],
            'archive_sha256': entry['archive_sha256']}
