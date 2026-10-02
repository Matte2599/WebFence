import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest


@unittest.skipUnless(os.name != 'nt' and shutil.which('bash') and shutil.which('sha256sum'),
                     'POSIX stub tools required; Windows restoration is exercised in CI')
class WindowsBinaryRestoreTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.cache = self.root / 'cache with spaces'
        self.cache.mkdir()
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        self.repo = Path(__file__).resolve().parents[2]
        self.entries = []
        for suffix in ('first', 'second'):
            name = 'mingw-w64-ucrt-x86_64-synthetic-' + suffix
            self.entries.append({'name': name, 'version': '1.2-3',
                                 'sha256': hashlib.sha256(b'synthetic archive').hexdigest(),
                                 'source_page': 'https://packages.msys2.org/packages/' + name})
        self.lock = self.root / 'lock.json'
        self.lock.write_text(json.dumps({'schema_version': 1, 'reviewed_on': '2026-10-02',
                                         'packages': self.entries}))
        self.state = self.root / 'state.json'
        self.state.write_text(json.dumps({entry['name']: '1.3-1' for entry in self.entries}))
        self.calls = self.root / 'calls.jsonl'
        self.env = dict(os.environ, PATH=str(self.bin) + os.pathsep + os.environ['PATH'],
                        RESTORE_STATE=str(self.state), RESTORE_CALLS=str(self.calls))
        self.tool('pacman', """
import json, os, pathlib, sys
state_path=pathlib.Path(os.environ['RESTORE_STATE'])
state=json.loads(state_path.read_text())
if sys.argv[1]=='-Q':
    name=sys.argv[2]
    print(name, state[name])
else:
    with open(os.environ['RESTORE_CALLS'], 'a') as output:
        output.write(json.dumps(sys.argv[1:])+'\\n')
    if not os.environ.get('RESTORE_IGNORE_INSTALL'):
        for archive in sys.argv[3:]:
            name=pathlib.Path(archive).name.removesuffix('-1.2-3-any.pkg.tar.zst')
            state[name]='1.2-3'
        state_path.write_text(json.dumps(state))
""")
        self.tool('curl', """
import os, pathlib, sys
with open(os.environ['RESTORE_CALLS'], 'a') as output:
    output.write('"download"\\n')
pathlib.Path(sys.argv[sys.argv.index('--output')+1]).write_bytes(
    b'tampered' if os.environ.get('RESTORE_BAD_HASH') else b'synthetic archive')
""")

    def tool(self, name, body):
        path = self.bin / name
        path.write_text('#!' + sys.executable + '\n' + body)
        path.chmod(0o755)

    def run_restore(self):
        return subprocess.run(['bash', 'scripts/restore-windows-binaries.sh', str(self.cache), str(self.lock)],
                              cwd=self.repo, env=self.env, capture_output=True, text=True, timeout=15)

    def test_batch_restores_drift_and_reuses_verified_archives(self):
        result = self.run_restore()
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = [json.loads(line) for line in self.calls.read_text().splitlines()]
        self.assertEqual(calls[:2], ['download', 'download'])
        self.assertEqual(calls[2][:2], ['-U', '--noconfirm'])
        self.assertEqual(len(calls[2]), 4)
        self.calls.write_text('')
        self.assertEqual(self.run_restore().returncode, 0)
        self.assertEqual(self.calls.read_text(), '')

    def test_bad_hash_never_installs_and_preserves_cached_archive(self):
        archive = self.cache / (self.entries[0]['name'] + '-1.2-3-any.pkg.tar.zst')
        archive.write_bytes(b'previous invalid cache')
        self.env['RESTORE_BAD_HASH'] = '1'
        result = self.run_restore()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(archive.read_bytes(), b'previous invalid cache')
        self.assertEqual(self.calls.read_text().splitlines(), ['"download"'])
        self.assertEqual(list(self.cache.glob('.webfence-reviewed.*')), [])

    def test_failed_version_restore_is_rejected(self):
        self.env['RESTORE_IGNORE_INSTALL'] = '1'
        self.assertNotEqual(self.run_restore().returncode, 0)


if __name__ == '__main__':
    unittest.main()
