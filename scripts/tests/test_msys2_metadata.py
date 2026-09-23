import hashlib
import importlib.util
import io
from pathlib import Path
import tarfile
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('msys2_metadata', Path(__file__).parents[1] / 'msys2_binary_metadata.py')
metadata = importlib.util.module_from_spec(spec)
spec.loader.exec_module(metadata)


class MSYS2MetadataTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.archive = Path(self.temp.name) / 'synthetic.pkg.tar.gz'
        self.name = 'mingw-w64-ucrt-x86_64-synthetic'
        self.member = 'ucrt64/bin/synthetic.dll'
        self.payload = b'Synthetic DLL fixture, never executable'
        self.required = {self.member: hashlib.sha256(self.payload).hexdigest()}
        self.pkg = (f'pkgname = {self.name}\npkgbase = mingw-w64-synthetic\npkgver = 1.2-3\narch = any\n').encode()
        self.build = (f'format = 2\npkgname = {self.name}\npkgname = {self.name}-debug\n'
                      'pkgbase = mingw-w64-synthetic\npkgver = 1.2-3\npkgarch = any\n'
                      'pkgbuild_sha256sum = ' + 'a' * 64 + '\n').encode()

    def write(self, extras=(), pkg=None, build=None, payload=None):
        entries = [('.PKGINFO', self.pkg if pkg is None else pkg),
                   ('.BUILDINFO', self.build if build is None else build),
                   (self.member, self.payload if payload is None else payload), *extras]
        with tarfile.open(self.archive, 'w:gz') as archive:
            for name, data in entries:
                entry = tarfile.TarInfo(name)
                if isinstance(data, tuple):
                    entry.type = tarfile.SYMTYPE
                    entry.linkname = data[0]
                    archive.addfile(entry)
                else:
                    entry.size = len(data)
                    archive.addfile(entry, io.BytesIO(data))

    def inspect(self, required=None):
        return metadata.inspect(self.archive, self.name, '1.2-3', self.required if required is None else required)

    def test_binding_preserves_metadata_without_extracting_code(self):
        self.write(extras=[('install.sh', b'never execute')])
        record, raw = self.inspect()
        self.assertEqual(record['source_package'], 'mingw-w64-synthetic')
        self.assertEqual(record['pkgbuild_sha256'], 'a' * 64)
        self.assertEqual(record['sha256'], metadata.sha(self.archive))
        self.assertEqual(record['verified_dll_count'], 1)
        self.assertEqual(raw['.BUILDINFO'], self.build)
        self.assertFalse((self.archive.parent / 'install.sh').exists())

    def test_changed_or_missing_dll_rejected(self):
        self.write(payload=b'changed after installation')
        with self.assertRaisesRegex(ValueError, 'Installed DLL differs'):
            self.inspect()
        self.write()
        with self.assertRaisesRegex(ValueError, 'Missing DLL'):
            self.inspect({'ucrt64/bin/absent.dll': '0' * 64})
        with self.assertRaises(ValueError):
            self.inspect({})

    def test_package_and_recipe_identity_rejected(self):
        variants = [self.build.replace(b'1.2-3', b'1.2-4'),
                    self.build.replace(b'format = 2', b'format = 1'),
                    self.build.replace(b'a' * 64, b'invalid'),
                    self.build + b'pkgver = 1.2-3\n',
                    self.build.replace(b'pkgbase = mingw-w64-synthetic', b'pkgbase = mingw-w64-other')]
        for variant in variants:
            with self.subTest(variant=variant):
                self.write(build=variant)
                with self.assertRaises(ValueError):
                    self.inspect()

    def test_unsafe_paths_links_duplicates_and_bounds(self):
        for extra in [[('../escape', b'x')], [('.PKGINFO', b'duplicate')], [(self.member, b'duplicate')]]:
            self.write(extras=extra)
            with self.assertRaises(ValueError):
                self.inspect()
        self.write(payload=('outside.dll',))
        with self.assertRaisesRegex(ValueError, 'nonregular'):
            self.inspect()
        self.write()
        for constant in ['MAX_ARCHIVE', 'MAX_UNPACKED', 'MAX_METADATA']:
            with patch.object(metadata, constant, 1), self.assertRaises(ValueError):
                self.inspect()
        self.assertFalse((self.archive.parent.parent / 'escape').exists())


if __name__ == '__main__':
    unittest.main()
