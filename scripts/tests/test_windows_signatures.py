"""Check pinned signature material and detached verification with synthetic data."""
import hashlib
import io
import json
from pathlib import Path
import shutil
import sys
import tarfile
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).parents[1]))
import verify_windows_signatures as signatures
sys.path.pop(0)


REPOSITORY = Path(__file__).resolve().parents[2]
FIXTURE = Path(__file__).resolve().parent / 'fixtures/windows-signature'
FINGERPRINT = '5847BE95BEDEA53971F0A6F456619F52F3482689'


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


class SignatureLockTest(unittest.TestCase):
    def test_reviewed_lock_and_public_keys(self):
        root = REPOSITORY / 'packaging/windows'
        locked = signatures.load_lock(root / 'source-signature-lock.json')
        self.assertEqual(set(locked), signatures.EXPECTED_PACKAGES)
        for entry in locked.values():
            key = root / entry['public_key_file']
            self.assertEqual(signatures.sha(key), entry['public_key_sha256'])

    def test_rejects_missing_package_and_unsafe_key_path(self):
        original = json.loads((REPOSITORY / 'packaging/windows/source-signature-lock.json').read_text())
        with tempfile.TemporaryDirectory() as temporary:
            lock = Path(temporary) / 'lock.json'
            lock.write_text(json.dumps(dict(original, entries=original['entries'][1:])))
            with self.assertRaisesRegex(ValueError, 'Incomplete'):
                signatures.load_lock(lock)
            changed = json.loads(json.dumps(original))
            changed['entries'][0]['public_key_file'] = '../outside.asc'
            lock.write_text(json.dumps(changed))
            with self.assertRaisesRegex(ValueError, 'Invalid'):
                signatures.load_lock(lock)

    def test_validsig_must_match_primary_and_signing_subkey(self):
        status = ('[GNUPG:] GOODSIG 021DE40BFB63B406 fixture\n'
                  '[GNUPG:] VALIDSIG BACF71F10404D5761C09D392021DE40BFB63B406 '
                  '2026-08-31 1788184599 0 4 0 1 8 00 A95536204A3BB489715231282A98E77EB6F24CA8\n')
        signatures._verified_signer(status, 'BACF71F10404D5761C09D392021DE40BFB63B406',
                                     'A95536204A3BB489715231282A98E77EB6F24CA8')
        with self.assertRaisesRegex(ValueError, 'signer'):
            signatures._verified_signer(status, 'BACF71F10404D5761C09D392021DE40BFB63B406',
                                         '0' * 40)
        with self.assertRaisesRegex(ValueError, 'signer'):
            signatures._verified_signer(status + '[GNUPG:] BADSIG 021DE40BFB63B406 fixture\n',
                                         'BACF71F10404D5761C09D392021DE40BFB63B406',
                                         'A95536204A3BB489715231282A98E77EB6F24CA8')


@unittest.skipUnless(shutil.which('gpg'), 'GnuPG is required for the detached-signature fixture')
class DetachedSignatureTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix='webfence-signature-test-')
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.base = 'mingw-w64-zlib'
        self.payload = (FIXTURE / 'payload.txt').read_bytes()
        self.signature = (FIXTURE / 'payload.txt.asc').read_bytes()
        key = self.root / 'source-signing-keys' / (FINGERPRINT + '.asc')
        key.parent.mkdir()
        shutil.copyfile(FIXTURE / 'public-key.asc', key)
        self.item = {'base': self.base, 'version': '1.0-1', 'pkgbuild_sha256': 'a' * 64}
        self.entry = {'source_package': self.base, 'version': '1.0-1',
                      'pkgbuild_sha256': 'a' * 64,
                      'signature_source': 'https://example.invalid/payload.txt.asc',
                      'signature_path': 'payload.txt.asc', 'signature_sha256': digest(self.signature),
                      'signature_size_bytes': len(self.signature),
                      'payload_source': 'https://example.invalid/payload.txt',
                      'payload_path': 'payload.txt', 'payload_sha256': digest(self.payload),
                      'payload_size_bytes': len(self.payload),
                      'primary_fingerprint': FINGERPRINT, 'signer_fingerprint': FINGERPRINT,
                      'public_key_file': 'source-signing-keys/' + FINGERPRINT + '.asc',
                      'public_key_sha256': signatures.sha(key)}
        self.srcinfo = {'validpgpkeys': [FINGERPRINT]}
        self.checks = [
            {'source': self.entry['signature_source'], 'path': 'payload.txt.asc',
             'status': 'no_strong_checksum', 'checksums_verified': []},
            {'source': self.entry['payload_source'], 'path': 'payload.txt',
             'status': 'verified', 'checksums_verified': ['sha256']},
        ]
        self.archive = self.root / 'source.tar.gz'
        self.make_archive()

    def make_archive(self):
        with tarfile.open(self.archive, 'w:gz') as output:
            for name, raw in [('payload.txt', self.payload), ('payload.txt.asc', self.signature)]:
                member = tarfile.TarInfo(self.base + '/' + name)
                member.size = len(raw)
                output.addfile(member, io.BytesIO(raw))
        self.entry['archive_sha256'] = signatures.sha(self.archive)
        self.files = {name: {'size_bytes': len(raw), 'hashes': {'sha256': digest(raw)}}
                      for name, raw in [('payload.txt', self.payload),
                                        ('payload.txt.asc', self.signature)]}

    def verify(self):
        return signatures.verify_signature(self.archive, self.item, self.files,
                                           self.srcinfo, self.checks, self.entry,
                                           key_root=self.root)

    def test_good_signature_and_tampered_payload(self):
        self.assertEqual(self.verify()['signer_fingerprint'], FINGERPRINT)
        self.payload += b'changed\n'
        self.entry['payload_sha256'] = digest(self.payload)
        self.entry['payload_size_bytes'] = len(self.payload)
        self.make_archive()
        with self.assertRaisesRegex(ValueError, 'verification failed'):
            self.verify()

    def test_changed_key_or_recipe_is_rejected_before_gpg(self):
        self.entry['public_key_sha256'] = '0' * 64
        with self.assertRaisesRegex(ValueError, 'key'):
            self.verify()
        self.entry['public_key_sha256'] = signatures.sha(
            self.root / self.entry['public_key_file'])
        self.item['pkgbuild_sha256'] = 'b' * 64
        with self.assertRaisesRegex(ValueError, 'archive or recipe'):
            self.verify()


if __name__ == '__main__':
    unittest.main()
