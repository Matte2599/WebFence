"""Exercise the reviewed binary signature set and a tampered synthetic package."""
import json
from pathlib import Path
import shutil
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).parents[1]))
from msys2_binary_metadata import load_binary_lock, sha
import verify_windows_binary_signatures as signatures
sys.path.pop(0)


REPOSITORY = Path(__file__).resolve().parents[2]
ROOT = REPOSITORY / 'packaging/windows'
FIXTURE = Path(__file__).resolve().parent / 'fixtures/windows-signature'
FIXTURE_FINGERPRINT = '5847BE95BEDEA53971F0A6F456619F52F3482689'


class BinarySignatureLockTest(unittest.TestCase):
    def setUp(self):
        self.binary = load_binary_lock(ROOT / 'msys2-binary-lock.json')
        self.path = ROOT / 'msys2-binary-signature-lock.json'

    def test_all_reviewed_packages_and_bytes(self):
        key, entries = signatures.load_lock(self.path, self.binary)
        self.assertEqual(len(entries), 22)
        self.assertEqual(key['fingerprint'], '5F944B027F7FE2091985AA2EFA11531AA0AA7F57')
        self.assertEqual(sha(ROOT / key['public_key_file']), key['public_key_sha256'])
        for entry in entries.values():
            signature = ROOT / entry['signature_file']
            self.assertEqual(sha(signature), entry['signature_sha256'])
            self.assertEqual(signature.stat().st_size, entry['signature_size_bytes'])

    def test_missing_or_changed_package_is_rejected(self):
        original = json.loads(self.path.read_text())
        with tempfile.TemporaryDirectory() as temporary:
            candidate = Path(temporary) / 'lock.json'
            candidate.write_text(json.dumps(dict(original, packages=original['packages'][1:])))
            with self.assertRaisesRegex(ValueError, 'Incomplete'):
                signatures.load_lock(candidate, self.binary)
            changed = json.loads(json.dumps(original))
            changed['packages'][0]['signature_file'] = '../other.sig'
            candidate.write_text(json.dumps(changed))
            with self.assertRaisesRegex(ValueError, 'location'):
                signatures.load_lock(candidate, self.binary)
            changed = json.loads(json.dumps(original))
            changed['packages'][0]['archive_sha256'] = '0' * 64
            candidate.write_text(json.dumps(changed))
            with self.assertRaisesRegex(ValueError, 'Unreviewed'):
                signatures.load_lock(candidate, self.binary)


@unittest.skipUnless(shutil.which('gpg'), 'GnuPG is required for the package fixture')
class BinarySignatureVerificationTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix='webfence-binary-signature-test-')
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.archive = self.root / 'synthetic.pkg.tar.zst'
        shutil.copyfile(FIXTURE / 'payload.txt', self.archive)
        key = self.root / 'binary-signing-keys' / (FIXTURE_FINGERPRINT + '.asc')
        key.parent.mkdir()
        shutil.copyfile(FIXTURE / 'public-key.asc', key)
        sig = self.root / 'binary-package-signatures/synthetic.pkg.tar.zst.sig'
        sig.parent.mkdir()
        shutil.copyfile(FIXTURE / 'payload.txt.asc', sig)
        self.key = {'fingerprint': FIXTURE_FINGERPRINT,
                    'public_key_file': key.relative_to(self.root).as_posix(),
                    'public_key_sha256': sha(key)}
        self.entry = {'archive_sha256': sha(self.archive),
                      'signature_file': sig.relative_to(self.root).as_posix(),
                      'signature_sha256': sha(sig),
                      'signature_size_bytes': sig.stat().st_size}

    def test_good_signature_then_changed_payload(self):
        result = signatures.verify_signature(self.archive, self.entry, self.key,
                                             key_root=self.root)
        self.assertEqual(result['signer_fingerprint'], FIXTURE_FINGERPRINT)
        self.archive.write_bytes(self.archive.read_bytes() + b'changed\n')
        self.entry['archive_sha256'] = sha(self.archive)
        with self.assertRaisesRegex(ValueError, 'verification failed'):
            signatures.verify_signature(self.archive, self.entry, self.key,
                                        key_root=self.root)

    def test_changed_key_or_signature_is_rejected(self):
        self.key['public_key_sha256'] = '0' * 64
        with self.assertRaisesRegex(ValueError, 'key'):
            signatures.verify_signature(self.archive, self.entry, self.key,
                                        key_root=self.root)
        self.key['public_key_sha256'] = sha(self.root / self.key['public_key_file'])
        self.entry['signature_sha256'] = '0' * 64
        with self.assertRaisesRegex(ValueError, 'signature'):
            signatures.verify_signature(self.archive, self.entry, self.key,
                                        key_root=self.root)


if __name__ == '__main__':
    unittest.main()
