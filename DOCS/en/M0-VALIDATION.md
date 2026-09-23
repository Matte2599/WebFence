# M0 — Verification and closure matrix

[Italiano](../it/M0-VALIDATION.md) · [Roadmap](../ROADMAP.md) · [Index](../README.md)

Updated: 2026-09-23. **M0 remains open.** This register tracks exit criteria, not percentages derived from task counts. “Partial” means positive evidence exists but at least one required check is missing. An offscreen test does not replace assistive or clean-machine trials.

| ID | Criterion | State and evidence | Required for closure |
| --- | --- | --- | --- |
| M0-01 | Native desktop, large table, evidence, IT/EN | Partial: main Qt app, 10,000 fixtures, filters, persistent language, long text, menus/focus and four-target self-tests. [Qt status](QT-DESKTOP.md). | Assistive, visual DPI and extended stability trials below. |
| M0-02 | Builds and packaging on requested targets | Partial: native macOS ARM64, Windows Server x86-64, Ubuntu x86-64/ARM64 builds; macOS development bundle. | Windows/Linux packages, dependencies/notices, launch outside toolchain, minimum systems and Windows 10/11 and Debian trials; explicit residual risks. |
| M0-03 | GUI, SQLite, JWS, keychain | Complete: Qt, SQLite/JWS and [native credentials](ADR-005-CREDENTIALS.md) selected; [four-target CI passed](https://github.com/Matte2599/WebFence/actions/runs/35870058803), including Windows anonymous-token denial. | Retain regressions; GUI integration and key lifecycle remain future work. |
| M0-04 | Versions, skeleton, catalogs, CI | Complete for code `f1d2a84`: [passing four-target CI](https://github.com/Matte2599/WebFence/actions/runs/35863431196). | Repeat on the final M0 commit and verify remote state. |
| M0-05 | Synthetic lab and initial scope/network tests | Complete: [ADR-003](ADR-003-TRANSPORT.md), four-target CI, loopback only. | Keep regressions passing; production transport belongs to M1. |
| M0-06 | Private channel, legal review and contributors | Partial: Private vulnerability reporting enabled and verified in GitHub UI; [SECURITY](../../SECURITY.md) updated. | [Qualified legal review](LEGAL-REVIEW.md) and contributor agreement before opening corresponding processes; do not claim approval without evidence. |

## Reproducible desktop trial

Record commit, package hash, system/version/architecture, Qt, reader/version, scale and keyboard-navigation preference. Use integrated fixtures only; no external targets. Record outcome and defects for each step in both languages. Save any screenshots/audio with synthetic data only.

1. Launch the package from a test account without Go or a Qt toolchain in PATH. Window and commands must work without unexpected network requests or administrator privileges.
2. Enable **VoiceOver** on macOS, **NVDA** on Windows 10 and 11, **Orca** on a Debian/derivative desktop. Check control names/roles/states, column names, selected row and evidence reading. An AX tree alone is insufficient: record actual announcements. Do not permanently alter personal preferences to make a test pass.
3. Using only the keyboard, load fixtures (Ctrl/Cmd+O), search `DEMO-10000` (Ctrl/Cmd+F), move to results (F6), read evidence (Ctrl/Cmd+Shift+E), select/copy text. The tag/script string remains inert text. Try Tab and Shift+Tab, menus and the copy button without focus traps.
4. Switch language (Ctrl/Cmd+1/2) preserving the row and selected text. Apply a filter with no matches: no previous evidence remains presented as selected. Show/hide details (Ctrl/Cmd+Shift+D) without losing focus. Clear, reload, close and reopen: language restored, fixtures not persisted.
5. Try 100%, 150%, 200% scaling, a reduced window and long text: readable labels, reachable commands, working scrolling/splitter, no clipped essential content. Move between monitors with different scaling when available. Local offscreen self-tests with `QT_SCALE_FACTOR=1.5` and `2` passed; these do not verify visual rendering or monitor switching.
6. Controlled session of at least 30 minutes cycling load/filter/language/clear and evidence reading. Record duration, starting/ending memory and trend, freezes, crashes or unbounded growth. Do not claim hardware requirements or benchmarks before measurement.

Known macOS observations: intermittent AX table node and native Tab skipping copy under the current preference; shortcuts and Qt tests do not close those issues. The author has been asked about Windows/NVDA and Debian/Orca machine availability; no result is presumed.

## Final closure

Complete open items with repeatable evidence first. Then run Go/race suites, bounded fuzzing, vet, module/advisory checks, builds, self-tests and applicable package trials. Check translations, links, licenses/notices and `git diff --check`. Update this register, ADRs, README, memory and roadmap; commit/push and CI on final code. Report residual limitations precisely: documenting them does not automatically satisfy a gate.

Packaging investigation: the [inspected local bundle](M0-PACKAGING.md) contains 19 Mach-O files requiring macOS 26 and 9 requiring 14. It does not prove support for older versions; the intended minimum has been requested from the author. Initial inventory and upstream SBOMs identified; compliant packaging and clean-machine trial remain open.

GUI trial on 2026-09-23: standard and 150% layouts inspected, but the 150% copy terminates with SIGSEGV during filter reset followed by language change through accessibility. Reproduced with a selected fixture; native trace being collected, cause still to isolate. Stability/DPI gate not closed.

Update: [Cocoa correction adopted](ADR-006-QT-COCOA.md) in the development bundle. The crash sequence passes at 150% and 200%; at 200% the table remains too compressed, some AX cells remain inconsistently exposed, and VoiceOver/prolonged stability trials remain open.

The 200% layout was subsequently corrected and tested: two complete rows, last column reachable by keyboard, scrollable evidence. Compact-window self-test added and passed locally at scales 1/1.5/2. Reader, multi-monitor and measured stability trials remain open.
