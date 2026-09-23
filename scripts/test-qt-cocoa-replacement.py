#!/usr/bin/env python3
"""Native macOS trial using trusted development bundles and their shipped materials.

Rebuild a diagnostic Cocoa variant, replace it in a private app copy and run
synthetic GUI checks. Output is a trial, never a distributable product bundle.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import shutil
import subprocess
import sys

MARKER = 'WEBFENCE_QT_REPLACEMENT_TRIAL'
# Test-only Qt modification; Qt source licensing is preserved in qt-cocoa/README.md.
PATCH = '''
--- a/src/plugins/platforms/cocoa/qcocoaintegration.mm
+++ b/src/plugins/platforms/cocoa/qcocoaintegration.mm
@@ -114,6 +114,7 @@
     , mNativeInterface(new QCocoaNativeInterface)
     , mKeyboardMapper(new QAppleKeyMapper)
 {
     logVersionInformation();
+    qWarning("WEBFENCE_QT_REPLACEMENT_TRIAL");
\x20
     if (mInstance)
'''


def sha(path):
    digest = hashlib.sha256()
    with path.open('rb') as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def tree(root):
    # Preserve symlink identity as well as file contents; do not follow dir links.
    result = {}
    for path in sorted(root.rglob('*')):
        name = str(path.relative_to(root))
        if path.is_symlink():
            result[name] = {'link': os.readlink(path)}
        elif path.is_file():
            result[name] = {'sha256': sha(path)}
    return result


def run(command, output, env=None, timeout=180):
    with output.open('xb') as log:
        subprocess.run([str(x) for x in command], stdout=log,
                       stderr=subprocess.STDOUT, check=True, timeout=timeout, env=env)
    return output.read_text(errors='replace')


def trial(bundle, qt, output):
    # Keep the Qt prefix spelling: framework install names use Homebrew's opt
    # path, not its resolved Cellar directory.
    bundle, qt, output = bundle.resolve(), qt.absolute(), output.absolute()
    if platform.system() != 'Darwin' or platform.machine() != 'arm64':
        raise ValueError('Requires Apple Silicon macOS')
    if output.resolve().is_relative_to(bundle) or output.resolve().is_relative_to(qt.resolve()):
        raise ValueError('Trial output must be outside the bundle and installed Qt')
    exe_rel = Path('Contents/MacOS/webfence')
    plugin_rel = Path('Contents/PlugIns/platforms/libqcocoa.dylib')
    scripts = bundle / 'Contents/Resources/notices/scripts'
    inventory = bundle / 'Contents/Resources/notices/native-build.json'
    installed_plugin = qt / 'share/qt/plugins/platforms/libqcocoa.dylib'
    for required in (bundle / exe_rel, bundle / plugin_rel, inventory, installed_plugin,
                     scripts / 'build-qt-cocoa.sh', scripts / 'qt-cocoa/accessibility.patch'):
        if not required.is_file():
            raise ValueError('Missing trial input: ' + str(required))
    output.mkdir(parents=True, exist_ok=False)
    baseline = tree(bundle)
    installed_hash = sha(installed_plugin)
    preference = Path.home() / 'Library/Application Support/WebFence/ui-language'
    preference_hash = sha(preference) if preference.exists() else None
    record = {'schema_version': 1, 'passed': False, 'distribution_ready': False,
              'scope': 'Modified Cocoa plugin only; not a complete Qt rebuild or legal review',
              'system': platform.mac_ver()[0], 'architecture': platform.machine(),
              'base_inventory_sha256': sha(inventory),
              'base_executable_sha256': sha(bundle / exe_rel),
              'base_plugin_sha256': sha(bundle / plugin_rel), 'marker': MARKER}
    try:
        run(['codesign', '--verify', '--deep', '--strict', bundle], output / 'base-signature.log')
        copy = output / 'WebFence replacement trial.app'
        run(['ditto', bundle, copy], output / 'copy.log')
        # Copy only build materials shipped in the app, never repo counterparts.
        local_scripts = output / 'scripts'
        local_scripts.mkdir()
        shutil.copy2(scripts / 'build-qt-cocoa.sh', local_scripts)
        shutil.copytree(scripts / 'qt-cocoa', local_scripts / 'qt-cocoa')
        (output / 'trial.patch').write_text(PATCH)
        with (local_scripts / 'qt-cocoa/accessibility.patch').open('a') as patch:
            patch.write(PATCH)
        env = os.environ.copy()
        for key in list(env):
            if key.startswith(('QT_', 'DYLD_')):
                del env[key]
        env['QT_QPA_PLATFORM'] = 'cocoa'
        # A source archive may be supplied by the caller; the shipped builder
        # always verifies its pinned SHA-256 before extraction and compilation.
        run(['sh', local_scripts / 'build-qt-cocoa.sh', qt, copy / plugin_rel],
            output / 'rebuild.log', env=env, timeout=600)
        record['trial_patch_sha256'] = sha(output / 'trial.patch')
        record['replacement_plugin_sha256_before_bundle_signing'] = sha(copy / plugin_rel)
        if record['replacement_plugin_sha256_before_bundle_signing'] == record['base_plugin_sha256']:
            raise ValueError('Diagnostic plugin unexpectedly identical to base plugin')
        # codesign may rewrite executable signature bytes; Go build identity
        # must be retained. No Go build/link command is invoked by this trial.
        build_id = subprocess.check_output(['go', 'tool', 'buildid', str(bundle / exe_rel)], text=True).strip()
        if not build_id:
            raise ValueError('Missing Go build ID')
        # Base inventory intentionally remains a base record; this sidecar marks
        # the modified copy explicitly and is included by its new ad hoc signature.
        (copy / 'Contents/Resources/replacement-trial.json').write_text(json.dumps(record, indent=2) + '\n')
        run(['codesign', '--force', '--deep', '--sign', '-', copy], output / 'sign.log')
        run(['codesign', '--verify', '--deep', '--strict', copy], output / 'signature.log')
        copy_id = subprocess.check_output(['go', 'tool', 'buildid', str(copy / exe_rel)], text=True).strip()
        if build_id != copy_id:
            raise ValueError('Go build identity changed')
        record['go_build_id'] = build_id
        record['go_build_id_preserved'] = True
        record['replacement_plugin_sha256_signed'] = sha(copy / plugin_rel)
        for name, argument in [('self-test', '--self-test'), ('soak', '--soak-test=10s')]:
            log = run([copy / exe_rel, argument], output / (name + '.log'), env=env)
            if log.splitlines().count(MARKER) != 1:
                raise ValueError('Expected exactly one loaded-plugin marker in ' + name)
            record[name] = {'exit_code': 0, 'loaded_plugin_marker_count': 1}
        record['passed'] = True
    finally:
        record['original_bundle_preserved'] = baseline == tree(bundle)
        record['installed_plugin_preserved'] = installed_hash == sha(installed_plugin)
        record['personal_language_preserved'] = preference_hash == (sha(preference) if preference.exists() else None)
        preserved = all(record[key] for key in ('original_bundle_preserved',
                                               'installed_plugin_preserved', 'personal_language_preserved'))
        record['passed'] = record['passed'] and preserved
        (output / 'result.json').write_text(json.dumps(record, indent=2) + '\n')
        if not preserved:
            raise ValueError('Original bundle, installed plugin or personal language changed')
    print('PASS Cocoa replacement: shipped materials, diagnostic marker loaded twice, '
          'Go build ID and original inputs preserved')
    print('Trial evidence: ' + str(output / 'result.json'))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('bundle', type=Path)
    parser.add_argument('qt_prefix', type=Path)
    parser.add_argument('new_output_directory', type=Path)
    args = parser.parse_args()
    try:
        trial(args.bundle, args.qt_prefix, args.new_output_directory)
    except (ValueError, OSError, subprocess.SubprocessError) as error:
        print('Cocoa replacement trial failed: ' + str(error), file=sys.stderr)
        sys.exit(1)
