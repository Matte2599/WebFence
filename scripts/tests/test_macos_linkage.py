from pathlib import Path
import importlib.util
import json
import os
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).parents[1] / 'check-macos-linkage.py'
spec = importlib.util.spec_from_file_location('check_macos_linkage', SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


def fake_otool(values):
    def inspect(flag, binary):
        own, deps, search = values[binary.name]
        if flag == '-L':
            return str(binary) + ':\n' + ''.join(
                '\t' + value + ' (compatibility version 1.0.0, current version 1.0.0)\n'
                for value in ([own] if own else []) + deps)
        if flag == '-D':
            return str(binary) + ':\n' + (own + '\n' if own else '')
        if flag == '-l':
            return ''.join('Load command ' + str(index) + '\n'
                           '      cmd LC_RPATH\n'
                           '  cmdsize 48\n'
                           '     path ' + value + ' (offset 12)\n'
                           for index, value in enumerate(search))
        raise AssertionError(flag)
    return inspect


class MacOSLinkageTest(unittest.TestCase):
    def fixture(self, root):
        app = Path(root) / 'WebFence.app'
        exe = app / 'Contents/MacOS/webfence'
        lib = app / 'Contents/Frameworks/libsafe.dylib'
        for file in (exe, lib):
            file.parent.mkdir(parents=True, exist_ok=True)
            file.write_bytes(b'\xcf\xfa\xed\xfe' + b'synthetic')
        notice = app / 'Contents/Resources/notices/native-build.json'
        notice.parent.mkdir(parents=True)
        notice.write_text(json.dumps({'files': [
            {'path': 'Contents/MacOS/webfence'},
            {'path': 'Contents/Frameworks/libsafe.dylib'}]}))
        values = {
            'webfence': (None, ['@executable_path/../Frameworks/libsafe.dylib',
                                '/usr/lib/libSystem.B.dylib'], []),
            'libsafe.dylib': ('/opt/homebrew/lib/libsafe.dylib',
                              ['/System/Library/Frameworks/Cocoa.framework/Cocoa'], [])}
        return app, values

    def test_self_install_name_is_not_external_dependency(self):
        with tempfile.TemporaryDirectory() as root:
            app, values = self.fixture(root)
            self.assertEqual(module.check(app, fake_otool(values)), {
                'binary_count': 2, 'load_references': 3,
                'apple_system_references': 2, 'bundle_references': 1})

    def test_external_and_unresolved_dependencies_fail(self):
        with tempfile.TemporaryDirectory() as root:
            app, values = self.fixture(root)
            for dep in ('/opt/homebrew/lib/libmissing.dylib',
                        '@executable_path/../Frameworks/libmissing.dylib',
                        '@rpath/libmissing.dylib',
                        '@loader_path/../../../../outside.dylib'):
                with self.subTest(dep=dep):
                    values['webfence'] = (None, [dep], [])
                    with self.assertRaises(ValueError):
                        module.check(app, fake_otool(values))

    def test_internal_rpath_resolves_and_external_or_ambiguous_search_fails(self):
        with tempfile.TemporaryDirectory() as root:
            app, values = self.fixture(root)
            values['webfence'] = (None, ['@rpath/libsafe.dylib'],
                                  ['@loader_path/../Frameworks'])
            self.assertEqual(module.check(app, fake_otool(values))['bundle_references'], 1)
            for search in (['/opt/homebrew/lib'],
                           ['@loader_path/../Frameworks', '@executable_path/../Frameworks'],
                           ['@loader_path/../../../../outside']):
                with self.subTest(search=search):
                    values['webfence'] = (None, ['@rpath/libsafe.dylib'], search)
                    with self.assertRaises(ValueError):
                        module.check(app, fake_otool(values))

    def test_inventory_symlink_escape_fails(self):
        with tempfile.TemporaryDirectory() as root:
            app, values = self.fixture(root)
            lib = app / 'Contents/Frameworks/libsafe.dylib'
            outside = Path(root) / 'outside.dylib'
            outside.write_bytes(lib.read_bytes())
            lib.unlink()
            try:
                lib.symlink_to(outside)
            except OSError:
                self.skipTest('Host does not permit creation of a symlink')
            with self.assertRaises(ValueError):
                module.check(app, fake_otool(values))

    def test_bad_tool_output_fails(self):
        with tempfile.TemporaryDirectory() as root:
            app, values = self.fixture(root)
            def malformed(flag, binary):
                return str(binary) + ':\n\tno version metadata\n' if flag == '-L' else fake_otool(values)(flag, binary)
            with self.assertRaises(ValueError):
                module.check(app, malformed)

    def test_sanitize_removes_only_external_staging_rpath(self):
        with tempfile.TemporaryDirectory() as root:
            app, values = self.fixture(root)
            values['webfence'] = (None, [], ['/opt/homebrew/Cellar/example/lib',
                                             '@executable_path/../Frameworks'])
            edits = []
            def edit(command, check):
                self.assertTrue(check)
                edits.append(command)
                self.assertEqual(command[0:3], ['install_name_tool', '-delete_rpath',
                                                 '/opt/homebrew/Cellar/example/lib'])
                values['webfence'][2].remove(command[2])
            result = module.sanitize(app, fake_otool(values), edit)
            self.assertEqual(result, {'binary_count': 2, 'removed_external_rpaths': 1})
            self.assertEqual(len(edits), 1)
            self.assertEqual(module.check(app, fake_otool(values))['bundle_references'], 0)

    def test_sanitize_refuses_shared_inode_before_edit(self):
        with tempfile.TemporaryDirectory() as root:
            app, values = self.fixture(root)
            lib = app / 'Contents/Frameworks/libsafe.dylib'
            outside = Path(root) / 'installed.dylib'
            original = lib.read_bytes()
            outside.write_bytes(original)
            lib.unlink()
            try:
                os.link(outside, lib)
            except OSError:
                self.skipTest('Host does not permit creation of a hard link')
            values['libsafe.dylib'] = (None, [], ['/opt/homebrew/lib'])
            edits = []
            with self.assertRaises(ValueError):
                module.sanitize(app, fake_otool(values), lambda *args, **kwargs: edits.append(args))
            self.assertEqual(edits, [])
            self.assertEqual(outside.read_bytes(), original)


if __name__ == '__main__':
    unittest.main()
