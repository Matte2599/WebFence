import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest
from unittest.mock import patch
import subprocess

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

    def test_qt_references_include_nonstandard_files_before_and_after_metadata(self):
        # Order must not matter; shared parent references are ordinary Qt data.
        self.make_archive([
            ('sample/shared/terms.txt', b'synthetic parent terms'),
            ('sample/shared/sub/qt_attribution.json',
             b'[{"LicenseFile":"../terms.txt","Description":"line one\nline two"},'
             b'{"LicenseFiles":["named-terms.txt","../terms.txt"]}]'),
            ('sample/shared/sub/named-terms.txt', b'synthetic local terms'),
            ('sample/unused.c', b'not a referenced notice')])
        out = self.root / 'notices'
        record = sources.collect_notices(self.archive, out)
        self.assertEqual((out / 'sample/shared/terms.txt').read_bytes(), b'synthetic parent terms')
        self.assertEqual((out / 'sample/shared/sub/named-terms.txt').read_bytes(), b'synthetic local terms')
        self.assertFalse((out / 'sample/unused.c').exists())
        self.assertEqual(len(record['qt_license_references']), 2)
        self.assertEqual(len(record['files']), 3)

    def test_qt_missing_linked_and_duplicate_references_rejected(self):
        metadata = ('sample/sub/qt_attribution.json', b'{"LicenseFile":"terms.txt"}')
        for other in ([], [('sample/sub/terms.txt', ('../../outside',))],
                      [('sample/sub/terms.txt', b'a'), ('sample/sub/terms.txt', b'b')],
                      [metadata, ('sample/sub/terms.txt', b'a')]):
            self.make_archive([metadata, *other])
            with self.subTest(other=other), tempfile.TemporaryDirectory(dir=self.root) as out:
                with self.assertRaises(ValueError):
                    sources.collect_notices(self.archive, Path(out))

    def test_qt_unsafe_reference_paths_and_metadata_rejected(self):
        for value in ('../../../outside', '/tmp/outside', 'C:/outside',
                      '..\\outside', 'x\nfile', '', 12):
            self.make_archive([('sample/sub/qt_attribution.json',
                                json.dumps({'LicenseFile': value}).encode())])
            with self.subTest(value=value), self.assertRaises(ValueError):
                sources.collect_notices(self.archive, self.root / 'notices')
        for content in (b'{invalid', b'[null]', b'{"LicenseFiles":true}'):
            self.make_archive([('sample/qt_attribution.json', content)])
            with self.subTest(content=content), self.assertRaises(ValueError):
                sources.collect_notices(self.archive, self.root / 'notices')
        self.assertFalse((self.root / 'notices').exists())

    def test_qt_reference_count_and_metadata_budgets(self):
        self.make_archive([('sample/qt_attribution.json', b'{"LicenseFiles":["one","two"]}')])
        with patch.object(sources, 'MAX_QT_REFERENCES', 1), self.assertRaisesRegex(ValueError, 'reference budget'):
            sources.collect_notices(self.archive, self.root / 'notices')
        with patch.object(sources, 'MAX_NOTICE', 4), self.assertRaisesRegex(ValueError, 'attribution size'):
            sources.collect_notices(self.archive, self.root / 'notices')
        self.assertFalse((self.root / 'notices').exists())

    def test_curl_version_and_transfer_policy(self):
        with patch.object(sources.subprocess, 'check_output', return_value='curl 8.3.0\n'), self.assertRaises(ValueError):
            sources.fetch('https://example.invalid/src', self.root / 'out', 123)
        with patch.object(sources.subprocess, 'check_output', return_value='curl 8.4.0\n'), patch.object(sources.subprocess, 'run') as run:
            sources.fetch('https://example.invalid/src', self.root / 'out', 123)
        args = run.call_args.args[0]
        self.assertEqual(args[args.index('--proto-redir') + 1], '=https')
        self.assertEqual(args[args.index('--max-filesize') + 1], '123')
        self.assertEqual(run.call_args.kwargs['timeout'], 190)

    def test_gnu_mirror_failure_retries_only_official_host(self):
        url = 'https://ftpmirror.gnu.org/gnu/gettext/gettext-1.0.tar.gz'
        official = 'https://ftp.gnu.org/gnu/gettext/gettext-1.0.tar.gz'
        with patch.object(sources.subprocess, 'check_output', return_value='curl 8.4.0\n'), \
                patch.object(sources.subprocess, 'run', side_effect=[
                    subprocess.CalledProcessError(22, ['curl']), None]) as run:
            self.assertEqual(sources.fetch(url, self.root / 'out', 123), official)
        self.assertEqual(run.call_count, 2)
        self.assertEqual(run.call_args_list[0].args[0][-1], url)
        self.assertEqual(run.call_args_list[1].args[0][-1], official)
        with patch.object(sources.subprocess, 'check_output', return_value='curl 8.4.0\n'), \
                patch.object(sources.subprocess, 'run', side_effect=subprocess.CalledProcessError(22, ['curl'])) as run:
            with self.assertRaises(subprocess.CalledProcessError):
                sources.fetch('https://example.invalid/src', self.root / 'out', 123)
        self.assertEqual(run.call_count, 1)

    def test_download_record_preserves_inventory_and_actual_acquisition_url(self):
        self.make_archive([('sample/LICENSE', b'synthetic license')])
        inventory = self.inventory()
        url = 'https://ftpmirror.gnu.org/gnu/sample/source.tar.gz'
        official = 'https://ftp.gnu.org/gnu/sample/source.tar.gz'
        inventory['packages']['sample@1.0']['upstream_archives'][0]['downloadLocation'] = url
        def copy_download(requested, destination, limit):
            self.assertEqual(requested, url)
            destination.write_bytes(self.archive.read_bytes())
            return official
        with patch.object(sources, 'fetch', side_effect=copy_download):
            record = sources.collect(self.write_inventory(inventory), self.root / 'materials')
        self.assertEqual(record['archives'][0]['url'], url)
        self.assertEqual(record['archives'][0]['acquisition_url'], official)


if __name__ == '__main__':
    unittest.main()
