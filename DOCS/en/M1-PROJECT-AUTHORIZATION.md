# M1 — First block: project and authorization declaration

[Italiano](../it/M1-PROJECT-AUTHORIZATION.md) · [Roadmap](../ROADMAP.md) · [Architecture](ARCHITECTURE.md)

`internal/project` introduces an **in-memory, network-free** project model. This first block prepares a run scope snapshot. The [later SQLite persistence](M1-PROJECT-STORE.md) is separate; scanning and a project-management screen are not yet implemented.

## Implemented contract

`project.New(Draft)` requires a project ID and name, claimed target-owner name, descriptive authorization reference, explicit operator confirmation, future expiry and one to 32 exact HTTP(S) origins. The reference is a label, not the document or a credential; none of these fields is sent over the network. Confirmation records an operator assertion: **WebFence does not independently verify ownership or the legal validity of authorization**.

Partial or ambiguous input is rejected. `internal/scope` validates and canonicalizes origins, which are then copied; equivalent duplicates are rejected. The project and origin list expose no mutable internal structures. `BeginRun()` creates an immutable snapshot only while authorization is current. `RunScope.CheckOrigin(url)` rechecks expiry on every call and enforces exact scheme, host and port. The zero value denies use. Stable error codes omit URLs, owner names and references.

This is **only the origin layer**. It does not yet restrict methods, paths, IP/CIDRs, DNS, budgets or rates, and it opens no socket. A passing check alone is insufficient to make a request. The [next M1 block](M1-AUTHORIZED-LAB.md) connects it to the loopback-only broker; the GUI remains disconnected. No external site was contacted.

## Verification and next blocks

Synthetic tests cover an allowed request, excluded schemes/ports/subdomains, expiry at the exact boundary and after run start, zero values, invalid configurations, canonical duplicates and attempts to widen a snapshot by mutating caller slices. The `internal/project` race suite passed locally; cross-platform CI includes the new package in its race suite.

Remaining M1 work includes store integration and later migrations, authorization renewal/versioning, method/path/exclusion and IP destination policies, a production broker with shared budgets/rate limits, IT/EN UI, discovery and redacted evidence. The lab-only per-hop integration is documented separately. [Human trials carried from M0](M0-VALIDATION.md) remain M1 validation prerequisites.
