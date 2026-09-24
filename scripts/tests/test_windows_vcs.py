"""Exercise offline Git-source verification with a synthetic packed repository."""
import hashlib
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parents[1]))
import collect_windows_sources as sources
import verify_windows_vcs as vcs
sys.path.pop(0)


@unittest.skipUnless(shutil.which('git'), 'Git is required for the offline VCS fixture')
class WindowsVCSTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.work = self.root / 'work'
        empty = self.root / 'empty-config'
        empty.write_bytes(b'')
        self.env = vcs._git_environment(empty, self.root / 'xdg')
        subprocess.run(['git', 'init', '-q', str(self.work)], env=self.env, check=True, capture_output=True)
        (self.work / 'fixture.txt').write_text('Synthetic source; never installed.\n')
        subprocess.run(['git', '-C', str(self.work), 'add', 'fixture.txt'],
                       env=self.env, check=True, capture_output=True)
        subprocess.run(['git', '-C', str(self.work), '-c', 'user.name=Fixture',
                        '-c', 'user.email=fixture@example.invalid', 'commit', '-qm', 'fixture'],
                       env=self.env, check=True, capture_output=True)
        subprocess.run(['git', '-C', str(self.work), '-c', 'pack.writeReverseIndex=true',
                        'repack', '-ad'], env=self.env, check=True, capture_output=True)
        self.commit = subprocess.check_output(['git', '-C', str(self.work), 'rev-parse', 'HEAD'],
                                              env=self.env, text=True).strip()
        self.git_archive = subprocess.check_output(['git', '-C', str(self.work),
                                                    '-c', 'core.abbrev=no', 'archive',
                                                    '--format=tar', self.commit], env=self.env)
        self.source = vcs.GIT_SOURCE + self.commit
        self.recipe = b'# Synthetic recipe; do not execute\n'
        self.item = {'base': 'mingw-w64-winpthreads', 'version': '1.2-3',
                     'pkgbuild_sha256': hashlib.sha256(self.recipe).hexdigest()}
        self.srcinfo = ('pkgbase = mingw-w64-winpthreads\npkgver = 1.2\npkgrel = 3\n'
                        'source = ' + self.source + '\nsha256sums = ' +
                        hashlib.sha256(self.git_archive).hexdigest() + '\n').encode()
        self.pack = {}
        for path in sorted((self.work / '.git/objects/pack').iterdir()):
            member = 'mingw-w64/objects/pack/' + path.name
            raw = path.read_bytes()
            self.pack[member] = {'raw': raw, 'size_bytes': len(raw),
                                 'sha256': hashlib.sha256(raw).hexdigest()}
        self.archive = self.root / 'source.tar.gz'
        self.make_archive()
        self.lock = {'schema': 1, 'source_package': self.item['base'],
                     'version': self.item['version'], 'pkgbuild_sha256': self.item['pkgbuild_sha256'],
                     'archive_sha256': vcs.sha(self.archive), 'vcs_source': self.source,
                     'commit': self.commit,
                     'git_archive_sha256': hashlib.sha256(self.git_archive).hexdigest(),
                     'git_archive_size_bytes': len(self.git_archive),
                     'pack_files': {name: {'size_bytes': record['size_bytes'],
                                           'sha256': record['sha256']}
                                    for name, record in self.pack.items()}}

    def make_archive(self, srcinfo=None, pack=None):
        entries = {'PKGBUILD': self.recipe, '.SRCINFO': self.srcinfo if srcinfo is None else srcinfo,
                   'mingw-w64/hooks/pre-commit': b'Not a hook to run\n'}
        entries.update({name: record['raw'] for name, record in (self.pack if pack is None else pack).items()})
        with tarfile.open(self.archive, 'w:gz') as output:
            for name, raw in entries.items():
                member = tarfile.TarInfo('mingw-w64-winpthreads/' + name)
                member.size = len(raw)
                output.addfile(member, io.BytesIO(raw))

    def test_source_collector_verifies_git_archive_without_checkout(self):
        lock_path = self.root / 'vcs-lock.json'
        lock_path.write_text(json.dumps(self.lock))
        lock = vcs.load_lock(lock_path)
        with patch.object(sources, 'load_vcs_lock', return_value=lock):
            inspected, _ = sources.inspect_source(self.archive, self.item)
        self.assertEqual(inspected['unverified_inputs'], [])
        self.assertEqual(inspected['source_checks'][0]['verification']['sha256'],
                         self.lock['git_archive_sha256'])
        self.assertFalse((self.root / 'hook-ran').exists())
        self.assertFalse((self.root / 'checkout').exists())

    def test_unreviewed_pack_or_recipe_checksum_is_rejected(self):
        with patch.object(sources, 'load_vcs_lock', return_value=dict(self.lock, git_archive_sha256='0' * 64)):
            with self.assertRaisesRegex(ValueError, 'declaration differs'):
                sources.inspect_source(self.archive, self.item)
        corrupted = dict(self.pack)
        member = next(name for name in corrupted if name.endswith('.pack'))
        corrupted[member] = dict(corrupted[member], raw=b'changed packed source')
        self.make_archive(pack=corrupted)
        changed_lock = dict(self.lock, archive_sha256=vcs.sha(self.archive))
        with patch.object(sources, 'load_vcs_lock', return_value=changed_lock):
            with self.assertRaisesRegex(ValueError, 'pack member differs'):
                sources.inspect_source(self.archive, self.item)

    def test_lock_rejects_unsafe_pack_location(self):
        lock_path = self.root / 'vcs-lock.json'
        member = next(iter(self.lock['pack_files']))
        bad = dict(self.lock, pack_files=dict(self.lock['pack_files']))
        bad['pack_files']['../outside.pack'] = bad['pack_files'].pop(member)
        lock_path.write_text(json.dumps(bad))
        with self.assertRaisesRegex(ValueError, 'pack lock entry'):
            vcs.load_lock(lock_path)


if __name__ == '__main__':
    unittest.main()
