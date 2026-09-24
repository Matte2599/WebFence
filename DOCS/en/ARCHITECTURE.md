# Desktop architecture

[Italiano](../it/ARCHITECTURE.md) · [Index](../README.md)

Status: planned product architecture. For the desktop, the entry point, Qt Widgets/MIQT workspace, IT/EN catalogs and offline fixtures described in [Qt M0](QT-DESKTOP.md) are implemented; the complete scanning workflow remains unimplemented; M0 scope/transport foundations are described below.

## Initial structure

Native desktop application, one operator and local data. A modular Go monolith contains the domain and coordination; the GUI toolkit must not enter analysis packages. The interface calls application services through internal Go APIs. No control HTTP server is exposed by default. A future CLI will reuse the same services.

```mermaid
flowchart TD
  UI[IT / EN desktop] --> APP[Go application services]
  APP --> POL[Authorization and policy]
  POL --> JOB[Budgeted scheduler]
  JOB --> CRAWL[Crawler and checks]
  JOB --> BR[Isolated browser]
  CRAWL --> NET[Network broker and scope]
  BR --> NET
  NET --> TARGET[Authorized targets]
  CRAWL --> EV[Normalized evidence]
  BR --> EV
  FEED[CVE synchronization] --> DB[(Local storage)]
  EV --> DB
  DB --> AI[Optional AI and minimized data]
  AI --> HYP[Structured hypotheses]
  HYP --> POL
  DB --> REP[Reports and signing]
```

The broker represents a boundary to implement and test for browser traffic, redirects, WebSockets and workers too. A diagram does not establish isolation. Browser scanning will not ship until browser containment passes its tests.

## Modules and responsibilities

| Module | Contract |
| --- | --- |
| `project` / `policy` | Project, versioned scope, authorizations, exclusions and immutable per-run budgets |
| `scheduler` | Persistent local queue, bounded workers, cancellation, checkpoints and state |
| `transport` | Exclusive target access; validates destinations, timeouts, sizes and rate limits |
| `discovery` / `checks` | Inventory and versioned checks, with no direct network or secret access |
| `evidence` | Redaction, immutable references, quota and retention |
| `intelligence` | CVE cache, mapping and provenance, separate from target traffic |
| `ai` | Adapters with schemas, budgets and separate egress policies |
| `retest` | Baselines, comparability, observations and regressions |
| `report` | Snapshots, rendering, manifests and signing through a component with limited key access |
| `storage` | Transactions, migrations, deletion and recovery |

The table describes planned complete contracts. `internal/scope` and `internal/transport` implement the M0 foundations; `internal/project` adds the [M1 authorization snapshot](M1-PROJECT-AUTHORIZATION.md), in memory and with no networking of its own. The [M1 authorized broker](M1-AUTHORIZED-LAB.md) connects the components for loopback only; `internal/storage` saves [project metadata](M1-PROJECT-STORE.md) in SQLite without GUI integration. Other engine modules do not yet exist. Avoid dynamic Go plugins in the first version: checks are compiled and reviewed. Future third-party plugins require an isolated process and versioned protocol.

## Persistence and external processes

SQLite is selected (ADR-004) for metadata and the future queue and observation store; separate files will hold larger evidence. The M1 store implements schema v3 for projects only, with transactional migration from v1/v2, authorization revisions, persistent revocation and foreign keys; disk quotas and later migrations remain open. Save a checkpoint and future queue state in one transaction when they describe the same progress. Write files to staging first, then promote them atomically; recovery removes orphans without deleting referenced evidence.

Keep credentials and private keys in the system keychain or an encrypted container unlocked by the operator. The database stores references, not plaintext secrets. SQLite encryption is not automatic: choose a compatible library explicitly and document metadata exposure.

Browsers and LLM runtimes receive temporary directories and minimal credentials. Use pipes or permission-protected local sockets for IPC; if a runtime uses local HTTP, bind to loopback with authentication and origin controls. The browser must not inherit the personal browser profile. Closing the app stops workers; restarting does not automatically repeat operations with uncertain effects.

## States and failures

Run: `draft → queued → running → completed | partial | failed | cancelled`. Restart changes an orphaned `running` run to `interrupted`; resuming creates a tracked attempt after authorization is rechecked. `completed` means planned operations finished, not universal coverage or absence of issues.

All long operations propagate cancellation. The GUI thread stays responsive; events are aggregated to avoid flooding it. Stable localizable errors have a `code`, message and redacted details. Retry only repeatable operations, with limits and backoff; target and provider rate limits are independent.

## Future growth

Server mode, shared users, PostgreSQL and distributed workers require dedicated ADRs and authorization boundaries. Do not deploy distributed infrastructure for one desktop application. A company license does not automatically imply multi-user functionality.

## Implemented scope foundation

`internal/scope` implements only immutable HTTP(S) origin comparison, with no networking or Qt dependencies. A loopback laboratory exists exclusively in tests. It does not replace the planned broker’s IP/DNS, authorization and budget boundaries: [M0 contract](M0-SCOPE.md).

`internal/transport` adds an HTTP/TLS broker confined to loopback grants, with pinned connection IPs and shared budgets/cancellation. M0 decision and production limitations: [ADR-003](ADR-003-TRANSPORT.md). No GUI calls.

`NewAuthorizedLab` applies the project snapshot on every hop of the same broker and uses its expiry to stop in-progress requests. [Managed runs](M1-MANAGED-RUNS.md) add local store-bound revocation; it remains a lab: [M1 contract](M1-AUTHORIZED-LAB.md).

SQLite and the JWS profile are selected in [ADR-004](ADR-004-STORAGE-SIGNATURE.md): SQLite feasibility tests, the [M1 store](M1-PROJECT-STORE.md) and `internal/signature` exist, without projects or reports in the GUI.
