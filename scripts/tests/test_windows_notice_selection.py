"""Keep archive-linked Windows attribution sidecars in development ZIPs."""
import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import unittest


SCRIPTS = Path(__file__).parents[1]
sys.path.insert(0, str(SCRIPTS))
spec = importlib.util.spec_from_file_location('windows_package', SCRIPTS / 'package-windows.py')
windows_package = importlib.util.module_from_spec(spec)
spec.loader.exec_module(windows_package)


class WindowsNoticeSelectionTest(unittest.TestCase):
    def test_selects_installed_attributions_and_their_license_sidecars(self):
        selected = (
            'doc/pcre2/AUTHORS.md',
            'qt6/wayland/protocols/MIT_LICENSE.txt',
            'qt6/wayland/protocols/appmenu/LGPL-2.1-or-later.txt',
            'qt6/wayland/protocols/appmenu/qt_attribution.json',
            'qt6/wayland/protocols/appmenu/REUSE.toml',
            'qt6/wayland/protocols/wayland/README',
            'licenses/icu/LICENSE',
        )
        for relative in selected:
            with self.subTest(relative=relative):
                self.assertEqual(str(windows_package.notice_relative('/ucrt64/share/' + relative)), relative)

    def test_excludes_code_build_files_and_unsafe_paths(self):
        excluded = (
            '/ucrt64/share/qt6/wayland/protocols/appmenu/appmenu.xml',
            '/ucrt64/lib/cmake/Qt6/3rdparty/kwin/qt_attribution.json',
            '/ucrt64/share/qt6/wayland/protocols/appmenu/',
            '/ucrt64/share/../lib/evil/qt_attribution.json',
            '/ucrt64/share/qt6\\wayland\\protocols\\REUSE.toml',
            '/usr/share/doc/pcre2/AUTHORS.md',
        )
        for name in excluded:
            with self.subTest(name=name):
                self.assertIsNone(windows_package.notice_relative(name))

    def test_qt_attribution_requires_selected_referenced_license(self):
        root = 'ucrt64/share/qt6/wayland/protocols/'
        attribution = root + 'appmenu/qt_attribution.json'
        license_text = root + 'MIT_LICENSE.txt'
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / 'qt_attribution.json'
            license_path = Path(directory) / 'MIT_LICENSE.txt'
            license_path.write_text('Synthetic license fixture\n')
            source.write_text(json.dumps({'LicenseFile': '../MIT_LICENSE.txt'}))
            files = {attribution: source, license_text: license_path}
            self.assertEqual(windows_package.qt_license_references(files), 1)
            with self.assertRaisesRegex(ValueError, 'Missing selected Qt license text'):
                windows_package.qt_license_references({attribution: source})
            for value in ('../../../../../../outside.txt', '/etc/passwd', '..\\MIT_LICENSE.txt', 123):
                with self.subTest(value=value):
                    source.write_text(json.dumps({'LicenseFile': value}))
                    with self.assertRaises(ValueError):
                        windows_package.qt_license_references(files)
            source.write_text(json.dumps({'LicenseId': 'MIT'}))
            with self.assertRaisesRegex(ValueError, 'no license text reference'):
                windows_package.qt_license_references(files)

    def test_qt_attribution_rejects_invalid_reference_structure(self):
        member = 'ucrt64/share/qt6/wayland/protocols/wayland/qt_attribution.json'
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / 'qt_attribution.json'
            for data in ([], ['not an object'], {'LicenseFiles': {'file': 'x'}},
                         {'LicenseFiles': []}):
                with self.subTest(data=data):
                    source.write_text(json.dumps(data))
                    with self.assertRaises(ValueError):
                        windows_package.qt_license_references({member: source})


if __name__ == '__main__':
    unittest.main()
