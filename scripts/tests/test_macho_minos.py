from pathlib import Path
import plistlib
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).parents[1]))
from macho_minos import declare_minimum, parse_otool, version
sys.path.pop(0)

MODERN = '''synthetic-binary:
Load command 0
      cmd LC_SEGMENT_64
  cmdsize 72
Load command 1
      cmd LC_BUILD_VERSION
  cmdsize 32
 platform 1
    minos 14.2
      sdk 27.0
   ntools 1
     tool 3
  version 27037.1
'''
LEGACY = '''synthetic-binary:
Load command 0
      cmd LC_VERSION_MIN_MACOSX
  cmdsize 16
  version 10.10
      sdk 15.0
'''


class MachOMinimumTest(unittest.TestCase):
    def test_modern_minimum_ignores_sdk_and_tool_versions(self):
        self.assertEqual(parse_otool(MODERN), {
            'command': 'LC_BUILD_VERSION', 'platform': 'macos', 'minimum': '14.2.0'})
        self.assertEqual(parse_otool(MODERN.replace('platform 1', 'platform MACOS'))['minimum'], '14.2.0')

    def test_legacy_and_numeric_version_order(self):
        self.assertEqual(parse_otool(LEGACY)['minimum'], '10.10.0')
        self.assertGreater(version('10.10'), version('10.9.9'))
        self.assertEqual(version('26'), (26, 0, 0))
        self.assertEqual(version('14.2.1'), (14, 2, 1))

    def test_missing_ambiguous_wrong_platform_and_malformed_commands_rejected(self):
        cases = ['', MODERN + LEGACY, MODERN.replace('platform 1', 'platform 6'),
                 MODERN.replace('platform 1', 'platform 1\n platform 2'),
                 MODERN.replace('minos 14.2', 'minos 14.2\n minos 15.0'),
                 MODERN.replace('minos 14.2', 'minos 0'),
                 MODERN.replace('minos 14.2', 'minos 14.256'),
                 MODERN.replace('minos 14.2', 'minos 14.2beta'),
                 LEGACY.replace('LC_VERSION_MIN_MACOSX', 'LC_VERSION_MIN_IPHONEOS')]
        for text in cases:
            with self.subTest(text=text), self.assertRaises(ValueError):
                parse_otool(text)

    def test_highest_library_requirement_and_higher_explicit_policy_preserved(self):
        files = [{'path': 'Contents/MacOS/webfence', 'minimum_macos': {'minimum': '14.0.0'}},
                 {'path': 'Contents/Frameworks/library.dylib', 'minimum_macos': {'minimum': '26.0.0'}}]
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'Info.plist'
            for previous, expected in [(None, '26.0.0'), ('15.0', '26.0.0'), ('27.1', '27.1.0')]:
                data = {'CFBundleIdentifier': 'synthetic.example'}
                if previous is not None:
                    data['LSMinimumSystemVersion'] = previous
                path.write_bytes(plistlib.dumps(data))
                result = declare_minimum(path, files)
                observed = plistlib.loads(path.read_bytes())
                self.assertEqual(observed['CFBundleIdentifier'], 'synthetic.example')
                self.assertEqual(observed['LSMinimumSystemVersion'], expected)
                self.assertEqual(result['mach_o_requirement'], '26.0.0')
                self.assertEqual(result['limiting_files'], ['Contents/Frameworks/library.dylib'])

    def test_invalid_or_conflicting_policy_preserves_plist(self):
        files = [{'path': 'synthetic', 'minimum_macos': {'minimum': '14.0.0'}}]
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'Info.plist'
            for data, records in [({'LSMinimumSystemVersion': 'unknown'}, files),
                                  ({'LSMinimumSystemVersionByArchitecture': {'arm64': '10.0'}}, files),
                                  ({'CFBundleIdentifier': 'synthetic.example'}, [])]:
                original = plistlib.dumps(data)
                path.write_bytes(original)
                with self.assertRaises(ValueError):
                    declare_minimum(path, records)
                self.assertEqual(path.read_bytes(), original)


if __name__ == '__main__':
    unittest.main()
