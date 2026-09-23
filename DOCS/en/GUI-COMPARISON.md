# M0 — Practical Fyne and Qt Widgets comparison

[Italiano](../it/GUI-COMPARISON.md) · [Index](../README.md) · [ADR-002 proposal](ADR-002-GUI.md)

**Historical report before the Qt choice** (code through `2000c9a`). Commands and state below describe that trial. Current state: [main Qt desktop](QT-DESKTOP.md), [accepted ADR](ADR-002-GUI.md).
Date: 2026-09-23. **Experimental comparison; final toolkit not selected.** The main program remains Go/Fyne. The [Qt/MIQT laboratory](../../experiments/qt/README.md) is a separate module and sends no network requests.

## Scenario and environment

Both use the same 10,000 records from `internal/demo` and the project's IT/EN catalogs. The Qt laboratory adds a simple summary view and a checkbox revealing advanced evidence. This is not the final design. Qt language selection is session-only; the Fyne prototype persists the preference.

Host: macOS 26.6.2 ARM64, Xcode 27, Go 1.27.1. Fyne 2.8.1; candidate MIQT 0.14.0 with Homebrew Qt 6.11.2 (`qtbase`, `pkgconf`). Qt requires explicit C++17 through `CGO_CXXFLAGS`. No websites were scanned.

## Verified results

| Check | Fyne | Qt Widgets / MIQT |
| --- | --- | --- |
| 10,000 rows, filters, last-row selection | Tests and GUI checked in the first task | Self-test and native GUI load/filter passed |
| IT/EN switching and selection identity | Verified | Self-test passed; full native selector workflow not validated |
| Empty state and no stale evidence | Tests passed | Self-test passed |
| Evidence as inert text | Verified | Test passed; editor returned 624,640 bytes intact |
| Keyboard and selectable text | Assistive gate not passed | Tab/arrow row navigation and Cmd+A text selection observed; typing did not modify evidence |
| macOS accessibility tree | Window/menu in normal build; incomplete optional bridge | Buttons, search, selectors, table, filtered-result cells and evidence text exposed |
| Actions through assistive tooling | Problematic activation | Load, filter and advanced details work; cell/language-menu click insufficient in the trial |
| Local bundle | Launch verified | `macdeployqt`, launch and local ad hoc signature verification passed; no Developer ID signing/notarization |

Qt inspection does not prove full accessibility. During further language-menu attempts, tooling returned `noWindowsAvailable`; the next capture showed an empty instance. Cause not isolated: do not automatically attribute a crash to Qt or mark that workflow passed. Controlled reproduction and actual VoiceOver testing are needed. NVDA and Orca were not tested. With all 10,000 rows loaded, tooling showed the table node without enumerating cells; cells were observed after filtering to one row. In the final-bundle repeat, navigation and evidence worked but the tree did not consistently include the table node: this also remains a gate to isolate between toolkit and inspection tooling.

## Local tests and measurements

- Main application: `go test -tags ci -race ./...` passed; excludes the nested Qt module.
- Laboratory: `go mod verify`, C++17-enabled `go vet ./...` and `QT_QPA_PLATFORM=offscreen ... --self-test` passed. The self-test uses real widgets/model on the owning OS thread; it is not a screen-reader simulation.
- Model value ownership: MIQT returns `QVariant` pointers that Qt copies. The experiment retains reusable values and frees them after model destruction. Two passes of 10,000 reads do not grow the cache on the second pass. This is a finite-fixture cache, not a retention design for future customer data; it is not a soak test or an RSS measurement.
- Latest local offscreen run: load including event processing **3.746 ms**, last-row filter **4.377 ms**. One prototype observation, not a percentile, presentation latency or Fyne performance comparison.
- First successful Qt build after C++17 configuration: **97.48 s**. Includes binding compilation; not a repeatable comparison with equivalent caches.
- Disk usage observed with `du -sh`: Fyne bundle **31 MiB**, Qt **88 MiB**. Different local artifacts, rounded and without common optimization; not RAM, compressed downloads or final release sizes.
- The offscreen backend emitted font and `propagateSizeHints` warnings; the self-test exited with code 0.

## Matrix and limitations

macOS ARM64 is the only Qt desktop tested directly. The dedicated workflow passed compilation, `go vet`, module verification and offscreen self-tests on **Ubuntu 24.04 x86-64 and ARM64**, Qt **6.4.2+dfsg-21.1build5**: [run 35849176371](https://github.com/Matte2599/WebFence/actions/runs/35849176371), code `90c7c7b`. Both logs confirm all checks, including 624,640-byte text and repeated model reads. All five Fyne prototype jobs also passed for the same code: [run 35849176327](https://github.com/Matte2599/WebFence/actions/runs/35849176327). The main Fyne workflow does not check Qt. No Qt runtime testing on Windows 10/11 or Debian desktop has occurred; upstream compatibility claims and CI builds do not replace those checks.

Multiple DPI settings, actual screen readers, prolonged stability, Windows/Linux packaging, the macOS bundle on a clean machine, license inventory and minimum OS versions remain open. Do not distribute the bundle as a ready product.

## Assessment

**Recommendation: continue validating Qt Widgets while retaining Go**, because the prototype already provides an assistive foundation and controls suited to the requested UX. Adoption remains conditional on the [ADR-002](ADR-002-GUI.md) gates. Costs include more complex compilation, distribution and binding ownership; Qt's maturity must not automatically be attributed to MIQT. Fyne remains simpler for Go code, but its current assistive limitation would require additional work without an already verified solution.

Licensing and the future Windows 10 constraint are in the ADR. The observations above come from local tests, not toolkit promises.
