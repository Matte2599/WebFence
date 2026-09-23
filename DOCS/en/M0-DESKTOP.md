# M0 — First desktop prototype

[Italiano](../it/M0-DESKTOP.md) · [Index](../README.md)

Date: 2026-09-23. First M0 task: Go foundations and offline GUI laboratory. **M0 is not complete.**

## Implemented behavior

- Go/Fyne window with a virtualized table of 10,000 examples, explicitly loaded by the user.
- Combined ID/URL and synthetic-severity filters; selected-record detail, previous/next navigation and evidence copying.
- Evidence displayed as plain text: even the fixture's `<script>` fragment stays inert. Reserved `example.invalid` URLs are never contacted.
- Embedded IT/EN catalogs, initial system language with English fallback and a saved preference. Switching language does not change IDs, canonical filters or evidence.
- Distinct empty and no-match states; clearing records or filtering out the selection also clears displayed evidence, preventing stale details.
- Pinned Go module/dependencies, display-free tests, CI workflow and local macOS bundle script.

There is no scanner, scope/network engine, persistent projects, keychain, CVE, signed reports, browser or AI. Severity values are GUI test data, not security assessments. Copying happens only on user action; clearing the GUI does not clear the clipboard.

## Requested platforms and verification

| Requested platform | This task's verification | Limitation |
| --- | --- | --- |
| Apple Silicon macOS / ARM64 | Local build/bundle on macOS 26.6.2 and ARM64 `macos-15` CI, Go 1.27.1 | Minimum version, distribution signing and notarization unvalidated |
| Windows 10/11 x86-64 | CI build on x64 Windows Server 2022 passed | Does not establish GUI execution on Windows 10/11 |
| Debian/derivatives x86-64 | CI build on x64 Ubuntu 24.04 passed | Debian and minimum versions untested |
| Debian/derivatives ARM64 | CI build on ARM64 Ubuntu 24.04 passed | Linux hardware and desktops untested |

Neither Intel macOS nor 32-bit architectures were requested. This is a support plan, not certification. Remote run outcomes are on the [Actions page](https://github.com/Matte2599/WebFence/actions); a workflow's existence does not establish a passing run.

**Verified CI:** [run 35845953673](https://github.com/Matte2599/WebFence/actions/runs/35845953673), commit `b9556c5`: all five jobs passed (tests and four native builds). The workflow also installs `libwayland-dev`: GLFW requires Wayland headers as well as X11 headers. Markdown-only changes do not rerun builds; this run identifies the verified code.

## Local verification

Go 1.27.1, Fyne 2.8.1, macOS 26.6.2 ARM64 host:

- `go test -tags ci -race ./...`: passed. Filter/identity, catalog/fallback, GUI flow with last-row selection, language, copying, empty state and navigation tests.
- `go vet -tags ci ./...`: passed.
- Native build and `sh scripts/package-macos.sh`: passed; valid plist. The linker warns about duplicate `-lobjc` without preventing compilation.
- Native window opened: verified 10,000-row loading, selection with evidence, IT/EN switching and filtering to `DEMO-10000`. This is not a latency or memory benchmark.
- The software test driver checks in-process interactions, not native accessibility.

Test data is synthetic; no remote target was assessed. Go toolchain/dependency downloads are separate from the application's offline behavior.

## Accessibility: gate not passed

The normal build exposes the window and menus to macOS inspection, but not widgets. Fyne's experimental `accessibility` tag exposes some labels and buttons; in the experiment, selectors, the table and evidence pane content were absent from the tree, with duplicated groups. Activating the button through the tree did not load records and a subsequent tool interaction timed out. The timeout alone does not establish an application crash.

Inspection of Fyne 2.8.1 source confirms that the macOS bridge visits `fyne.Accessible` objects and recurses through `fyne.Container`; enabling the tag is not sufficient to make every composite widget accessible. On Linux this version does not include the same native bridge. The public `fyne.Accessible` interface only provides a label and role; element creation in the inspected macOS bridge passes null action callbacks. This explains the observed activation limitation and requires a toolkit-level solution or comparison with alternatives, not merely changes to WebFence labels. Full VoiceOver, NVDA and Orca testing has not been performed.

The normal bundle remains useful for visual/functional experiments but **does not meet product accessibility requirements**. The script's optional tag only reproduces the investigation. Do not bypass this gate by declaring M0 complete or making unvalidated GUI components permanent.

Next task: determine whether public, supported APIs can resolve the limitation; otherwise compare a desktop toolkit with adequate assistive support on the same scenario (10,000 rows, filters, selectable evidence, IT/EN and four targets). Record the result in the ADR before replacing Fyne. Go remains the core language; a web GUI is not an authorized substitution.

## Remaining M0 work

Accessibility and final GUI choice; DPI and long-text tests; packaging on other platforms and minimum versions; SQLite/JWS/keychain selection and testing; isolated lab and network policy. Legal, CLA and private-reporting tasks remain separate from technical checks. The [UX specification](UX.md) guides subsequent design.

Technical sources: [Fyne 2.8.1](https://github.com/fyne-io/fyne/releases/tag/v2.8.1), [macOS bridge](https://github.com/fyne-io/fyne/blob/v2.8.1/internal/driver/glfw/accessibility_darwin.go), [build without bridge](https://github.com/fyne-io/fyne/blob/v2.8.1/internal/driver/glfw/accessibility_notdarwin.go), [GitHub runners](https://docs.github.com/en/actions/reference/runners/github-hosted-runners). WebFence observations are local checks, not guarantees attributed to these sources.
