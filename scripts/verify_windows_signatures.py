"""Verify reviewed MSYS2 source signatures with pinned public keys, offline."""
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

from msys2_binary_metadata import sha, tar_stream


EXPECTED_PACKAGES = frozenset({
    'mingw-w64-freetype', 'mingw-w64-gcc', 'mingw-w64-gettext',
    'mingw-w64-libiconv', 'mingw-w64-libjpeg-turbo', 'mingw-w64-pcre2',
    'mingw-w64-zlib', 'mingw-w64-zstd',
})
MAX_LOCK = 64 * 1024
MAX_KEY = 64 * 1024
MAX_PAYLOAD = 256 * 1024 * 1024
MAX_SIGNATURE = 16 * 1024
HEX_SHA = re.compile(r'[0-9a-f]{64}\Z')
HEX_FINGERPRINT = re.compile(r'[0-9A-F]{40}\Z')
SOURCE_NAME = re.compile(r'[A-Za-z0-9][A-Za-z0-9._+~-]*\Z')


def load_lock(path):
    path = Path(path)
    if not path.is_file() or path.is_symlink() or path.stat().st_size > MAX_LOCK:
        raise ValueError('Invalid Windows source signature lock')
    data = json.loads(path.read_text(encoding='utf-8'))
    if (not isinstance(data, dict) or set(data) != {'schema', 'entries'}
            or type(data['schema']) is not int or data['schema'] != 1
            or not isinstance(data['entries'], list)):
        raise ValueError('Invalid Windows source signature lock schema')
    result = {}
    fields = {'source_package', 'version', 'pkgbuild_sha256', 'archive_sha256',
              'signature_source', 'signature_path', 'signature_sha256', 'signature_size_bytes',
              'payload_source', 'payload_path', 'payload_sha256', 'payload_size_bytes',
              'primary_fingerprint', 'signer_fingerprint', 'public_key_file',
              'public_key_sha256', 'key_retrieval_url'}
    for entry in data['entries']:
        if not isinstance(entry, dict) or set(entry) != fields:
            raise ValueError('Invalid Windows source signature lock entry')
        base, primary = entry['source_package'], entry['primary_fingerprint']
        if (not isinstance(base, str) or base not in EXPECTED_PACKAGES or base in result
                or not isinstance(entry['version'], str) or not SOURCE_NAME.fullmatch(entry['version'])
                or any(not isinstance(entry[key], str) or not HEX_SHA.fullmatch(entry[key])
                       for key in ('pkgbuild_sha256', 'archive_sha256', 'signature_sha256',
                                   'payload_sha256', 'public_key_sha256'))
                or not isinstance(primary, str) or not HEX_FINGERPRINT.fullmatch(primary)
                or not isinstance(entry['signer_fingerprint'], str)
                or not HEX_FINGERPRINT.fullmatch(entry['signer_fingerprint'])
                or entry['public_key_file'] != 'source-signing-keys/' + primary + '.asc'
                or not isinstance(entry['signature_path'], str)
                or not SOURCE_NAME.fullmatch(entry['signature_path'])
                or not entry['signature_path'].endswith(('.sig', '.asc'))
                or not isinstance(entry['payload_path'], str)
                or not SOURCE_NAME.fullmatch(entry['payload_path'])
                or entry['signature_path'].rsplit('.', 1)[0] != entry['payload_path']
                or type(entry['signature_size_bytes']) is not int
                or not 1 <= entry['signature_size_bytes'] <= MAX_SIGNATURE
                or type(entry['payload_size_bytes']) is not int
                or not 1 <= entry['payload_size_bytes'] <= MAX_PAYLOAD
                or any(not isinstance(entry[key], str) or not entry[key].startswith('https://')
                       for key in ('signature_source', 'payload_source'))
                or not isinstance(entry['key_retrieval_url'], str)
                or entry['key_retrieval_url'] not in {
                    'https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x' + primary,
                    'https://keys.openpgp.org/vks/v1/by-fingerprint/' + primary,
                }):
            raise ValueError('Invalid Windows source signature lock entry')
        result[base] = entry
    if set(result) != EXPECTED_PACKAGES:
        raise ValueError('Incomplete Windows source signature lock')
    return result


def _gpg_environment(home):
    env = {key: value for key, value in os.environ.items()
           if not key.startswith('GPG_') and key not in {'GNUPGHOME', 'XDG_CONFIG_HOME', 'XDG_CONFIG_DIRS'}}
    env.update(GNUPGHOME=str(home / 'gnupg'), HOME=str(home),
               XDG_CONFIG_HOME=str(home / 'xdg'), LC_ALL='C')
    return env


def _run_gpg(gpg, home, env, *arguments, timeout=30):
    try:
        return subprocess.run([gpg, '--homedir', home.name, '--no-options', '--batch',
                               '--no-auto-key-retrieve', *arguments], env=env,
                              cwd=home.parent, capture_output=True, text=True,
                              check=True, timeout=timeout)
    except (OSError, subprocess.SubprocessError) as error:
        raise ValueError('Offline Windows source signature verification failed') from error


def _key_fingerprints(key_listing):
    primary, subkeys, current = [], [], None
    for line in key_listing.splitlines():
        parts = line.split(':')
        if parts[0] in {'pub', 'sub'}:
            current = parts[0]
        elif parts[0] == 'fpr' and len(parts) > 9:
            if current == 'pub':
                primary.append(parts[9])
            elif current == 'sub':
                subkeys.append(parts[9])
            current = None
    return primary, subkeys


def _verified_signer(status, expected_signer, expected_primary):
    lines = [line.removeprefix('[GNUPG:] ').split() for line in status.splitlines()
             if line.startswith('[GNUPG:] ')]
    valid = [line for line in lines if line and line[0] == 'VALIDSIG']
    bad = {'BADSIG', 'ERRSIG', 'REVKEYSIG', 'EXPKEYSIG', 'SIGEXPIRED',
           'KEYEXPIRED', 'NODATA', 'NO_PUBKEY'}
    if (len(valid) != 1 or len(valid[0]) < 11
            or valid[0][1] != expected_signer or valid[0][-1] != expected_primary
            or any(line and line[0] in bad for line in lines)):
        raise ValueError('Windows source signer differs from reviewed key')


def verify_signature(archive_path, item, files, srcinfo, checks, entry, *,
                     key_root, zstd='zstd', gpg='gpg'):
    """Verify one detached signature against the reviewed recipe and key."""
    if (item['base'] != entry['source_package'] or item['version'] != entry['version']
            or item['pkgbuild_sha256'] != entry['pkgbuild_sha256']
            or sha(archive_path) != entry['archive_sha256']):
        raise ValueError('Unreviewed Windows signed source archive or recipe')
    primary = entry['primary_fingerprint']
    if primary not in srcinfo.get('validpgpkeys', []):
        raise ValueError('Source recipe does not allow reviewed signer')
    matching = lambda source: [check for check in checks if check['source'] == source]
    signatures = matching(entry['signature_source'])
    payloads = matching(entry['payload_source'])
    if (len(signatures) != 1 or signatures[0].get('path') != entry['signature_path']
            or signatures[0]['status'] != 'no_strong_checksum'
            or len(payloads) != 1 or payloads[0].get('path') != entry['payload_path']
            or payloads[0]['status'] != 'verified'
            or 'sha256' not in payloads[0]['checksums_verified']):
        raise ValueError('Windows signature declaration differs from reviewed lock')
    for kind in ('signature', 'payload'):
        record = files.get(entry[kind + '_path'])
        if (record is None or record['size_bytes'] != entry[kind + '_size_bytes']
                or record['hashes']['sha256'] != entry[kind + '_sha256']):
            raise ValueError('Windows signed source bytes differ from reviewed lock')
    key = Path(key_root) / entry['public_key_file']
    if (not key.is_file() or key.is_symlink() or key.stat().st_size > MAX_KEY
            or sha(key) != entry['public_key_sha256']):
        raise ValueError('Missing or changed reviewed Windows source signing key')
    if shutil.which(gpg) is None:
        raise ValueError('GnuPG is required for offline Windows source signature verification')
    with tempfile.TemporaryDirectory(prefix='webfence-source-signature-') as temporary:
        root = Path(temporary)
        home = root / 'gnupg'
        home.mkdir(mode=0o700)
        env = _gpg_environment(root)
        copied = set()
        wanted = {entry['signature_path'], entry['payload_path']}
        with tar_stream(archive_path, zstd) as archive:
            for member in archive:
                name = member.name.removeprefix('./')
                prefix = item['base'] + '/'
                if not name.startswith(prefix):
                    continue
                relative = name[len(prefix):]
                if relative not in wanted:
                    continue
                kind = 'signature' if relative == entry['signature_path'] else 'payload'
                if (relative in copied or not member.isfile()
                        or member.size != entry[kind + '_size_bytes']):
                    raise ValueError('Duplicate or nonregular signed source member')
                digest = hashlib.sha256()
                with archive.extractfile(member) as source, (root / relative).open('wb') as destination:
                    for chunk in iter(lambda: source.read(1024 * 1024), b''):
                        digest.update(chunk)
                        destination.write(chunk)
                if digest.hexdigest() != entry[kind + '_sha256']:
                    raise ValueError('Signed source member checksum differs from lock')
                copied.add(relative)
        if copied != wanted:
            raise ValueError('Missing signed source member')
        copied_key = root / 'key.asc'
        shutil.copyfile(key, copied_key)
        if sha(copied_key) != entry['public_key_sha256']:
            raise ValueError('Copied Windows source signing key differs from lock')
        _run_gpg(gpg, home, env, '--import', copied_key.name)
        listing = _run_gpg(gpg, home, env, '--with-colons', '--fingerprint',
                           '--fingerprint', '--list-keys').stdout
        primaries, subkeys = _key_fingerprints(listing)
        signer = entry['signer_fingerprint']
        if primaries != [primary] or signer not in primaries + subkeys:
            raise ValueError('Reviewed source key fingerprint differs from lock')
        result = _run_gpg(gpg, home, env, '--status-fd', '1', '--verify',
                          entry['signature_path'], entry['payload_path'], timeout=120)
        _verified_signer(result.stdout, signer, primary)
    return {'method': 'offline_openpgp_detached_signature',
            'primary_fingerprint': primary, 'signer_fingerprint': signer,
            'public_key_sha256': entry['public_key_sha256'],
            'signature_sha256': entry['signature_sha256'],
            'payload_sha256': entry['payload_sha256']}
