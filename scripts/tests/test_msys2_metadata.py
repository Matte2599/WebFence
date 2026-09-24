import hashlib
import importlib.util
import io
import json
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
        self.notice_member = 'ucrt64/share/licenses/synthetic/LICENSE'
        self.notice_payload = b'Synthetic license notice fixture\n'
        self.required_notices = {self.notice_member: hashlib.sha256(self.notice_payload).hexdigest()}
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

    def inspect(self, required=None, notices=None):
        return metadata.inspect(self.archive, self.name, '1.2-3', self.required if required is None else required,
                                required_notices=notices)

    def test_binding_preserves_metadata_without_extracting_code(self):
        self.write(extras=[('install.sh', b'never execute')])
        record, raw = self.inspect()
        self.assertEqual(record['source_package'], 'mingw-w64-synthetic')
        self.assertEqual(record['pkgbuild_sha256'], 'a' * 64)
        self.assertEqual(record['sha256'], metadata.sha(self.archive))
        self.assertEqual(record['verified_dll_count'], 1)
        self.assertEqual(record['verified_notice_count'], 0)
        self.assertEqual(raw['.BUILDINFO'], self.build)
        self.assertFalse((self.archive.parent / 'install.sh').exists())

    def test_installed_notice_must_match_cached_binary_package(self):
        self.write(extras=[(self.notice_member, self.notice_payload)])
        record, _ = self.inspect(notices=self.required_notices)
        self.assertEqual(record['verified_notice_count'], 1)
        self.write(extras=[(self.notice_member, b'altered notice')])
        with self.assertRaisesRegex(ValueError, 'Installed notice differs'):
            self.inspect(notices=self.required_notices)
        self.write()
        with self.assertRaisesRegex(ValueError, 'Missing notice'):
            self.inspect(notices=self.required_notices)
        self.write(extras=[(self.notice_member, ('outside',))])
        with self.assertRaisesRegex(ValueError, 'nonregular'):
            self.inspect(notices=self.required_notices)
        self.write(extras=[(self.notice_member, self.notice_payload),
                           (self.notice_member, self.notice_payload)])
        with self.assertRaisesRegex(ValueError, 'Duplicate'):
            self.inspect(notices=self.required_notices)

    def test_published_binary_checksum_lock_rejects_unreviewed_archives(self):
        self.write()
        binary, _ = self.inspect()
        page = 'https://packages.msys2.org/packages/' + self.name
        entry = {'name': self.name, 'version': '1.2-3', 'sha256': binary['sha256'],
                 'source_page': page}
        lock_path = Path(self.temp.name) / 'binary-lock.json'
        data = {'schema_version': 1, 'reviewed_on': '2026-09-24', 'packages': [entry]}
        lock_path.write_text(json.dumps(data))
        lock = metadata.load_binary_lock(lock_path)
        self.assertEqual(metadata.verify_binary_lock(lock, self.name, '1.2-3', binary['sha256']), page)
        for owner, version, digest in [(self.name, '1.2-4', binary['sha256']),
                                       (self.name, '1.2-3', '0' * 64),
                                       (self.name + '-other', '1.2-3', binary['sha256'])]:
            with self.subTest(owner=owner, version=version, digest=digest), self.assertRaisesRegex(ValueError, 'Unreviewed'):
                metadata.verify_binary_lock(lock, owner, version, digest)
        data['packages'].append(dict(entry))
        lock_path.write_text(json.dumps(data))
        with self.assertRaisesRegex(ValueError, 'duplicate'):
            metadata.load_binary_lock(lock_path)
        data['packages'] = [dict(entry, source_page='https://example.invalid/unreviewed')]
        lock_path.write_text(json.dumps(data))
        with self.assertRaisesRegex(ValueError, 'Invalid'):
            metadata.load_binary_lock(lock_path)

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
