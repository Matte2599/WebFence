# M0 — Verification matrix and closure report

[Italiano](../it/M0-VALIDATION.md) · [Roadmap](../ROADMAP.md) · [Index](../README.md)

Updated: 2026-09-24. This is the sole source for M0 status. Closure covers the automatable scope defined by the author, not a supported release or certification of accessibility, security or legal compliance. Work requiring people or real hardware is a prerequisite for M1; deferral is not a passing result.

**Later note (September 25, 2026):** the transfer to M1 below records the September 24 M0 closure decision. The author later [closed M1 on technical criteria and local/CI tests](M1-VALIDATION.md), deferring human trials and outside counsel to relevant support/distribution work. This note does not alter the historical M0 outcome or mark those trials as passed.

## M0 gates

| ID | Criterion | Automatable outcome and evidence | Limit carried into M1 |
| --- | --- | --- | --- |
| M0-01 | Native desktop, large table, evidence, IT/EN | **Closed for the automatable scope.** Qt prototype with 10,000 fixtures, filters, reading/copying, persistent language, focus and keyboard. [AT-SPI on Debian ARM64 in an isolated environment](../evidence/debian-atspi-2026-09-24.md), [headless Orca](../evidence/debian-orca-headless-2026-09-24.md) and [blocking Cocoa AX test on macOS 15/26](https://github.com/Matte2599/WebFence/actions/runs/35975333332) pass on synthetic flows. The [30-plus-minute Cocoa session](M0-STABILITY.md) used a build preceding the final AX patch; the final patch has self-test, a 30-round AX sequence and a short soak. | Announcements and use with VoiceOver/NVDA/Orca on real desktops; transitions between differently scaled monitors. |
| M0-02 | Development builds and packaging on requested targets | **Closed for the automatable scope.** macOS bundle, Windows x86-64 ZIP and Debian x86-64/ARM64 `.deb` files launched in isolated environments; checks for signatures, hashes, imports, dependencies, provenance and native materials are described in [ADR-007](ADR-007-PACKAGING.md), [source materials](M0-SOURCE-MATERIALS.md) and the [Windows inventory](../evidence/windows-pe-import-inventory-2026-09-24.md). **By the author's decision, the current source and notice inventory is sufficient for M0: no additional verification layers are required.** This does not approve licenses or commercial distribution. | Trials on real Windows 10/11 and Debian systems and declared minimum systems; professional distribution review. |
| M0-03 | GUI, SQLite, JWS, credential choice | **Complete.** Qt Widgets/MIQT, SQLite, Ed25519 JWS and [native credential adapter](ADR-005-CREDENTIALS.md) selected and exercised by relevant tests. | Product integration of these components in M1. |
| M0-04 | Versions, skeleton, catalogs, CI | **Complete.** Go module, IT/EN catalogs, formatting and test checks, CI matrix for macOS, Windows and Linux; separate Debian packaging. | Keep CI green in subsequent blocks. |
| M0-05 | Synthetic lab and initial scope/network checks | **Complete.** [Transport policy](ADR-003-TRANSPORT.md), `.invalid` fixtures, test traffic limited to loopback, budgets and regressions. | Scanning engine and policy for real targets belong to M1. |
| M0-06 | Security channel, legal text, contributors | **Closed for the automatable scope.** Private vulnerability reporting verified on GitHub; [SECURITY](../../SECURITY.md), [legal dossier](LEGAL-REVIEW.md) and contributor-agreement specification prepared. | Qualified legal review and an approved contributor agreement before opening the corresponding processes. |

## Comprehensive closure verification

This block updates work rules, IT/EN matrices, README/roadmap links and CI activation for Markdown changes. On product code unchanged from `51f6219`, local checks on 2026-09-24 passed: 79 Python tests (4 skipped), validation of 93 Markdown files/800 local links, `go test ./...`, race suite for foundation components, `go vet ./...`, `go mod verify`, two 10-second fuzz tests, Go formatting and `git diff --check`. The Apple Silicon bundle rebuilt from the same code passed signing, Cocoa self-test, a 12.043-second/four-cycle soak and a native AX client for three 10,000 → 1 → 0 → 10,000 cycles. A 30-minute soak started on the final build was **stopped at the author's request** and is not counted as passing. Branch and final squash-commit CI must be checked on GitHub; their outcomes do not require a status-only commit.

## Requires human intervention — prerequisite for M1

| Gate | Reason and required trial |
| --- | --- |
| VoiceOver on macOS | The AX client and self-tests cannot establish what the reader actually announces. A person must check navigation, headers, rows, selection and evidence reading. |
| NVDA on Windows 10/11 | Windows tests use automated backends and sessions; listening and navigation with a reader on a real desktop are missing. |
| Orca on Debian or derivatives | The headless trial uses null audio output and logs commands, not audible announcements or usability on a real desktop. |
| Multiple monitors at different scales | The inventoried local hardware has one display; 150/200% scaling in one window does not reproduce moving between screens. |
| Real Windows 10/11 and Debian machines | Runners, containers and isolated environments test packages and synthetic flows, not installation/use on independent workstations. |
| Minimum systems | The macOS minimum is derived from bundle binaries but has not been tested on a machine running it; trials on the minimum supported Windows/Debian versions are also missing. |
| Legal review | [LICENSE](../../LICENSE), notices and dossier are prepared; a professional must assess effectiveness, compatibility and distribution conditions. The review is yet to be arranged. |
| Contributor agreement | A specification exists, not an approved and adopted text; rights must be defined and approved before accepting external contributions requiring relicensing. |

## Procedure for human trials

Use synthetic fixtures only; record commit, package hash, OS version, reader, Qt, scale, monitor setup and IT/EN outcome. Launch the package outside the toolchain and exercise keyboard, menus, filter sequence 10,000 → 1 → 0 → 10,000, evidence selection/copy, language changes and restart. Record actual announcements, defects and limits without permanently changing personal preferences. No external website is a target of these trials.

The [M1 operational plan](M1-PREREQUISITES.md) provides records and criteria for collecting future results.
