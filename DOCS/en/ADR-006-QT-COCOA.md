# ADR-006 — Temporary Qt Cocoa correction

[Italiano](../it/ADR-006-QT-COCOA.md) · [Index](../README.md)

Date: 2026-09-23. Status: adopted for the macOS development bundle using Qt 6.11.2; new build CI verification completed on all four targets. Does not close the M0 assistive gate.

## Problem and decision

With the original Qt 6.11.2 Cocoa plugin, reading the accessibility tree after row selection, filter reset and language change reproduced a SIGSEGV. The trace PC falls inside the Cocoa plugin; the stripped binary does not allow direct function attribution. Upstream regression [b1ed5f6](https://github.com/qt/qtbase/commit/b1ed5f656f064e553b33752f8e87d2f5b9553e38) and [proposed correction 765434](https://codereview.qt-project.org/c/qt/qtbase/+/765434), by Caleb Meadows, describe incorrect ownership handling of table accessibility interfaces.

The first mitigation adopted the two production guards from patch set 1, revision `c7fd3f34b997bb363be15650647665b3b6b8a5f4`: synthesized parent-managed elements must not delete the table's shared accessible interface. The upstream revision is **NEW**, neither approved nor included in a verified release. Accessibility stays enabled. The local before/after comparison supports this diagnosis without proving that other Qt defects are absent.

## Build and maintenance

`scripts/build-qt-cocoa.sh` checks exact version 6.11.2, downloads the official archive and verifies SHA-256 before extraction. It applies the production changes documented here and builds the plugin against matching installed libraries/private headers. Everything runs in a temporary directory; Homebrew Qt stays unchanged. CMake and Ninja are additional prerequisites. `WEBFENCE_QT_SOURCE_ARCHIVE` allows reuse of a local archive, still checksum-verified.

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

The first correction CI (`9655816`, run 35872886041) passed Windows/Linux and macOS tests, but Cocoa compilation required MoltenVK headers missing from the runner. MoltenVK and Vulkan headers are now explicit prerequisites. The layout was subsequently corrected too: reduced internal margins, table space for at least two complete rows plus scrollbars, non-collapsible panels. Visual trial at 200% with two rows and the last column reached by keyboard; logical 756×430 window self-test passed at scales 1/1.5/2 and desktop vet passed. By-value Qt returns have finalizers removed before explicit destruction; tests caught and corrected a double release while developing this change, before commit. Follow-up CI `953ed2d` passed on all four targets ([run 35873709410](https://github.com/Matte2599/WebFence/actions/runs/35873709410)), including the new self-test and Cocoa bundle.

## Replacement with proposal 772484

The first mitigation’s 30-minute trial completed 600 cycles but produced 17,730 AX warnings; cells remained intermittent. [Yuri Barreira’s proposal 772484](https://codereview.qt-project.org/c/qt/qtbase/+/772484), patch set 1, revision `de050555112940ed643dfa7bd9ed16470adc5a09`, removes deletion of Qt interfaces by native Cocoa elements. Interfaces remain owned by the Qt cache and view. This also explains invalidated references retained by tables; the comparison supports the diagnosis, not a general certification.

The current patch replaces the earlier two guards entirely with production `.h`/`.mm` changes only. Upstream status NEW verified on 2026-09-23; no approval or inclusion in a release presumed. Separate proposal 772485 for cells without a position is not applied, as that case was not reproduced.

Isolated copy with the same executable and new plugin: 20 cycles in 60.003 s, exit zero, 13 RSS samples and empty stderr after a CUA AX read during the workload. In the subsequent OS-input trial all four `DEMO-10000` cells remain available after selection, IT/EN, an empty filter and restoration; the row also returns as the focused element. Language preference restored to Italian. This is a short trial; it does not replace VoiceOver, multiple monitors or a new prolonged session. Upstream tests were not executed.

[Synthetic logs and short-trial metadata](../evidence/macos-ax-772484-2026-09-23/analysis.json).

Patch 772484 and the geometry correction are verified in [CI `8d3053f`, six passing jobs](https://github.com/Matte2599/WebFence/actions/runs/35885936443), including the Cocoa bundle, native self-test and stability smoke. Updated prolonged session completed: 584 cycles in over 30 minutes, empty stderr; RSS trend and cadence limitations in M0-STABILITY.

Rebuilding and replacing the Cocoa plugin with a diagnostic modification: [procedure and evidence](M0-QT-REPLACEMENT.md). Shipped bundle materials reused, same Go build ID, self-test and four Cocoa cycles passed locally; originals preserved. This proves replacement of the plugin only, not the entire distribution.

A [new steady-state AX trial](../evidence/macos-ax-table-reset-2026-09-24.md) isolated a limitation not excluded by the earlier short trial: after filtering 10,000 → 1 → 0 → 10,000 rows, `AXRows` again reports 10,000 but its first row returns `kAXErrorInvalidUIElement`. The internal Qt self-test passes. Two private variants did not fix it; the third is inconclusive because a later UI check found the Mac locked without establishing when. The adopted patch is unchanged; an unlocked-screen retest and real reader trial are needed before closing M0-01.

## Table model while filtering

The repeated CI trial confirms that a full model reset on every filter change can leave Cocoa rows/cells with invalid AX references. The subsequent application change keeps all 10,000 fixtures in the source model and uses `QSortFilterProxyModel` for visible rows. Qt can then emit row insertion/removal during filtering; source resets are restricted to load and clear. Selection identity remains based on the language-independent ID. In the inspected Qt 6.11.2 source, `QAccessibleTable::modelChange(ModelReset)` deletes all child interfaces from its cache, while `RowsRemoved` retains those with valid persistent indexes. This motivates the change but does not establish that it fixes the Cocoa bridge. The self-test checks the first and last accessible cells and that filtering emits row changes without `ModelReset`; native AX diagnosis and a VoiceOver trial remain required.
