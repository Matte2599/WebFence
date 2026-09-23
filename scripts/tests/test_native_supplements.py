import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parents[1]))
import native_supplements as supplements
sys.path.pop(0)


class NativeSupplementsTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.recipe = self.root / 'homebrew/sample@1.0/.brew/sample.rb'
        self.recipe.parent.mkdir(parents=True)
        self.recipe.write_bytes(b'# Synthetic recipe; never execute\n')
        self.native = self.root / 'native-build.json'
        self.native.write_text(json.dumps({'schema': 1, 'packages': {'sample@1.0': {
            'formula': 'sample', 'version': '1.0', 'metadata_files': ['.brew/sample.rb']}}}))
        self.payload = b'synthetic patch, never execute'
        self.plan = {'schema': 1, 'packages': {'sample@1.0': {
            'installed_recipe_sha256': supplements.download.sha(self.recipe), 'files': [{
                'path': 'patch.diff', 'url': 'https://example.invalid/patch.diff',
                'size_bytes': len(self.payload), 'sha256': hashlib.sha256(self.payload).hexdigest()}]}}}
        self.plan_path = self.root / 'plan.json'
        self.save_plan()
        self.output = self.root / 'output'
        self.reuse = self.root / 'reuse'
        (self.reuse / 'sample@1.0').mkdir(parents=True)
        (self.reuse / 'sample@1.0/patch.diff').write_bytes(self.payload)

    def save_plan(self):
        self.plan_path.write_text(json.dumps(self.plan))

    def test_verified_reuse_provenance_and_existing_output_preserved(self):
        with patch.object(supplements.download, 'fetch', side_effect=AssertionError('unexpected network')):
            result = supplements.collect(self.native, self.plan_path, self.output, self.reuse)
        self.assertEqual((self.output / 'sample@1.0/patch.diff').read_bytes(), self.payload)
        self.assertEqual((self.output / 'sample@1.0/installed-recipe.rb').read_bytes(), self.recipe.read_bytes())
        self.assertEqual(result['native_build_sha256'], supplements.download.sha(self.native))
        self.assertEqual(result['plan_sha256'], supplements.download.sha(self.plan_path))
        self.assertFalse(result['distribution_ready'])
        self.assertFalse(result['corresponding_sources_complete'])
        with self.assertRaises(FileExistsError):
            supplements.collect(self.native, self.plan_path, self.output, self.reuse)
        self.assertEqual((self.output / 'sample@1.0/patch.diff').read_bytes(), self.payload)

    def test_recipe_mismatch_prevents_download(self):
        self.recipe.write_text('changed recipe')
        with patch.object(supplements.download, 'fetch', side_effect=AssertionError('unexpected network')):
            with self.assertRaisesRegex(ValueError, 'recipe differs'):
                supplements.collect(self.native, self.plan_path, self.output)
        self.assertFalse(self.output.exists())

    def test_linked_material_rejected_when_os_allows_symlinks(self):
        material = self.reuse / 'sample@1.0/patch.diff'
        original = self.root / 'original.diff'
        material.rename(original)
        try:
            material.symlink_to(original)
        except OSError as error:
            self.skipTest('OS does not permit synthetic symlink: ' + str(error))
        with self.assertRaisesRegex(ValueError, 'without links'):
            supplements.collect(self.native, self.plan_path, self.output, self.reuse)
        self.assertFalse(self.output.exists())
        self.assertFalse(list(self.root.glob('.webfence-supplements-*')))

    def test_verified_download_and_corruption_cleanup(self):
        def fetch(url, path, limit):
            self.assertEqual(limit, len(self.payload))
            path.write_bytes(self.payload)
        with patch.object(supplements.download, 'fetch', side_effect=fetch):
            supplements.collect(self.native, self.plan_path, self.output)
        def corrupt(url, path, limit):
            path.write_bytes(b'x' * len(self.payload))
        with patch.object(supplements.download, 'fetch', side_effect=corrupt):
            with self.assertRaisesRegex(ValueError, 'checksum mismatch'):
                supplements.collect(self.native, self.plan_path, self.root / 'failed')
        self.assertFalse((self.root / 'failed').exists())
        self.assertFalse(list(self.root.glob('.webfence-supplements-*')))

    def test_unsafe_paths_urls_duplicates_and_budget(self):
        entry = self.plan['packages']['sample@1.0']['files'][0]
        for key, value in [('path', '../outside'), ('url', 'http://example.invalid/patch'),
                           ('url', 'https://user:password@example.invalid/patch'), ('size_bytes', 0)]:
            original = entry[key]
            entry[key] = value
            self.save_plan()
            with self.assertRaises(ValueError):
                supplements.collect(self.native, self.plan_path, self.output, self.reuse)
            entry[key] = original
        self.plan['packages']['sample@1.0']['files'].append(dict(entry))
        self.save_plan()
        with self.assertRaisesRegex(ValueError, 'duplicate'):
            supplements.collect(self.native, self.plan_path, self.output, self.reuse)
        self.plan['packages']['sample@1.0']['files'].pop()
        self.save_plan()
        with patch.object(supplements, 'MAX_TOTAL', 1), self.assertRaisesRegex(ValueError, 'budget'):
            supplements.collect(self.native, self.plan_path, self.output, self.reuse)
        self.assertFalse(self.output.exists())


if __name__ == '__main__':
    unittest.main()
