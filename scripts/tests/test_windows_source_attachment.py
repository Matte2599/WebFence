from contextlib import contextmanager
import hashlib
import json
from pathlib import Path
import shutil
import sys
import tarfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parents[1]))
import attach_windows_sources as attachment
import test_windows_sources as fixtures
sys.path.pop(0)


class WindowsSourceAttachmentTest(unittest.TestCase):
    def setUp(self):
        self.fixture = fixtures.WindowsSourcesTest()
        self.fixture.setUp()
        self.addCleanup(self.fixture.doCleanups)
        self.root = self.fixture.root
        self.fixture.make()
        self.current = self.root / 'current.json'
        self.current.write_text(json.dumps(self.fixture.inventory))
        self.materials = self.root / 'materials'
        @contextmanager
        def fixture_tar(path, zstd):
            with tarfile.open(path, mode='r|gz') as source:
                yield source
        def fetch(url, path, limit):
            shutil.copyfile(self.fixture.archive, path)
        self.patch_tar = patch.object(attachment.sources, 'tar_stream', fixture_tar)
        self.patch_tar.start()
        self.addCleanup(self.patch_tar.stop)
        with patch.object(attachment.sources.download, 'fetch', side_effect=fetch):
            self.manifest = attachment.sources.collect(self.current, self.materials)
        self.record = self.manifest['archives'][0]
        self.output = self.root / 'attached'

    def write_manifest(self):
        (self.materials / 'source-materials.json').write_text(json.dumps(self.manifest))

    def assert_no_publication(self):
        self.assertFalse(self.output.exists())
        self.assertFalse(list(self.root.glob('.webfence-windows-sources-*')))

    def test_original_archive_and_regenerated_recipes_with_distinct_provenance(self):
        # A later package can add provenance without changing the dependency plan.
        current = json.loads(self.current.read_text())
        current['build_revision'] = 'synthetic-new-build'
        self.current.write_text(json.dumps(current))
        (self.materials / self.record['recipe_directory'] / 'PKGBUILD').write_text('tampered loose recipe')
        result = attachment.attach(self.current, self.materials, self.output)
        self.assertEqual((self.output / self.record['recipe_directory'] / 'PKGBUILD').read_bytes(), self.fixture.recipe)
        self.assertEqual(attachment.sha(self.output / self.record['archive']), self.record['archive_sha256'])
        self.assertEqual((self.output / 'native-build.current.json').read_bytes(), self.current.read_bytes())
        self.assertNotEqual(result['current_native_build_sha256'], result['collection_input_sha256'])
        self.assertFalse(result['distribution_ready'])
        self.assertFalse(result['corresponding_sources_complete'])
        saved = (self.output / 'attachment.json').read_bytes()
        with self.assertRaises(FileExistsError):
            attachment.attach(self.current, self.materials, self.output)
        self.assertEqual((self.output / 'attachment.json').read_bytes(), saved)

    def test_mismatched_build_and_incomplete_collection_rejected(self):
        (self.materials / 'INCOMPLETE').touch()
        with self.assertRaisesRegex(ValueError, 'incomplete'):
            attachment.attach(self.current, self.materials, self.output)
        (self.materials / 'INCOMPLETE').unlink()
        current = json.loads(self.current.read_text())
        current['packages'][self.fixture.owner]['binary_package']['pkgbuild_sha256'] = '0' * 64
        self.current.write_text(json.dumps(current))
        with self.assertRaisesRegex(ValueError, 'differ from current'):
            attachment.attach(self.current, self.materials, self.output)
        self.assert_no_publication()

    def test_corrupt_archive_and_forged_collection_evidence_rejected(self):
        archive = self.materials / self.record['archive']
        original = archive.read_bytes()
        archive.write_bytes(bytes([original[0] ^ 1]) + original[1:])
        with self.assertRaisesRegex(ValueError, 'checksum mismatch'):
            attachment.attach(self.current, self.materials, self.output)
        archive.write_bytes(original)
        self.record['files']['upstream.tar']['sha256'] = '0' * 64
        self.write_manifest()
        with self.assertRaisesRegex(ValueError, 'Regenerated source evidence'):
            attachment.attach(self.current, self.materials, self.output)
        self.assert_no_publication()

    def test_path_size_and_provenance_rejected(self):
        self.record['archive'] = '../escape'
        self.write_manifest()
        with self.assertRaisesRegex(ValueError, 'location'):
            attachment.attach(self.current, self.materials, self.output)
        self.record['archive'] = 'archives/' + self.fixture.item['base'] + '-1.2-3.src.tar.zst'
        self.write_manifest()
        with patch.object(attachment.sources, 'MAX_TOTAL', 1), self.assertRaisesRegex(ValueError, 'size budget'):
            attachment.attach(self.current, self.materials, self.output)
        self.manifest['input_manifest_sha256'] = '0' * 64
        self.write_manifest()
        with self.assertRaisesRegex(ValueError, 'provenance'):
            attachment.attach(self.current, self.materials, self.output)
        self.assert_no_publication()

    def test_links_rejected_when_os_allows_symlinks(self):
        archive = self.materials / self.record['archive']
        saved = self.root / 'real-archive'
        archive.rename(saved)
        try:
            archive.symlink_to(saved)
        except OSError as error:
            self.skipTest('OS does not permit synthetic symlink: ' + str(error))
        with self.assertRaisesRegex(ValueError, 'without links'):
            attachment.attach(self.current, self.materials, self.output)
        self.assert_no_publication()


if __name__ == '__main__':
    unittest.main()
