# M1 — Human-intervention prerequisite plan

[Italiano](../it/M1-PREREQUISITES.md) · [M0 matrix](M0-VALIDATION.md) · [Roadmap](../ROADMAP.md)

This plan makes the gates carried from M0 into M1 executable. **M0** status remains exclusively in the [M0 matrix](M0-VALIDATION.md). A CI build, AX/AT-SPI test or container cannot replace a person's judgment on the required desktop. The [local preflight dated 2026-09-24](../evidence/m1-prerequisite-preflight-2026-09-24.md) records only checks actually performed.

## Execution matrix

| ID | Prerequisite | Environment and trial owner | Gate closure criterion |
| --- | --- | --- | --- |
| M1-H01 | VoiceOver | Real Apple Silicon Mac; person listening to and using VoiceOver | IT and EN record with actual announcements, navigation and evidence reading; blocking defects fixed or limitation explicitly accepted. |
| M1-H02 | NVDA | Real Windows 10/11 x86-64 desktop; person listening to and using NVDA | Same IT/EN trial on extracted ZIP, launched outside MSYS2; Windows and NVDA versions recorded. |
| M1-H03 | Orca | Real Debian/derivative desktop, x86-64 or ARM64; person listening to and using Orca | Same IT/EN trial on installed `.deb`; X11/Wayland session and Orca version recorded. |
| M1-H04 | Mixed monitors | Real desktop with two active monitors at different scales; operator | Move window between displays in both directions with focus, table, menus, popups and text usable in IT/EN; record scales and resolutions. |
| M1-H05 | Supported workstations | Real Windows 10 x86-64, Windows 11 x86-64, Debian/derivative x86-64 and ARM64, possibly with different operators | For each combination: obtain and verify package, install/extract, launch outside toolchain, IT/EN synthetic workflow, exit and restart. Record successes and defects separately. |
| M1-H06 | Minimum OS versions | Macs, Windows and Debian machines at the **declared** minimum versions after support decision | Set minimum version per platform/architecture, ensure binaries and dependencies are compatible, execute synthetic workflow on that real OS. A newer version is insufficient. |
| M1-H07 | Legal review | Qualified legal counsel retained by the author | Opinion on precise versions/hashes of LICENSE, notices/packages and [dossier](LEGAL-REVIEW.md); agreed changes applied; documented author approval before relevant distribution. |
| M1-H08 | Contributor agreement | Author and counsel, then acceptance process | Text for individual and company rights holders reviewed/approved, version and acceptance verifiable before merging substantial contributions requiring relicensing. |

**Initial state:** procedures prepared; none of M1-H01…H08 passes merely because this document exists. H01 can be performed on the current Mac only with a listener; H04 needs a second monitor. H02/H03/H05 need real workstations and operators. For H06 the author chose **macOS 26** as the temporary minimum target on Apple Silicon; the current bundle declares 26.0.0, but a real 26.0 trial is still needed. Windows/Debian minima remain to be chosen and tested. H07/H08 need a professional and author approval.

## Common preparation

1. Record source commit, name and SHA-256 of the **actual tested artifact**, architecture, full OS/Qt/reader versions and session type. On macOS record hashes of the bundle executable and Cocoa plugin; on Windows hash the ZIP; on Debian hash the `.deb`. Do not reuse an older build's hash.
2. Use a real workstation independent of the CI runner, a test account without customer data, and a development package. Before claiming a “clean host” install, record any preinstalled Qt, MSYS2, Go or other dependencies: if they affect launch, qualify the result.
3. Launch WebFence from the installed/extracted artifact, not `go run` or a loose executable. Use the included 10,000 synthetic fixtures and, for the new M1 window, only an operator-owned loopback server if available; configure no external target.
4. Keep notes, optional recordings and screenshots private. Publish only a redacted summary with hashes/versions, outcomes and reproducible defects; no tester names, voices, personal data or account preferences.

### Handoff for the available Windows 10 PC

The author can perform a guided trial on Windows 10. First record the **Windows version and build** (`winver` or PowerShell: `(Get-CimInstance Win32_OperatingSystem) | Select-Object Caption,Version,BuildNumber`) and obtain a development ZIP built from the commit under test; current CI tests the ZIP but does not publish it as a downloadable artifact. Prepare it with the [UCRT64 Windows toolchain and procedure](DEVELOPMENT.md#running-and-checking) / [packaging commands](ADR-007-PACKAGING.md#available-commands), or transfer a development ZIP privately with an agreed hash. Do not treat a lone `.exe` as the package. After extraction, record `(Get-FileHash -Algorithm SHA256 .\webfence-windows-amd64.zip).Hash` in PowerShell, run `WebFence\webfence.exe`, complete the IT/EN workflow below and restart. For automated package validation, PowerShell 7 can run `./scripts/test-windows-package.ps1 -Archive ./dist/webfence-windows-amd64.zip` from the matching checkout; that does not replace desktop use. Do not install NVDA or change assistive settings merely to produce a false H02 result: that gate stays open until a person can listen to it.

## IT/EN desktop workflow

Repeat **in each language**. Record what the reader actually announces, not just the expected label; distinguish “not announced”, “announced but ambiguous” and “unreachable”.

1. Open menus and controls with the keyboard; check focus order, visible focus and return to the table without a mouse.
2. Load 10,000 examples. Reach headers, first and last row, four cells, selection and evidence pane; read ID, severity, title and evidence. Confirm the reader does not encounter invalid row/cell references after the action.
3. Search `DEMO-10000` (one row), then `no-such-fixture` (zero), then clear search (10,000). Repeat reading and selection after each transition; record displayed count and announcements. This sequence covers the historical AX defect.
4. Explicitly copy **only** synthetic evidence and verify the result. Open and close advanced details; check long text and absence of stale evidence from a previous selection when results are empty.
5. Change language using menu/shortcut, exit and restart. Check translations, saved preference and stable IDs. Restore the operator's original preference after the trial.
6. For M1-H04 move the window, with results and popups open, from monitor A to B and back; repeat focus, rows and menus at each scale. Record any restart needed, clipping, blur or focus loss.
7. Open **M1 Scan** using the keyboard. Check labels, unchecked authorization confirmation, fields and buttons, coverage warning, redacted result and IT/EN text. A scan trial requires a loopback server the operator controls; do not use external sites for this gate. After restart, verify the saved result and then delete the test project using the confirmation dialog.

An assistive gate passes only when a person has actually **heard** the announcements and completed the workflow with the reader active. Automation with muted audio is diagnostic support. If there is a defect, record reproduction, environment, severity and a retest on the corrected package; do not mark “pass” merely because a trial was attempted.

## Trial record to copy per environment

Keep the operational record out of the repository if it contains personal data; publish a redacted version. Every `pass` line needs direct observation.

```text
Gate ID:                    Status: pending | pass | fail | blocked
Date (UTC):                 Source commit:
Artifact / SHA-256:         Qt / MIQT:
Machine/architecture:      OS and build:
Session/displays/scales:    Reader and version (or none):
Operator: person, private identifier; do not publish name
IT: launch / keyboard / 10000→1→0→10000 / evidence / copy / language / restart:
EN: launch / keyboard / 10000→1→0→10000 / evidence / copy / language / restart:
Observed announcements (H01-H03):
Monitor A→B→A (H04):
Preinstalled dependencies / host limitations:
Defects and retest references:
Blocker or acceptance rationale:
Operator attestation (kept privately):
```

For H05 create four separate records, one per listed combination. H06 needs an additional record for every agreed minimum version, even if it coincides with H05: link the two without duplicating evidence. macOS 26 is the agreed target, still to be tried on version 26.0; Windows 10 and Debian minima remain product decisions. Do not infer a real trial from the upstream Qt matrix or bundle metadata.

## Legal and contribution track

For H07 provide counsel with the [dossier](LEGAL-REVIEW.md), IT/EN LICENSE, CONTRIBUTING and an inventory of artifacts actually intended for distribution. Keep the engagement, jurisdictions/uses considered, opinion and changes private; record publicly only text version/hash, date, author decision and open issues. H08 uses the [specification in the dossier](LEGAL-REVIEW.md#contributor-agreement-specification-to-draft); a template or PR checkbox is not acceptance. Until H07/H08 close, do not claim distribution and relicensing are approved or merge substantial contributions intended for relicensing.

These gates do not prevent continued **local** M1 development and tests with synthetic fixtures; they are prerequisites for the stated claims and activities, and remain visible until completed.
