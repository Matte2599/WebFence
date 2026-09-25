# M1 — Controlled alpha validation

[Italiano](../it/M1-VALIDATION.md) · [Roadmap](../ROADMAP.md) · [Human prerequisites](M1-PREREQUISITES.md) · [Development](DEVELOPMENT.md)

The **automatable technical scope of M1** is implemented. WebFence provides a bounded, scope-controlled HTTP(S) GET scan with one observational check (`HTTP-XCTO-001`), persistent redacted results and a bilingual Qt window. This is not equivalent to a commercial scanner, a complete audit or a distribution-ready release. **Formal M1 closure remains open** until the human gates below are documented and accepted.

## Technical criteria

| Criterion | State | Evidence and limit |
| --- | --- | --- |
| Projects, authorization, scope, transport, budgets, cancellation | Implemented and locally tested | `internal/project`, `internal/scope`, `internal/transport`, managed runs and `RunCrawl`; seed preflight, exact pinned IPs, GET/path policy, pacing and revocation. A file lock excludes a second process for the same store; one persistent crawler run at a time is allowed per instance. A declaration does not prove target ownership. |
| HTTP(S) discovery, low-impact check, redacted evidence | Implemented and locally tested | HTML links and forms are observed; links are fetched only with opt-in, forms are never submitted. `HTTP-XCTO-001` distinguishes `nosniff_present`, `nosniff_absent`, inconclusive and not applicable. Synthetic tests cover present/absent headers, redirects, exclusions and limits. The ledger holds no raw target content. |
| Persistence, recovery, limits, basic deletion | Implemented and locally tested | SQLite v4 migrates v1/v2/v3, stores redacted runs and visits, marks an open run `interrupted` at restart, caps the main DB at 64 MiB and runs at 100 per project, offers `Backup` via `VACUUM INTO` to a new private file, and cascades project deletion. The recovery test simulates interruption by leaving a run unfinished; it does not prove behavior after a hardware power loss. WAL and separate backups are not forensically erased. |
| IT/EN desktop: progress, results, incomplete coverage | Implemented and tested with offscreen Qt on macOS | The M1 window creates/selects/deletes projects, requires an authorization confirmation, configures one origin/seed, policy and limits, starts/cancels, and shows reopened visits/results. Progress counts completed visits, **not** a percentage of the site. Incomplete or interrupted coverage remains visible. The native self-test uses a locally owned server. |
| Technical exit: project → lab scan → results → restart | Locally verified | Storage/scanner tests reopen the DB and read visits/outcomes; the Qt self-test covers creation → loopback scan → bilingual result → deletion. No LLM is required. No external target was contacted in tests. |

The database stores project ID, declaration and origins in plaintext, plus redacted visit codes/counts; it stores no visited URLs, queries, headers, bodies or credentials. Basic deletion removes logical project rows, not separate backups or forensic remnants. The SQLite quota limits the main file; WAL and backups can use additional temporary space. Public mode requires operator-pinned public IPs and has not been tested on a real public network. The M1 UI supports one origin and one seed per scan; the core supports multiple within its documented bounds.

## Technical closure test

Run the [development suite](DEVELOPMENT.md) from the candidate commit: `go mod verify`, race-enabled core tests, `go test ./...`, `go vet ./...`, a Qt build and the offscreen `--self-test`. Verify CI for the branch and final commit on `main`; report results against the commit actually tested, without making a commit solely to record CI. The native self-test is not a screen-reader trial or a test on real Windows/Debian hosts.

**Local run on September 25, 2026:** on Apple Silicon/macOS 26.6.2, `go mod verify`, the race-enabled core suite, `go test ./...`, `go vet ./...`, a Qt build, the offscreen `--self-test` (including the M1 loopback-server workflow), `--soak-test=10s` (4 cycles), `git diff --check` and 79 supporting Python tests (4 environment-specific skips) passed. The lock test also uses a second process; the absent/present-header case is read after restart. Cross-platform CI must be evaluated for the branch and final `main` commit; this paragraph does not preempt its result.

## Outstanding human gates

Every item below **requires human intervention before declaring M1 closed**; procedures and pass criteria are in the [M1 plan](M1-PREREQUISITES.md).

| Gate | State and reason |
| --- | --- |
| M1-H01 VoiceOver | Open: a person must listen and navigate the IT/EN workflow on an Apple Silicon Mac. |
| M1-H02 NVDA | Open: assistive trial is missing on the available real Windows 10 machine; Windows build/version and NVDA have not been recorded. |
| M1-H03 Orca | Open: a real Debian desktop and a person listening to Orca are unavailable. |
| M1-H04 Mixed monitors | Open: a two-monitor, mixed-scale trial is missing. |
| M1-H05 Supported workstations | Open: Mac and Windows 10 are available for guided tests, but complete trials are not recorded; real Windows 11 and Debian x86-64/ARM64 hosts remain unavailable. CI and containers do not substitute. |
| M1-H06 Minimum versions | Open: macOS 26 was selected but needs a trial on the minimum version; Windows/Debian minimums remain to be chosen and tested. |
| M1-H07 Legal review | Open: counsel has not yet been engaged; license, notices and dossier lack professional approval. |
| M1-H08 Contributor agreement | Open: text and acceptance process require counsel review and author approval. |

Do not mark these gates passed based on CI, emulation, code inspection or offscreen Qt tests. They are required before the relevant support/distribution claims; local development may continue.
