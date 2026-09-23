#!/usr/bin/env python3
"""Exercise a signed Cocoa bundle with a minimal child environment.

Uses only synthetic desktop self-tests. The loader trace observes paths loaded
on these flows; it does not prove arbitrary plugins or a clean-host install.
"""

import json
import re
from pathlib import Path
import subprocess
import sys


APPLE_ROOTS = (Path('/System/Library'), Path('/usr/lib'),
               Path('/System/Volumes/Preboot/Cryptexes/OS'),
               Path('/System/Cryptexes/OS'))
LOADED_PLUGIN = re.compile(r'qt\.core\.library: "([^"]+)" loaded library')
LOADED_IMAGE = re.compile(r'^dyld\[[0-9]+\]: <[0-9A-Fa-f-]+> (/.+)$')


def plugin_paths(log, app):
    app = Path(app).resolve()
    paths = [Path(value).resolve() for value in LOADED_PLUGIN.findall(log)]
    if not paths:
        raise ValueError('No Qt plugin load observed')
    if any(not path.is_file() or not path.is_relative_to(app) for path in paths):
        raise ValueError('Qt loaded a plugin outside the bundle')
    cocoa = (app / 'Contents/PlugIns/platforms/libqcocoa.dylib').resolve()
    if cocoa not in paths:
        raise ValueError('Bundled Cocoa platform plugin was not loaded')
    return paths


def image_paths(log, app):
    app = Path(app).resolve()
    paths = []
    for line in log.splitlines():
        match = LOADED_IMAGE.fullmatch(line)
        if not match:
            continue
        path = Path(match.group(1)).resolve()
        if not path.is_relative_to(app) and not any(path.is_relative_to(root) for root in APPLE_ROOTS):
            raise ValueError('dyld loaded an image outside app and Apple system: ' + str(path))
        paths.append(path)
    if not paths or not any(path.is_relative_to(app) for path in paths):
        raise ValueError('No app image observed in dyld trace')
    cocoa = (app / 'Contents/PlugIns/platforms/libqcocoa.dylib').resolve()
    if cocoa not in paths:
        raise ValueError('Cocoa plugin absent from dyld trace')
    return paths


def passed_soak(log):
    events = []
    for line in log.splitlines():
        if line.startswith('{"event":'):
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError as error:
                raise ValueError('Malformed soak result') from error
    if len(events) != 2 or events[-1].get('event') != 'passed' or events[-1].get('qt_platform') != 'cocoa' or events[-1].get('cycles', 0) < 1:
        raise ValueError('Cocoa soak did not report completed cycles')
    return events[-1]


def run(app):
    app = Path(app).resolve()
    executable = app / 'Contents/MacOS/webfence'
    if not executable.is_file():
        raise ValueError('Missing packaged executable')
    environment = {'PATH': '/usr/bin:/bin', 'QT_QPA_PLATFORM': 'cocoa',
                   'QT_DEBUG_PLUGINS': '1'}
    self_test = subprocess.run([str(executable), '--self-test'], env=environment,
                               text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                               timeout=60, check=False)
    if self_test.returncode != 0:
        raise ValueError('Isolated Cocoa self-test failed, exit ' + str(self_test.returncode))
    plugins = plugin_paths(self_test.stdout, app)
    environment.pop('QT_DEBUG_PLUGINS')
    environment['DYLD_PRINT_LIBRARIES'] = '1'
    soak = subprocess.run([str(executable), '--soak-test=10s'], env=environment,
                          text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                          timeout=45, check=False)
    if soak.returncode != 0:
        raise ValueError('Isolated Cocoa soak failed, exit ' + str(soak.returncode))
    result = passed_soak(soak.stdout)
    images = image_paths(soak.stdout, app)
    return {'plugins_loaded_from_bundle': len(plugins),
            'images_observed': len(images),
            'app_images': sum(path.is_relative_to(app) for path in images),
            'soak_cycles': result['cycles'], 'soak_elapsed_seconds': result['elapsed_seconds']}


if __name__ == '__main__':
    if len(sys.argv) != 2:
        raise SystemExit('usage: test-macos-bundle-isolated.py WebFence.app')
    result = run(sys.argv[1])
    print('PASS isolated Cocoa bundle: {plugins_loaded_from_bundle} bundled plugins, '
          '{app_images}/{images_observed} app images, {soak_cycles} cycles in '
          '{soak_elapsed_seconds:.3f}s'.format(**result))
