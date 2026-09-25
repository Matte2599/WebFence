# M1 — Controlled alpha validation

[Italiano](../it/M1-VALIDATION.md) · [Roadmap](../ROADMAP.md) · [Target platforms](M1-SUPPORT-POLICY.md) · [Internal legal assessment](M1-LEGAL-ASSESSMENT.md) · [Development](DEVELOPMENT.md)

**M1 — controlled traditional alpha: closed by author decision on September 25, 2026.** Its exit criterion is a working flow verified with local code tests, owned fixtures and cross-platform CI; assistive, mixed-monitor and real-workstation trials and outside legal counsel are no longer M1 gates. WebFence provides a bounded, scope-controlled HTTP(S) GET scan with one observational check (`HTTP-XCTO-001`), persistent redacted results and a bilingual Qt window. Closure **does not** claim equivalence to a commercial scanner, a complete audit, certified accessibility or a distribution-ready release.

## Technical criteria

| Criterion | State | Evidence and limit |
| --- | --- | --- |
| Projects, authorization, scope, transport, budgets, cancellation | Implemented and locally tested | `internal/project`, `internal/scope`, `internal/transport`, managed runs and `RunCrawl`; seed preflight, exact pinned IPs, GET/path policy, pacing and revocation. A file lock excludes a second process for the same store; one persistent crawler run at a time is allowed per instance. A declaration does not prove target ownership. |
| HTTP(S) discovery, low-impact check, redacted evidence | Implemented and locally tested | HTML links and forms are observed; links are fetched only with opt-in, forms are never submitted. `HTTP-XCTO-001` distinguishes `nosniff_present`, `nosniff_absent`, inconclusive and not applicable. Synthetic tests cover present/absent headers, redirects, exclusions and limits. The ledger holds no raw target content. |
| Persistence, recovery, limits, basic deletion | Implemented and locally tested | SQLite v4 migrates v1/v2/v3, stores redacted runs and visits, marks an open run `interrupted` at restart, caps the main DB at 64 MiB and runs at 100 per project, offers `Backup` via `VACUUM INTO` to a new private file, and cascades project deletion. The recovery test simulates interruption by leaving a run unfinished; it does not prove behavior after a hardware power loss. WAL and separate backups are not forensically erased. |
| IT/EN desktop: progress, results, incomplete coverage | Implemented and tested with offscreen Qt on macOS | The M1 window creates/selects/deletes projects, requires an authorization confirmation, configures one origin/seed, policy and limits, starts/cancels, and shows reopened visits/results. Progress counts completed visits, **not** a percentage of the site. Incomplete or interrupted coverage remains visible. The native self-test uses a locally owned server. |
| Technical exit: project → lab scan → results → restart | Locally verified | Storage/scanner tests reopen the DB and read visits/outcomes; the Qt self-test covers creation → loopback scan → bilingual result → deletion. No LLM is required. No external target was contacted in tests. |

The database stores project ID, declaration and origins in plaintext, plus redacted visit codes/counts; it stores no visited URLs, queries, headers, bodies or credentials. Basic deletion removes logical project rows, not separate backups or forensic remnants. The SQLite quota limits the main file; WAL and backups can use additional temporary space. Public mode requires operator-pinned public IPs and has not been tested on a real public network. The M1 UI supports one origin and one seed per scan; the core supports multiple within its documented bounds.

## Overall closure test

The [development suite](DEVELOPMENT.md) includes `go mod verify`, race-enabled core tests, `go test ./...`, `go vet ./...`, a Qt build and the offscreen `--self-test`. Report results against the actual tested commit; verify CI on the branch and final `main` commit without creating a status-only commit. A native self-test is not a screen-reader trial or a test on real Windows/Ubuntu hosts.

**Earlier technical evidence:** on September 25, 2026, Apple Silicon/macOS 26.6.2 passed `go mod verify`, the race-enabled core suite, `go test ./...`, `go vet ./...`, a Qt build, offscreen `--self-test` (including the M1 loopback-server workflow), `--soak-test=10s` (4 cycles), `git diff --check` and 79 supporting Python tests (4 environment-specific skips). The lock test also uses a second process; the absent/present-header case is read after restart. [Branch CI](https://github.com/Matte2599/WebFence/actions/runs/36119542128) and [squash `12b3f5e` CI](https://github.com/Matte2599/WebFence/actions/runs/36120434350) both passed (7/7 jobs each). These results do not prove the observational items below.

## Trials and decisions deferred beyond M1

The author explicitly replaced the previous closure requirement: M1-H01…H08 **are not marked as passed** and no longer block the milestone. They remain quality, compatibility, distribution or contribution conditions for relevant future activity; procedures are in the [trial plan](M1-PREREQUISITES.md). [Target minimum versions](M1-SUPPORT-POLICY.md) are set, but have not all been tested on real hardware.

| Gate | State and reason |
| --- | --- |
| M1-H01 VoiceOver | Open: a person must listen and navigate the IT/EN workflow on an Apple Silicon Mac. |
| M1-H02 NVDA | Open: assistive trial is missing on the available real Windows 10 machine; Windows build/version and NVDA have not been recorded. |
| M1-H03 Orca | Not performed: a listener on a real Ubuntu 24.04 desktop is unavailable. |
| M1-H04 Mixed monitors | Open: a two-monitor, mixed-scale trial is missing. |
| M1-H05 Supported workstations | Not performed: Mac and Windows 10 are available for guided trials, but complete real-workstation results are missing; CI and containers test packages, not independent use. |
| M1-H06 Minimum versions | Decision complete: macOS 26 ARM64, Windows 10 1809+ x86-64 and Ubuntu 24.04 LTS x86-64/ARM64. Real trials at the minimum versions have not been performed. |
| M1-H07 Legal review | [Internal M1 assessment](M1-LEGAL-ASSESSMENT.md) completed on the license text; no professional opinion or distribution approval has been obtained. |
| M1-H08 Contributor agreement | Not active: needed before merging substantial contributions intended for commercial relicensing. |

Do not infer assistive or distribution outcomes from CI, emulation, code inspection or offscreen Qt. M1 closure covers the technical alpha; resolve the relevant risks before support, accessibility or distribution claims without presenting them as already passed.
