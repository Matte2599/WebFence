import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('native_sources', Path(__file__).parents[1] / 'collect-native-sources.py')
sources = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sources)


class NativeSourcesTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.archive = self.root / 'source.tar.gz'

    def make_archive(self, entries):
        with tarfile.open(self.archive, 'w:gz') as output:
            for name, value in entries:
                info = tarfile.TarInfo(name)
                if isinstance(value, tuple):
                    info.type = tarfile.SYMTYPE
                    info.linkname = value[0]
                    output.addfile(info)
                else:
                    info.size = len(value)
                    output.addfile(info, io.BytesIO(value))

    def inventory(self):
        return {'schema': 1, 'packages': {'sample@1.0': {'upstream_archives': [{
            'downloadLocation': 'https://source.example.invalid/source.tar.gz',
            'checksums': [{'algorithm': 'SHA256', 'checksumValue': sources.sha(self.archive)}]}]}}}

    def write_inventory(self, data=None):
        p = self.root / 'native-build.json'
        p.write_text(json.dumps(data or self.inventory()))
        return p

    def test_exact_source_notices_and_links_without_executing_or_following(self):
        self.make_archive([('sample/LICENSES/LGPL.txt', b'synthetic license'),
                           ('sample/COPYING', ('LICENSES/LGPL.txt',)),
                           ('sample/configure', b'never execute this'),
                           ('sample/thirdparty/qt_attribution.json', b'{}')])
        target = self.root / 'materials'
        with patch.object(sources, 'fetch', side_effect=AssertionError('unexpected network')):
            result = sources.collect(self.write_inventory(), target, [self.archive])
        item = result['archives'][0]
        self.assertEqual((target / item['archive']).read_bytes(), self.archive.read_bytes())
        notices = target / item['notice_root']
        self.assertEqual((notices / 'sample/LICENSES/LGPL.txt').read_bytes(), b'synthetic license')
        self.assertFalse((notices / 'sample/COPYING').exists())
        self.assertFalse((notices / 'sample/configure').exists())
        self.assertEqual(item['archive_links'][0]['target'], 'LICENSES/LGPL.txt')
        self.assertFalse(result['distribution_ready'])
        self.assertFalse(result['corresponding_sources_complete'])
        self.assertFalse((target / 'INCOMPLETE').exists())
        with self.assertRaises(FileExistsError):
            sources.collect(self.write_inventory(), target, [self.archive])
        self.assertTrue((target / 'source-materials.json').exists())

    def test_wrong_download_hash_does_not_publish_success(self):
        self.make_archive([('sample/LICENSE', b'synthetic')])
        manifest = self.write_inventory()
        def corrupt(url, path, limit):
            path.write_bytes(b'not the expected source')
        target = self.root / 'materials'
        with patch.object(sources, 'fetch', side_effect=corrupt), self.assertRaisesRegex(ValueError, 'checksum'):
            sources.collect(manifest, target)
        self.assertTrue((target / 'INCOMPLETE').exists())
        self.assertFalse((target / 'source-materials.json').exists())
        self.assertEqual(list((target / 'archives').glob('*.archive')), [])

    def test_traversal_and_duplicate_notices_rejected(self):
        for entries in [[('../LICENSE', b'x')], [('/LICENSE', b'x')],
                        [('sample/LICENSE', b'a'), ('sample/LICENSE', b'b')]]:
            with self.subTest(entries=entries):
                self.make_archive(entries)
                with tempfile.TemporaryDirectory(dir=self.root) as target:
                    with self.assertRaises(ValueError):
                        sources.collect_notices(self.archive, Path(target))
        self.assertFalse((self.root / 'LICENSE').exists())

    def test_notice_and_unpacked_budgets(self):
        self.make_archive([('sample/LICENSE', b'12345')])
        for constant in ['MAX_NOTICE', 'MAX_TOTAL_NOTICES', 'MAX_UNPACKED']:
            with self.subTest(constant=constant), patch.object(sources, constant, 4):
                with self.assertRaises(ValueError):
                    sources.collect_notices(self.archive, self.root / constant)

    def test_referenced_notices_in_nonstandard_filenames(self):
        self.make_archive([('sample/LICENSE.TXT', b'See docs/FTL.TXT'),
                           ('sample/docs/FTL.TXT', b'synthetic referenced terms')])
        out = self.root / 'notices'
        sources.collect_notices(self.archive, out, {'docs/FTL.TXT'})
        self.assertEqual((out / 'sample/docs/FTL.TXT').read_bytes(), b'synthetic referenced terms')
        with self.assertRaisesRegex(ValueError, 'Missing referenced'):
            sources.collect_notices(self.archive, self.root / 'missing', {'absent/terms.txt'})

    def test_bad_manifest_fails_before_creating_output(self):
        self.make_archive([('sample/LICENSE', b'x')])
        for value in ['http://example.invalid/src', 'https://user:secret@example.invalid/src', 'file:///tmp/src']:
            data = self.inventory()
            data['packages']['sample@1.0']['upstream_archives'][0]['downloadLocation'] = value
            with self.subTest(value=value), self.assertRaises(ValueError):
                sources.collect(self.write_inventory(data), self.root / 'absent')
            self.assertFalse((self.root / 'absent').exists())
        data = self.inventory()
        data['packages']['sample@1.0']['upstream_archives'][0]['checksums'] = []
        with self.assertRaises(ValueError):
            sources.archive_plan(data)

    def test_curl_version_and_transfer_policy(self):
        with patch.object(sources.subprocess, 'check_output', return_value='curl 8.3.0\n'), self.assertRaises(ValueError):
            sources.fetch('https://example.invalid/src', self.root / 'out', 123)
        with patch.object(sources.subprocess, 'check_output', return_value='curl 8.4.0\n'), patch.object(sources.subprocess, 'run') as run:
            sources.fetch('https://example.invalid/src', self.root / 'out', 123)
        args = run.call_args.args[0]
        self.assertEqual(args[args.index('--proto-redir') + 1], '=https')
        self.assertEqual(args[args.index('--max-filesize') + 1], '123')
        self.assertEqual(run.call_args.kwargs['timeout'], 190)


if __name__ == '__main__':
    unittest.main()
