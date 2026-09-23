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

## Open gates

The author's choice settles the toolkit decision, not product accessibility. The inconsistently exposed table node and menu actions from the [comparison](GUI-COMPARISON.md) still need isolation; VoiceOver, NVDA, Orca and DPI checks remain. Shortcuts provide an additional path, not assistive certification.

Windows/Linux installers, signing/notarization, Windows 10/11 and Debian desktop execution, a macOS machine without development tools, minimum versions and Windows 10 maintenance after Qt 6.12 remain open. WebFence licensing is unchanged; Qt/MIQT inventory and obligations must be completed before distribution. No purchase or commercial license entered into.
