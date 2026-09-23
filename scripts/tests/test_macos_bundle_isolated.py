from pathlib import Path
import importlib.util
import json
import os
import tempfile
import unittest


SCRIPT = Path(__file__).parents[1] / 'test-macos-bundle-isolated.py'
spec = importlib.util.spec_from_file_location('test_macos_bundle_isolated', SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class IsolatedBundleLogTest(unittest.TestCase):
    def fixture(self, root):
        app = Path(root) / 'WebFence.app'
        executable = app / 'Contents/MacOS/webfence'
        cocoa = app / 'Contents/PlugIns/platforms/libqcocoa.dylib'
        style = app / 'Contents/PlugIns/styles/libqmacstyle.dylib'
        for path in (executable, cocoa, style):
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(b'synthetic')
        plugin_log = ''.join(
            'qt.core.library: "{}" loaded library\n'.format(path)
            for path in (cocoa, style))
        image_log = ''.join(
            'dyld[123]: <12345678-1234-1234-1234-123456789abc> {}\n'.format(path)
            for path in (executable, cocoa,
                         Path('/System/Library/Frameworks/AppKit.framework/AppKit'),
                         Path('/System/Volumes/Preboot/Cryptexes/OS/System/Library/Frameworks/WebKit.framework/WebKit')))
        return app, plugin_log, image_log

    def test_valid_bundle_and_apple_cryptex_images(self):
        with tempfile.TemporaryDirectory() as root:
            app, plugins, images = self.fixture(root)
            self.assertEqual(len(module.plugin_paths(plugins, app)), 2)
            if os.name == 'posix':
                self.assertEqual(len(module.image_paths(images, app)), 4)

    def test_external_plugin_and_missing_cocoa_are_rejected(self):
        with tempfile.TemporaryDirectory() as root:
            app, plugins, _ = self.fixture(root)
            cocoa = app / 'Contents/PlugIns/platforms/libqcocoa.dylib'
            outside = Path(root) / 'libexternal.dylib'
            outside.write_bytes(b'synthetic')
            with self.assertRaisesRegex(ValueError, 'outside the bundle'):
                module.plugin_paths(plugins +
                                    'qt.core.library: "{}" loaded library\n'.format(outside), app)
            with self.assertRaisesRegex(ValueError, 'Cocoa'):
                module.plugin_paths(plugins.replace(
                    'qt.core.library: "{}" loaded library\n'.format(cocoa), ''), app)

    def test_external_image_and_absent_trace_are_rejected(self):
        with tempfile.TemporaryDirectory() as root:
            app, _, images = self.fixture(root)
            external = 'dyld[123]: <12345678-1234-1234-1234-123456789abc> /opt/homebrew/lib/libQt6Core.dylib\n'
            with self.assertRaisesRegex(ValueError, 'outside app and Apple system'):
                module.image_paths(images + external, app)
            with self.assertRaisesRegex(ValueError, 'No app image'):
                module.image_paths('unrelated output', app)

    def test_soak_requires_passed_cocoa_and_cycles(self):
        started = {'event': 'started', 'qt_platform': 'cocoa'}
        passed = {'event': 'passed', 'qt_platform': 'cocoa', 'cycles': 4,
                  'elapsed_seconds': 12.0}
        log = '\n'.join(json.dumps(event, separators=(',', ':'))
                        for event in (started, passed))
        self.assertEqual(module.passed_soak(log)['cycles'], 4)
        for invalid in (log.replace('"cocoa"', '"offscreen"'),
                        log.replace('"cycles":4', '"cycles":0'),
                        log.replace('"passed"', '"failed"'),
                        log + '\n{"event":broken'):
            with self.subTest(log=invalid), self.assertRaises(ValueError):
                module.passed_soak(invalid)


if __name__ == '__main__':
    unittest.main()
