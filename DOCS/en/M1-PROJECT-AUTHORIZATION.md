# M1 — First block: project and authorization declaration

[Italiano](../it/M1-PROJECT-AUTHORIZATION.md) · [Roadmap](../ROADMAP.md) · [Architecture](ARCHITECTURE.md)

This document describes the **historical first increment** of the isolated model. Later persistence/UI and policies are in the [M1 matrix](M1-VALIDATION.md); “not yet” below refers to that increment.

`internal/project` introduces an **in-memory, network-free** project model. This first block prepares a run scope snapshot. [SQLite persistence](M1-PROJECT-STORE.md) is a separate component; scanning and a project-management screen are not yet implemented.

## Implemented contract

`project.New(Draft)` requires a project ID and name, claimed target-owner name, descriptive authorization reference, explicit operator confirmation, future expiry and one to 32 exact HTTP(S) origins. The reference is a label, not the document or a credential; none of these fields is sent over the network. Confirmation records an operator assertion: **WebFence does not independently verify ownership or the legal validity of authorization**.

Partial or ambiguous input is rejected. `internal/scope` validates and canonicalizes origins, which are then copied; equivalent duplicates are rejected. The project and origin list expose no mutable internal structures. `BeginRun()` creates an immutable snapshot only while authorization is current. `RunScope.CheckOrigin(url)` rechecks expiry on every call and enforces exact scheme, host and port. The zero value denies use. Stable error codes omit URLs, owner names and references.

A complete new declaration can be validated with `Project.ReviseAuthorization(AuthorizationDraft)`: it preserves the ID and name, increments the revision and leaves the previous project and `RunScope` untouched. `Project.RevokeAuthorization()` creates a non-runnable revision; fresh confirmed consent can reactivate it. `Project.Revision()` and `RunScope.Revision()` identify the version. The [v4 store](M1-PROJECT-STORE.md) saves revisions and revocation, detects concurrent edits and returns only the current one for new runs. Old runs retain their original limits; runs [managed by the store](M1-MANAGED-RUNS.md) are also stopped when the declaration changes.

This is **only the origin layer**. It does not yet restrict methods, paths, IP/CIDRs, DNS, budgets or rates, and it opens no socket. A passing check alone is insufficient to make a request. The [next M1 block](M1-AUTHORIZED-LAB.md) connects it to the loopback-only broker; the GUI remains disconnected. No external site was contacted.

## Verification and next blocks

Synthetic tests cover an allowed request, excluded schemes/ports/subdomains, expiry at the exact boundary and after run start, zero values, invalid configurations, canonical duplicates, revisioning and attempts to widen a snapshot by mutating caller slices or renewing a project. Cross-platform CI includes the package in its race suite.

Later blocks added desktop integration, policy, discovery and redacted observations as recorded in the [M1 matrix](M1-VALIDATION.md). [Human trials carried from M0](M0-VALIDATION.md) have not been performed; by the author's later decision they do not block technical M1 closure and remain in the [trial plan](M1-PREREQUISITES.md).
