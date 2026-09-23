# ADR-006 — Temporary Qt Cocoa correction

[Italiano](../it/ADR-006-QT-COCOA.md) · [Index](../README.md)

Date: 2026-09-23. Status: adopted for the macOS development bundle using Qt 6.11.2; new build CI verification still in progress. Does not close the M0 assistive gate.

## Problem and decision

With the original Qt 6.11.2 Cocoa plugin, reading the accessibility tree after row selection, filter reset and language change reproduced a SIGSEGV. The trace PC falls inside the Cocoa plugin; the stripped binary does not allow direct function attribution. Upstream regression [b1ed5f6](https://github.com/qt/qtbase/commit/b1ed5f656f064e553b33752f8e87d2f5b9553e38) and [proposed correction 765434](https://codereview.qt-project.org/c/qt/qtbase/+/765434), by Caleb Meadows, describe incorrect ownership handling of table accessibility interfaces.

We adopt the two production guards from patch set 1, revision `c7fd3f34b997bb363be15650647665b3b6b8a5f4`: synthesized parent-managed elements must not delete the table's shared accessible interface. The upstream revision is **NEW**, neither approved nor included in a verified release. Accessibility stays enabled. The local before/after comparison supports this diagnosis without proving that other Qt defects are absent.

## Build and maintenance

`scripts/build-qt-cocoa.sh` checks exact version 6.11.2, downloads the official archive and verifies SHA-256 before extraction. It applies only the two guards and builds the plugin against matching installed libraries/private headers. Everything runs in a temporary directory; Homebrew Qt stays unchanged. CMake and Ninja are additional prerequisites. `WEBFENCE_QT_SOURCE_ARCHIVE` allows reuse of a local archive, still checksum-verified.

Packaging replaces the plugin after `macdeployqt`, makes QtCore/QtGui links relative, removes the corresponding Homebrew RPATH and signs the bundle again ad hoc. Errors prevent publication and preserve the previous bundle. A different Qt version requires explicit reassessment: no silent application to a different private ABI. `go run` and unpackaged binaries still use installed Qt and may exhibit the original defect.

[Source, attribution, patch and license texts](../../scripts/qt-cocoa/README.md) are separate from the WebFence license. Distributing a modified Qt library also requires completing corresponding sources, notices and replacement instructions; this task does not make the package release-ready. Reassess and remove the patch when a corrected official Qt version passes the same trial.

## Checks and limitations

- Isolated plugin build and complete local bundle succeeded; `codesign --deep --strict` passed.
- Trial copies at Qt scale 150% and 200%, same Go executable and only the plugin replaced: select `DEMO-10000`, open evidence, switch IT/EN and filter `MO-10000` without the previous crash.
- At 150%, also checked empty results, clearing and reloading. Controls and evidence visible; no network requests.
- At 200%, controls and evidence reachable, but the initial table is too compressed vertically and needs layout work. Do not mark DPI complete.
- Some cells do not appear consistently in AX summaries after selection; actual VoiceOver testing is still needed beyond tree inspection. The new upstream native tests, measured 30-minute cycle and multi-monitor checks were not performed.

This is a verified mitigation of the reproduced crash sequence, not a general accessibility or stability certification.

Additional checks: invalid archive rejected before extraction; plugin dependencies limited to bundled frameworks and system libraries. The bundle rebuilt by the new procedure also passes the GUI sequence at standard scale. Unpackaged binary offscreen self-test passed. The bundle includes only Cocoa: attempting offscreen exits because that plugin is missing; pointing it at Homebrew plugins loads two Qt copies and fails. Do not use this combination to test the bundle: exercise its native GUI.
