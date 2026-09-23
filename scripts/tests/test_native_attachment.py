import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('native_attachment', Path(__file__).parents[1] / 'attach-native-sources.py')
attachment = importlib.util.module_from_spec(spec)
spec.loader.exec_module(attachment)


class NativeAttachmentTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        archive = self.root / 'source.tar.gz'
        with tarfile.open(archive, 'w:gz') as output:
            for name, value in [('sample/LICENSE', b'synthetic terms'), ('sample/configure', b'never run')]:
                info = tarfile.TarInfo(name)
                info.size = len(value)
                output.addfile(info, io.BytesIO(value))
        self.inventory = self.root / 'native-build.json'
        self.inventory.write_text(json.dumps({'schema': 1, 'webfence_build': 'acquisition', 'packages': {
            'sample@1.0': {'upstream_archives': [{'downloadLocation': 'https://example.invalid/src',
                'checksums': [{'algorithm': 'SHA256', 'checksumValue': attachment.sources.sha(archive)}]}]}}}))
        self.materials = self.root / 'materials'
        self.manifest = attachment.sources.collect(self.inventory, self.materials, [archive])
        self.output = self.root / 'attached'

    def run_attach(self):
        return attachment.attach(self.inventory, self.materials, self.output)

    def test_regenerated_notices_bind_current_build_and_preserve_acquisition(self):
        old = self.inventory.read_bytes()
        data = json.loads(old)
        data['webfence_build'] = 'new build with unchanged native libraries'
        self.inventory.write_text(json.dumps(data))
        record = self.manifest['archives'][0]
        loose = self.materials / record['notice_root'] / 'sample/LICENSE'
        loose.write_text('modified loose file; do not copy')
        result = self.run_attach()
        self.assertEqual((self.output / record['notice_root'] / 'sample/LICENSE').read_bytes(), b'synthetic terms')
        self.assertFalse((self.output / 'archives').exists())
        self.assertFalse((self.output / record['notice_root'] / 'sample/configure').exists())
        self.assertEqual(result['current_native_build_sha256'], attachment.sources.sha(self.inventory))
        self.assertEqual((self.output / 'native-build.input.json').read_bytes(), old)
        self.assertFalse(result['distribution_ready'])
        self.assertFalse(result['corresponding_sources_complete'])
        self.assertEqual(result['notice_count'], 1)
        with self.assertRaises(FileExistsError):
            self.run_attach()
        self.assertEqual(json.loads((self.output / 'attachment.json').read_text()), result)

    def test_inventory_mismatch_does_not_create_output(self):
        original = self.inventory.read_text()
        for key, value in [('downloadLocation', 'https://example.invalid/changed'),
                           ('checksums', [{'algorithm': 'SHA256', 'checksumValue': '0' * 64}])]:
            data = json.loads(original)
            data['packages']['sample@1.0']['upstream_archives'][0][key] = value
            self.inventory.write_text(json.dumps(data))
            with self.assertRaisesRegex(ValueError, 'differ'):
                self.run_attach()
            self.assertFalse(self.output.exists())
        data = json.loads(original)
        data['packages']['sample@2.0'] = data['packages'].pop('sample@1.0')
        self.inventory.write_text(json.dumps(data))
        with self.assertRaisesRegex(ValueError, 'differ'):
            self.run_attach()

    def test_corruption_and_bounds_leave_no_partial_attachment(self):
        archive = self.materials / self.manifest['archives'][0]['archive']
        original = archive.read_bytes()
        archive.write_bytes(b'x' * len(original))
        with self.assertRaisesRegex(ValueError, 'checksum'):
            self.run_attach()
        archive.write_bytes(original)
        for constant in ['MAX_ARCHIVE', 'MAX_TOTAL_ARCHIVES', 'MAX_NOTICE', 'MAX_TOTAL_NOTICES']:
            with patch.object(attachment.sources, constant, 1), self.assertRaises(ValueError):
                self.run_attach()
        self.assertFalse(self.output.exists())
        self.assertEqual(list(self.root.glob('.webfence-notices-*')), [])

    def test_manifest_tampering_and_incomplete_collection(self):
        manifest_path = self.materials / 'source-materials.json'
        original = manifest_path.read_text()
        for change in ['input', 'notice', 'archive', 'ready', 'qt_reference', 'missing']:
            data = json.loads(original)
            if change == 'input':
                data['input_manifest_sha256'] = '0' * 64
            elif change == 'notice':
                data['archives'][0]['files'][0]['sha256'] = '0' * 64
            elif change == 'archive':
                data['archives'][0]['archive'] = '../source.tar.gz'
            elif change == 'ready':
                data['distribution_ready'] = True
            elif change == 'qt_reference':
                data['archives'][0]['qt_license_references'] = [{'attribution': 'forged', 'path': 'forged'}]
            else:
                data['archives'] = []
            manifest_path.write_text(json.dumps(data))
            with self.subTest(change=change), self.assertRaises(ValueError):
                self.run_attach()
            self.assertFalse(self.output.exists())
        manifest_path.write_text(original)
        (self.materials / 'INCOMPLETE').touch()
        with self.assertRaisesRegex(ValueError, 'incomplete'):
            self.run_attach()

    def test_source_links_are_not_followed(self):
        archive = self.materials / self.manifest['archives'][0]['archive']
        other = self.root / 'other.archive'
        archive.rename(other)
        try:
            archive.symlink_to(other)
        except OSError:
            self.skipTest('Symlink creation not available on this host')
        with self.assertRaisesRegex(ValueError, 'without links'):
            self.run_attach()
        self.assertFalse(self.output.exists())
        self.assertTrue(other.is_file())

    def test_qt_reference_ledger_is_regenerated_and_tampering_rejected(self):
        archive = self.root / 'qt.tar'
        with tarfile.open(archive, 'w') as output:
            for name, value in [('qt/terms.txt', b'synthetic referenced terms'),
                                ('qt/qt_attribution.json', b'{"LicenseFile":"terms.txt"}')]:
                info = tarfile.TarInfo(name)
                info.size = len(value)
                output.addfile(info, io.BytesIO(value))
        data = json.loads(self.inventory.read_text())
        data['packages']['sample@1.0']['upstream_archives'][0]['checksums'][0]['checksumValue'] = attachment.sources.sha(archive)
        self.inventory.write_text(json.dumps(data))
        self.materials = self.root / 'qt-materials'
        manifest = attachment.sources.collect(self.inventory, self.materials, [archive])
        path = self.materials / 'source-materials.json'
        original = path.read_text()
        manifest['archives'][0]['qt_license_references'][0]['path'] = 'qt/forged.txt'
        path.write_text(json.dumps(manifest))
        with self.assertRaisesRegex(ValueError, 'Regenerated'):
            self.run_attach()
        self.assertFalse(self.output.exists())
        path.write_text(original)
        result = self.run_attach()
        record = json.loads(original)['archives'][0]
        self.assertEqual((self.output / record['notice_root'] / 'qt/terms.txt').read_bytes(), b'synthetic referenced terms')
        self.assertEqual(result['notice_count'], 2)


if __name__ == '__main__':
    unittest.main()
