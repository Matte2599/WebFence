# M0 — Qt as the main desktop

[Italiano](../it/QT-DESKTOP.md) · [Index](../README.md) · [ADR-002](ADR-002-GUI.md)

Date: 2026-09-23. **Author-selected Qt integrated into the main entry point. M0 remains open.**

## Change

`cmd/webfence` launches Qt Widgets through MIQT 0.14.0. The workspace lives in `internal/desktop`; the main module no longer depends on Fyne. The nested laboratory was consolidated, avoiding two GUI copies. Fyne trials and the previous comparison remain historical documentation and Git history (`2000c9a`).

The desktop retains 10,000 fixtures, filters, selection, inert evidence, explicit copying and IT/EN. Simple view shows a summary; “Advanced details” reveals original text. The Language menu and Cmd/Ctrl+1 and Cmd/Ctrl+2 shortcuts switch language without changing IDs or evidence. UI text lives in shared catalogs.

The Go `preferences` package persists language independently of Qt bindings: user file `WebFence/ui-language`, containing only `it` or `en`. Failed saving shows a warning while the session remains usable. Old Fyne preferences are neither imported nor deleted. Commands and paths are in the [guide](DEVELOPMENT.md).

No scanner, target networking, CVE, AI, database or signed reports introduced. The Qt value cache remains bounded by fixtures and two languages; it is not a retention mechanism for real data.

## Checks in this task

- Race-enabled Go tests for fixtures, catalogs and preferences passed: repeated language changes, fallback, corrupt data and I/O errors.
- Main build, C++17-enabled `go vet ./...` and module verification passed on macOS ARM64.
- Qt offscreen self-test passed: 10,000 rows, selection identity, filters, empty state, persistent language actions, long text, intercepted copying and recoverable error warning. Uses a temporary directory rather than user preferences; does not alter the clipboard.
- Main Qt bundle generated, plist and local ad hoc signature verified. GUI opened: 10,000 rows, last-row filter, selection and advanced evidence; Cmd+2 switches to English preserving evidence, closing/reopening restores English, Cmd+1 restores Italian and Cmd+O loads examples. No Developer ID signature/notarization.
- **Final CI passed:** [run 35853356437](https://github.com/Matte2599/WebFence/actions/runs/35853356437), code `fa8bd32`. All four jobs completed: build, race-enabled Go tests, vet and Qt self-test; macOS bundle also passed. macOS 15 ARM64 uses Qt 6.11.2, Windows Server 2022 x86-64 uses MSYS2 UCRT64/Qt 6.11.2-2, Ubuntu 24.04 x86-64 and ARM64 use Qt 6.4.2. These outcomes do not establish assistive GUI use on Windows 10/11 or Debian desktop.

Packaging reuses the build’s C++ flags, avoiding a second binding compilation. macOS CI uses `-O0 -g0`; cold compilation remains lengthy (about 11 minutes in the final run), while subsequent packaging finished in about 40 seconds. These are observations from this run, not product benchmarks. The default local `-O2 -g` build was checked separately.

## Keyboard and accessibility checks — 2026-09-23

The **View** menu provides direct access to search, results and evidence; File precedes View and Language. Search and severity have visible labels associated with their controls. Tab leaves the table (arrows navigate rows) and the evidence text; Shift+Tab moves backward. Tab order follows commands → filters → table → summary → evidence/copy when visible. Hiding details while evidence or its copy button owns focus returns focus to “Advanced details”.

| Action | Windows/Linux | macOS |
| --- | --- | --- |
| Search, selecting the current filter | Ctrl+F | Cmd+F |
| Results; first row if nothing is selected | F6 | F6 (possibly Fn+F6) |
| Open and read evidence | Ctrl+Shift+E | Cmd+Shift+E |
| Show/hide advanced details | Ctrl+Shift+D | Cmd+Shift+D |

Language changes emit data/header notifications without resetting the model. They preserve the current row and evidence text selection. Filters that change rows still use Qt’s model reset.

The self-test checks action focus, menu/checkbox synchronization, stale-data prevention, text selection and `QAccessibleTableInterface` (dimensions and last-cell ID after loading, translation, filtering, restoring and clearing). In offscreen mode it requests Qt accessibility activation for the test only: without an active reader, Qt may not invalidate its cell cache on resets. With Qt 6.4 offscreen, `SetActive` notifies observers but does not activate the bridge: the test logs `qt_accessibility_active=false` and explicitly resets the interface cache before reading it. In that case it checks data and dimensions, not automatic event delivery. Activation was true on local Qt 6.11.2. The normal GUI retains system-managed activation. The test also sends Tab/Shift+Tab events to Qt widgets to check leaving the table, summary, evidence, copy and skipping hidden controls. These checks do not measure the macOS bridge or VoiceOver announcements.

The missing table node after filtering was reproduced through macOS AX inspection before this change. The new keyboard workflow and internal Qt checks are insufficient to declare that limitation resolved. To close the gate, repeat with VoiceOver, NVDA and Orca: load → filter → read headers/row → open evidence → change language → remove filter → clear; verify announcements, focus, original text and absence of stale cells. Record OS/Qt/reader versions, DPI and results per platform.

**macOS GUI trial:** updated bundle opened; Cmd+O/F → DEMO-10000 filter → F6 selects the row; Tab reaches the summary; Cmd+Shift+E opens evidence; Cmd+A followed by Cmd+2 preserves the selected text; Cmd+Shift+D hides the pane and returns focus to the advanced control. View → Read evidence was also activated from the menu. Language restored to Italian. In the observed native configuration, Tab from evidence goes to search, skipping the copy button: the actual cycle also depends on macOS navigation conventions/settings, not just Qt tab order. No system accessibility preferences were changed. The table node remained intermittent in this trial: absent immediately after filtering, exposed again after Tab. No VoiceOver certification.

**Task CI passed:** [run 35857456908](https://github.com/Matte2599/WebFence/actions/runs/35857456908), code `6268ae3`: all four jobs completed with build, vet, Go tests and updated Qt self-test; macOS bundle included. Tab/Shift+Tab and selection preservation pass on all runners. Linux Qt 6.4.2 logs declare the explicit accessible cache reset; this is not an Orca test.

Implementation references: [Qt focus](https://doc.qt.io/qt-6/focus.html), [model notifications](https://doc.qt.io/qt-6/qabstractitemmodel.html#dataChanged), [reset in Qt 6.11.2 source](https://github.com/qt/qtbase/blob/v6.11.2/src/widgets/itemviews/qabstractitemview.cpp). MIQT values returned by `Index()` and `TextCursor()` have automatic finalizers: do not manually free the same value without first removing its finalizer.

## Open gates

The author's choice settles the toolkit decision, not product accessibility. The inconsistently exposed table node and menu actions from the [comparison](GUI-COMPARISON.md) still need isolation; VoiceOver, NVDA, Orca and DPI checks remain. Shortcuts provide an additional path, not assistive certification.

Windows/Linux installers, signing/notarization, Windows 10/11 and Debian desktop execution, a macOS machine without development tools, minimum versions and Windows 10 maintenance after Qt 6.12 remain open. WebFence licensing is unchanged; Qt/MIQT inventory and obligations must be completed before distribution. No purchase or commercial license entered into.

The macOS bundle now rebuilds the Qt 6.11.2 Cocoa plugin with a temporary assistive-crash correction: [ADR-006](ADR-006-QT-COCOA.md). CMake and Ninja are also required (`brew install cmake ninja`); Qt sources are downloaded and hash-verified. Other Qt versions are rejected until reassessed. Unpackaged binaries continue using installed Qt.
