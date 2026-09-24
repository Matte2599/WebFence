"""Keep archive-linked Windows attribution sidecars in development ZIPs."""
import importlib.util
from pathlib import Path
import sys
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


if __name__ == '__main__':
    unittest.main()
