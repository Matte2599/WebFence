# Persistent data, deletion and fix validation

[Italiano](../it/DATA-AND-RETEST.md) · [Index](../README.md)

Status: M1–M4 logical model; only the [v2 project-metadata and authorization-revision schema](M1-PROJECT-STORE.md) is implemented. The other entities, complete retention and retesting remain planned. SQLite driver selection and temporary M0 experiments are in [ADR-004](ADR-004-STORAGE-SIGNATURE.md).

## Entities

| Entity | Data and invariants |
| --- | --- |
| Project | ID, local owner, environment, language, retention; no secrets |
| ScopeRevision / Authorization | Allowed/excluded rules, authorization evidence or reference, expiry |
| Asset / Endpoint | Origin, method, route and parameter locations; dynamic data separate |
| CredentialRef | Keychain reference, logical identity and expiry; no tokens in the record |
| ScanRun / CheckExecution | Versions, immutable scope, status, budgets, checkpoints, per-check outcome |
| Finding / Observation | Stable issue identity and immutable per-run observations |
| Evidence | Redacted object, hash, source, capture limitations, date and optional expiry |
| IntelligenceSnapshot | Source and revision to reconstruct historical correlation |
| RetestRun / Comparison | Baseline, context, comparable outcomes, coverage and possible regressions |
| ReportArtifact / AuditEvent | Signed references and redacted actions, without reconstructing deleted secrets |

IDs and machine codes are language-independent. Use UTC timestamps; transactions and foreign keys protect references. Never overwrite historical evidence with new results. Proposed deduplication: rule/family ID, origin, method, route, parameter location and logical role. Keep fingerprint versions; algorithm changes require migration or explicit mapping. Exclude tokens and volatile values without merging semantically distinct paths.

## Retention

“Keep the site in memory” means retaining inventory, configuration, findings and history on disk until explicit deletion, not keeping every page in RAM or cloning the entire site. Before the first scan, show what is stored and the quota.

Metadata and redacted evidence needed for retests remain until deletion, with optional TTL policy. Raw bodies and screenshots are disabled by default; when requested, they have a short configurable expiry, initially proposed as 7 days. Credentials stay in the keychain with explicit reuse and the shortest necessary lifetime. Quotas stop new capture and report incompleteness; they do not silently discard a baseline.

Project deletion stops jobs and removes project credentials, records, files, AI caches, any indexes and temporary data. Shared credential references are detached without removing credentials used by other projects. A minimal deletion log contains no URLs or evidence. Backups and exported copies have an explicit policy: app-managed copies are removed or expire as declared; copies delivered elsewhere cannot be revoked by the app. Recovery must not reintroduce deleted data without notice and an operator decision.

Do not promise guaranteed physical erasure on SSDs, snapshots or external backups. Do not automatically claim “GDPR compliance”; processing responsibilities and terms depend on actual use.

## Retest outcomes

| State | Meaning |
| --- | --- |
| `still_present` | Sufficient evidence reproduces the problem |
| `fixed_verified` | Equivalent check executed, valid prerequisites, required confirmation evidence satisfied |
| `not_observed` | Issue not seen, but verification is insufficient to confirm a fix |
| `inconclusive` | WAF, timeout, expired login, different environment or missing data |
| `not_tested` | Disabled rule, excluded scope or exhausted budget before the check |

`accepted_risk` and `false_positive` are justified triage decisions, not technical retest outcomes. A suppression must not automatically hide new observations.

Before comparing, verify environment, authorization, identity and role, route, rule revision, test data and conditions. Refresh dynamic tokens; do not blindly replay old authenticated requests. A 404, 403 or version change alone does not prove remediation. If a new rule is not equivalent, retain `inconclusive` and require a justified new baseline.

A possible new problem is `newly_observed`; it becomes `regression` only with comparable evidence that the relevant check previously passed. New coverage or rules may expose a pre-existing problem. Show finding delta separately from coverage delta. Successful retesting does not exclude regressions outside the reassessed scope.
