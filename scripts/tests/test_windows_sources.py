from contextlib import contextmanager
import hashlib
import io
import json
from pathlib import Path
import shutil
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parents[1]))
import collect_windows_sources as sources
sys.path.pop(0)


class WindowsSourcesTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.archive = self.root / 'synthetic.tar.gz'
        self.recipe = b'# Synthetic recipe, never execute\n'
        self.payload = b'synthetic upstream bytes'
        self.item = {'base': 'mingw-w64-synthetic', 'version': '1.2-3',
                     'pkgbuild_sha256': hashlib.sha256(self.recipe).hexdigest()}
        self.owner = 'mingw-w64-ucrt-x86_64-synthetic'
        self.inventory = {'schema': 1, 'packages': {self.owner: {'version': '1.2-3', 'binary_package': {
            'source_package': self.item['base'], 'version': '1.2-3', 'pkgbuild_sha256': self.item['pkgbuild_sha256']}}}}
        self.srcinfo = ('pkgbase = mingw-w64-synthetic\npkgver = 1.2\npkgrel = 3\n'
                        'source = https://example.invalid/upstream.tar\nsha256sums = ' + hashlib.sha256(self.payload).hexdigest() + '\n')

    def make(self, payload=None, srcinfo=None, extras=()):
        entries = [('PKGBUILD', self.recipe), ('.SRCINFO', (srcinfo or self.srcinfo).encode()),
                   ('upstream.tar', self.payload if payload is None else payload), *extras]
        with tarfile.open(self.archive, 'w:gz') as output:
            for path, data in entries:
                entry = tarfile.TarInfo(self.item['base'] + '/' + path)
                entry.size = len(data)
                output.addfile(entry, io.BytesIO(data))

    def test_recipe_and_source_checks_with_explicit_unverified_inputs(self):
        info = self.srcinfo + ('source = upstream.sig\nsha256sums = SKIP\n'
                              'source = git+https://example.invalid/repo.git#commit=abc\nsha256sums = SKIP\n')
        self.make(srcinfo=info, extras=[('upstream.sig', b'synthetic signature')])
        record, raw = sources.inspect_source(self.archive, self.item)
        self.assertEqual(raw['PKGBUILD'], self.recipe)
        self.assertEqual(record['files']['upstream.tar']['sha256'], hashlib.sha256(self.payload).hexdigest())
        self.assertEqual([r['status'] for r in record['source_checks']], ['verified', 'no_strong_checksum', 'vcs_unverified'])
        self.assertEqual(len(record['unverified_inputs']), 2)
        self.assertFalse((self.root / 'upstream.tar').exists())

    def test_wrong_recipe_source_identity_and_unsafe_entries(self):
        self.make()
        with self.assertRaisesRegex(ValueError, 'Recipe differs'):
            sources.inspect_source(self.archive, dict(self.item, pkgbuild_sha256='0' * 64))
        self.make(payload=b'altered upstream')
        with self.assertRaisesRegex(ValueError, 'checksum mismatch'):
            sources.inspect_source(self.archive, self.item)
        self.make(srcinfo=self.srcinfo.replace('pkgrel = 3', 'pkgrel = 4'))
        with self.assertRaisesRegex(ValueError, 'identity differs'):
            sources.inspect_source(self.archive, self.item)
        for extra in [('PKGBUILD', b'duplicate'), ('../escape', b'outside')]:
            self.make(extras=[extra])
            with self.assertRaises(ValueError):
                sources.inspect_source(self.archive, self.item)
        self.make()
        for constant in ('MAX_UNPACKED', 'MAX_METADATA'):
            with patch.object(sources, constant, 1), self.assertRaises(ValueError):
                sources.inspect_source(self.archive, self.item)

    def test_split_package_deduplication_and_conflicting_recipes(self):
        self.inventory['packages'][self.owner + '-extra'] = json.loads(json.dumps(self.inventory['packages'][self.owner]))
        plan = sources.source_plan(self.inventory)
        self.assertEqual(len(plan), 1)
        self.assertEqual(len(plan[0]['binary_packages']), 2)
        self.inventory['packages'][self.owner + '-extra']['binary_package']['pkgbuild_sha256'] = '0' * 64
        with self.assertRaisesRegex(ValueError, 'Conflicting'):
            sources.source_plan(self.inventory)

    def test_collection_publication_and_failure_marker(self):
        self.make()
        manifest = self.root / 'native.json'
        manifest.write_text(json.dumps(self.inventory))
        def fetch(url, path, limit):
            shutil.copyfile(self.archive, path)
        @contextmanager
        def fixture_tar(path, zstd):
            # Synthetic gzip transport; actual zstd is exercised by native
            # Windows packaging and the separately recorded real acquisition.
            with tarfile.open(path, mode='r|gz') as archive:
                yield archive
        with patch.object(sources.download, 'fetch', side_effect=fetch), patch.object(sources, 'tar_stream', fixture_tar):
            target = self.root / 'complete'
            result = sources.collect(manifest, target)
            self.assertFalse((target / 'INCOMPLETE').exists())
            self.assertFalse(result['distribution_ready'])
            self.assertFalse(result['corresponding_sources_complete'])
            self.assertTrue((target / result['archives'][0]['archive']).is_file())
            with self.assertRaises(FileExistsError):
                sources.collect(manifest, target)
            self.make(payload=b'wrong bytes')
            incomplete = self.root / 'incomplete'
            with self.assertRaises(ValueError):
                sources.collect(manifest, incomplete)
            self.assertTrue((incomplete / 'INCOMPLETE').exists())
            self.assertFalse((incomplete / 'source-materials.json').exists())


if __name__ == '__main__':
    unittest.main()
