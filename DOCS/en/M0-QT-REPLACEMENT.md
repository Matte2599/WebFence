# M0 — Rebuilding and replacing the Qt Cocoa plugin

[Italiano](../it/M0-QT-REPLACEMENT.md) · [Index](../README.md) · [Cocoa decision](ADR-006-QT-COCOA.md)

Technical procedure for the ARM64 macOS development bundle with Qt 6.11.2. It demonstrates that a modified Cocoa plugin can be rebuilt and loaded while retaining the same Go build. It does not demonstrate a complete Qt rebuild, compatibility with other versions/private ABIs or legal compliance of the package. M0-02 and legal review remain open.

## Materials and prerequisites

Use a trusted development bundle and the tools described in the [development guide](DEVELOPMENT.md): Apple Silicon macOS, Apple developer tools, Go, Qt 6.11.2 with matching private headers, CMake, Ninja and MoltenVK/Vulkan headers. Execution requires a Cocoa desktop session. The minimum OS depends on the [bundle binaries](M0-PACKAGING.md).

`Contents/Resources/notices/scripts` includes the [builder](../../scripts/build-qt-cocoa.sh), [trial](../../scripts/test-qt-cocoa-replacement.py) and `qt-cocoa/` with CMake, patch and licenses. The builder downloads official Qt sources or reuses `WEBFENCE_QT_SOURCE_ARCHIVE`, always verifying SHA-256 before extraction. The URL, hash and attribution are in the [patch README](../../scripts/qt-cocoa/README.md). Installed Qt libraries provide the build environment; they are not replaced.

## Repeatable trial

From the repository root, after producing the bundle with `sh scripts/package-macos.sh`:

```sh
app="$PWD/dist/WebFence.app"
trial_dir="$(mktemp -d /tmp/webfence-replacement.XXXXXX)/trial"
python3 "$app/Contents/Resources/notices/scripts/test-qt-cocoa-replacement.py" \
  "$app" "$(brew --prefix qtbase)" "$trial_dir"
cat "$trial_dir/result.json"
```

The final directory must be new and outside the bundle and Qt. Preserve the prefix returned by Homebrew: manually resolving it to a `Cellar` path can change the framework names expected by the builder. To avoid downloading the Qt archive again, set `WEBFENCE_QT_SOURCE_ARCHIVE` to the file already verified by the source collector. The trial does not use the repository to rebuild the plugin: it copies materials included in the app.

The procedure:

1. Verifies the original bundle signature, records file hashes/link targets and copies the app to a separate directory whose name contains spaces.
2. Copies shipped materials, adds the `WEBFENCE_QT_REPLACEMENT_TRIAL` message at Cocoa initialization only in the copied patch and rebuilds the plugin. The accessibility correction remains applied.
3. Replaces `Contents/PlugIns/platforms/libqcocoa.dylib` in the copy, retains links to bundled frameworks and signs it ad hoc again. It does not rebuild or relink Go; it compares the Go build ID because signing can change executable bytes.
4. Runs the self-test and ten-second synthetic soak with the Cocoa backend and a cleaned Qt/DYLD environment. Requires zero exit status and one diagnostic message per execution, proving the modified plugin is actually loaded.
5. Rechecks the entire original bundle, installed Qt plugin and any personal language preference. Self-test and soak already use temporary preferences and intercepted clipboard copying.

`result.json` and logs outside the app describe the final outcome. The copy retains the base bundle inventory and adds `replacement-trial.json` before signing: that sidecar describes the variant **before** testing, with `passed=false`. The copy is a trial artifact, not a distributable package with an updated inventory. Returning to the original bundle provides rollback; the original app is never replaced.

## Evidence and limitations

[Local trial on 2026-09-23](../evidence/macos-cocoa-replacement-2026-09-23/result.json) passed on macOS 26.6.2 ARM64: variant rebuilt from shipped materials, valid signature, same Go build ID, self-test and four cycles in 12.003 s; diagnostic message detected in both executions. Original bundle, installed Qt plugin and personal language unchanged. Invalid archive rejected before extraction; existing output refused while preserving its result. The [context](../evidence/macos-cocoa-replacement-2026-09-23/context.json) distinguishes the previous bundle with local modifications from the new trial script and records the first failed attempt involving the Homebrew prefix. CI on the new code remains to be verified. Technical success does not approve LICENSE/CLA or close distribution review. Corresponding materials and embedded-dependency mapping, procedures for other libraries/platforms, actual readers, clean desktops and minimum systems remain to be completed. This ad hoc signature does not verify Developer ID/notarization or hardened-runtime behavior.
