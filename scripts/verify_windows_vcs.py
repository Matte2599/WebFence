"""Verify a pinned MSYS2 Git source from its archive without a checkout or network."""
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

from msys2_binary_metadata import sha, tar_stream


MAX_LOCK = 16 * 1024
MAX_GIT_ARCHIVE = 512 * 1024 * 1024
GIT_SOURCE = 'mingw-w64::git+https://git.code.sf.net/p/mingw-w64/mingw-w64#commit='


def load_lock(path):
    path = Path(path)
    if not path.is_file() or path.is_symlink() or path.stat().st_size > MAX_LOCK:
        raise ValueError('Invalid Windows VCS source lock')
    data = json.loads(path.read_text(encoding='utf-8'))
    if (not isinstance(data, dict) or set(data) != {'schema', 'source_package', 'version',
            'pkgbuild_sha256', 'archive_sha256', 'vcs_source', 'commit',
            'git_archive_sha256', 'git_archive_size_bytes', 'pack_files'}
            or data['schema'] != 1 or data['source_package'] != 'mingw-w64-winpthreads'
            or not isinstance(data['version'], str)
            or not re.fullmatch(r'[A-Za-z0-9._+~-]+', data['version'])
            or any(not isinstance(data[key], str) or not re.fullmatch(r'[0-9a-f]{64}', data[key])
                   for key in ('pkgbuild_sha256', 'archive_sha256', 'git_archive_sha256'))
            or not isinstance(data['commit'], str)
            or not re.fullmatch(r'[0-9a-f]{40}', data['commit'])
            or data['vcs_source'] != GIT_SOURCE + data['commit']
            or type(data['git_archive_size_bytes']) is not int
            or not 1 <= data['git_archive_size_bytes'] <= MAX_GIT_ARCHIVE
            or not isinstance(data['pack_files'], dict) or len(data['pack_files']) != 3):
        raise ValueError('Invalid Windows VCS source lock schema')
    stems, extensions = set(), set()
    for member, record in data['pack_files'].items():
        match = (re.fullmatch(r'mingw-w64/objects/pack/(pack-[0-9a-f]{40})\.(idx|pack|rev)', member)
                 if isinstance(member, str) else None)
        if (match is None or not isinstance(record, dict)
                or set(record) != {'sha256', 'size_bytes'}
                or not isinstance(record['sha256'], str)
                or not re.fullmatch(r'[0-9a-f]{64}', record['sha256'])
                or type(record['size_bytes']) is not int
                or not 1 <= record['size_bytes'] <= MAX_GIT_ARCHIVE):
            raise ValueError('Invalid Windows VCS pack lock entry')
        stems.add(match.group(1))
        extensions.add(match.group(2))
    if len(stems) != 1 or extensions != {'idx', 'pack', 'rev'}:
        raise ValueError('Expected one complete reviewed Git pack')
    return data


def _git_environment(empty_config, xdg):
    env = {key: value for key, value in os.environ.items()
           if not key.startswith('GIT_') and key not in {'XDG_CONFIG_HOME', 'XDG_CONFIG_DIRS'}}
    env.update(GIT_CONFIG_NOSYSTEM='1', GIT_CONFIG_GLOBAL=str(empty_config),
               GIT_ATTR_NOSYSTEM='1', GIT_NO_REPLACE_OBJECTS='1',
               GIT_OPTIONAL_LOCKS='0', GIT_TERMINAL_PROMPT='0', XDG_CONFIG_HOME=str(xdg))
    return env


def verify_git_archive(archive_path, item, files, srcinfo, checks, lock, zstd='zstd', git='git'):
    """Return a checksum result only after an isolated bare-repository trial."""
    if (item['base'] != lock['source_package'] or item['version'] != lock['version']
            or item['pkgbuild_sha256'] != lock['pkgbuild_sha256']
            or sha(archive_path) != lock['archive_sha256']):
        raise ValueError('Unreviewed Windows VCS source archive or recipe')
    source = lock['vcs_source']
    values, sums = srcinfo.get('source', []), srcinfo.get('sha256sums', [])
    if (len(values) != len(sums) or values.count(source) != 1
            or sums[values.index(source)] != lock['git_archive_sha256']
            or len([entry for entry in checks if entry['source'] == source
                    and entry['status'] == 'vcs_unverified']) != 1):
        raise ValueError('Windows VCS source/checksum declaration differs from lock')
    pack_files = lock['pack_files']
    found = {name for name in files if name.startswith('mingw-w64/objects/pack/')}
    if found != set(pack_files):
        raise ValueError('Windows VCS Git pack set differs from lock')
    for name, expected in pack_files.items():
        actual = files[name]
        if (actual['size_bytes'] != expected['size_bytes']
                or actual['hashes']['sha256'] != expected['sha256']):
            raise ValueError('Windows VCS Git pack member differs from lock')
    if shutil.which(git) is None:
        raise ValueError('Git is required for offline Windows VCS source verification')
    with tempfile.TemporaryDirectory(prefix='webfence-winpthreads-git-') as temporary:
        root = Path(temporary)
        repository = root / 'repo.git'
        empty_config = root / 'empty-config'
        empty_config.write_bytes(b'')
        env = _git_environment(empty_config, root / 'xdg')
        subprocess.run([git, 'init', '--bare', '--quiet', str(repository)], env=env,
                       stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, check=True, timeout=30)
        pack_directory = repository / 'objects' / 'pack'
        copied = set()
        with tar_stream(archive_path, zstd) as archive:
            for member in archive:
                name = member.name.removeprefix('./')
                prefix = item['base'] + '/'
                if not name.startswith(prefix):
                    continue
                relative = name[len(prefix):]
                if relative not in pack_files:
                    continue
                expected = pack_files[relative]
                if not member.isfile() or relative in copied or member.size != expected['size_bytes']:
                    raise ValueError('Duplicate or nonregular Windows VCS Git pack member')
                target = pack_directory / Path(relative).name
                digest = hashlib.sha256()
                with archive.extractfile(member) as source_stream, target.open('wb') as destination:
                    for chunk in iter(lambda: source_stream.read(1024 * 1024), b''):
                        digest.update(chunk)
                        destination.write(chunk)
                if digest.hexdigest() != expected['sha256']:
                    raise ValueError('Windows VCS Git pack bytes differ from lock')
                copied.add(relative)
        if copied != set(pack_files):
            raise ValueError('Missing reviewed Windows VCS Git pack member')
        commit = lock['commit']
        subprocess.run([git, '--git-dir', str(repository), 'fsck', '--strict', '--full', commit],
                       env=env, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE,
                       check=True, timeout=120)
        generated = root / 'source.tar'
        with generated.open('wb') as output:
            subprocess.run([git, '--git-dir', str(repository), '-c', 'core.abbrev=no',
                            'archive', '--format=tar', commit], env=env, stdout=output,
                           stderr=subprocess.PIPE, check=True, timeout=180)
        if (generated.stat().st_size != lock['git_archive_size_bytes']
                or sha(generated) != lock['git_archive_sha256']):
            raise ValueError('Windows VCS Git archive checksum differs from recipe')
    return {'method': 'isolated_git_archive_sha256', 'commit': commit,
            'sha256': lock['git_archive_sha256'],
            'size_bytes': lock['git_archive_size_bytes'],
            'reviewed_pack_count': len(pack_files)}
