import hashlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest
from unittest.mock import patch
import zipfile

spec = importlib.util.spec_from_file_location('macos_sources', Path(__file__).parents[1] / 'package-macos-sources.py')
package = importlib.util.module_from_spec(spec)
spec.loader.exec_module(package)


class MacOSSourcePackageTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.app = self.root / 'Synthetic.app'
        self.notices = self.app / 'Contents/Resources/notices'
        self.exe = self.app / 'Contents/MacOS/webfence'
        self.exe.parent.mkdir(parents=True)
        self.exe.write_bytes(b'synthetic executable; never run')
        for name in ('LICENSE', 'README.md', 'README.en.md', 'DOCS/ROADMAP.md',
                     'scripts/build-qt-cocoa.sh', 'scripts/qt-cocoa/accessibility.patch',
                     'homebrew/sample@1.0/.brew/sample.rb'):
            p = self.notices / name
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_bytes(b'synthetic packaged material; never execute')
        archive = self.root / 'source.tar.gz'
        with tarfile.open(archive, 'w:gz') as t:
            data = b'synthetic license'
            info = tarfile.TarInfo('sample/LICENSE')
            info.size = len(data)
            t.addfile(info, io.BytesIO(data))
        self.inventory = self.notices / 'native-build.json'
        self.inventory.write_text(json.dumps({'schema': 1, 'distribution_ready': False,
            'webfence_build': {'vcs.revision': 'synthetic'},
            'files': [{'path': 'Contents/MacOS/webfence'}],
            'packages': {'sample@1.0': {'formula': 'sample', 'version': '1.0',
                'metadata_files': ['.brew/sample.rb'], 'upstream_archives': [{
                    'downloadLocation': 'https://example.invalid/source.tar.gz',
                    'checksums': [{'algorithm': 'SHA256', 'checksumValue': package.sha(archive)}]}]}}}))
        self.materials = self.root / 'collected'
        self.collected = package.attachment.sources.collect(self.inventory, self.materials, [archive])
        self.supplement = self.notices / 'homebrew-supplements/sample@1.0/patch.diff'
        self.supplement.parent.mkdir(parents=True)
        self.supplement.write_bytes(b'synthetic patch')
        plan = {'schema': 1, 'packages': {'sample@1.0': {
            'installed_recipe_sha256s': [package.sha(self.notices / 'homebrew/sample@1.0/.brew/sample.rb')],
            'files': [{'path': 'patch.diff', 'url': 'https://example.invalid/patch.diff',
                       'size_bytes': self.supplement.stat().st_size, 'sha256': package.sha(self.supplement)}]}}}
        (self.notices / 'homebrew-supplements/plan.json').write_text(json.dumps(plan))
        self.output = self.root / 'native-sources.zip'
        # Synthetic fixtures do not claim a real codesign trial. Actual signed
        # bundle verification is performed separately on macOS.
        self.signature = patch.object(package.subprocess, 'run')
        self.signature_mock = self.signature.start()
        self.addCleanup(self.signature.stop)
        no_network = patch.object(package.supplements.download, 'fetch', side_effect=AssertionError('unexpected network'))
        no_network.start()
        self.addCleanup(no_network.stop)

    def assemble(self):
        return package.assemble(self.app, self.materials, self.output)

    def test_complete_bytes_and_bundle_binding_without_input_changes(self):
        before = {str(p): package.sha(p) for p in self.app.rglob('*') if p.is_file()}
        result = self.assemble()
        with zipfile.ZipFile(self.output) as z:
            saved = json.loads(z.read('WebFence-native-sources/source-package.json'))
            self.assertEqual(saved, result)
            for f in result['files']:
                data = z.read('WebFence-native-sources/' + f['path'])
                self.assertEqual(len(data), f['size_bytes'])
                self.assertEqual(hashlib.sha256(data).hexdigest(), f['sha256'])
            a = self.collected['archives'][0]
            self.assertEqual(z.read('WebFence-native-sources/notices/upstream-source/' + a['archive']),
                             (self.materials / a['archive']).read_bytes())
            self.assertIn('Included here', json.loads(z.read('WebFence-native-sources/notices/upstream-source/attachment.json'))['archives_location'])
        self.assertEqual(result['signed_bundle_files'][0]['sha256'], package.sha(self.exe))
        self.assertFalse(result['distribution_ready'])
        self.assertFalse(result['corresponding_sources_complete'])
        self.assertEqual(result['archive_count'], 1)
        self.assertEqual(result['supplement_file_count'], 1)
        self.assertEqual(before, {str(p): package.sha(p) for p in self.app.rglob('*') if p.is_file()})
        self.assertEqual(self.signature_mock.call_count, 2)
        self.assertFalse(list(self.root.glob('.webfence-source-package-*')))

    def test_existing_output_and_paths_inside_inputs_rejected(self):
        self.output.write_bytes(b'existing source package')
        with self.assertRaises(FileExistsError):
            self.assemble()
        self.assertEqual(self.output.read_bytes(), b'existing source package')
        for root in (self.app, self.materials):
            with self.assertRaisesRegex(ValueError, 'outside'):
                package.assemble(self.app, self.materials, root / 'new.zip')
            self.assertFalse((root / 'new.zip').exists())
        self.signature_mock.assert_not_called()

    def test_incomplete_collection_and_changed_supplement_leave_no_output(self):
        marker = self.materials / 'INCOMPLETE'
        marker.touch()
        with self.assertRaisesRegex(ValueError, 'incomplete'):
            self.assemble()
        marker.unlink()
        self.supplement.write_bytes(b'corrupted patch')
        with self.assertRaises(ValueError):
            self.assemble()
        self.assertFalse(self.output.exists())
        self.assertFalse(list(self.root.glob('.webfence-source-package-*')))

    def test_signature_failure_and_changed_bundle_refuse_publication(self):
        self.signature_mock.side_effect = subprocess.CalledProcessError(1, ['codesign'])
        with self.assertRaises(subprocess.CalledProcessError):
            self.assemble()
        self.signature_mock.side_effect = None
        original_copy = package.copy_notices
        def change_bundle(source, target):
            original_copy(source, target)
            self.exe.write_bytes(b'concurrent synthetic modification')
        with patch.object(package, 'copy_notices', side_effect=change_bundle):
            with self.assertRaisesRegex(ValueError, 'Bundle inputs changed'):
                self.assemble()
        self.assertFalse(self.output.exists())

    def test_notice_budget_and_links_rejected(self):
        with patch.object(package, 'MAX_NOTICE_BYTES', 1), self.assertRaisesRegex(ValueError, 'budget'):
            self.assemble()
        link = self.notices / 'linked-secret'
        try:
            link.symlink_to(self.exe)
        except OSError:
            self.skipTest('Host cannot create fixture symlink')
        with self.assertRaisesRegex(ValueError, 'regular files'):
            self.assemble()
        self.assertFalse(self.output.exists())
        self.assertFalse(list(self.root.glob('.webfence-source-package-*')))


if __name__ == '__main__':
    unittest.main()
