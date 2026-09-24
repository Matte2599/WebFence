# M1 — Project snapshot in the HTTP lab

[Italiano](../it/M1-AUTHORIZED-LAB.md) · [Project and authorization](M1-PROJECT-AUTHORIZATION.md) · [M0 transport](ADR-003-TRANSPORT.md) · [Roadmap](../ROADMAP.md)

`transport.NewAuthorizedLab(ctx, permit, grants, limits, resolver)` connects a `project.RunScope` snapshot to the **loopback-only** HTTP broker. This advances the first M1 item; it is not a broker for real networks or websites. The operator's declaration does not prove target ownership.

The constructor rejects absent or expired snapshots. Network grants may cover more origins than the snapshot: they are a separate egress capability and do not widen project authorization. `Fetch` checks the project policy for each initial URL and each post-redirect destination before reserving budget, resolving DNS or opening a socket. The broker still applies its IP grants, actual-peer, TLS, timeout, redirect and budget checks. A rejected destination does not consume another attempt; the request that produced its redirect does.

Snapshot expiry is also a `Fetch` context deadline: it interrupts waiting, DNS, dialing or an in-progress read. The exposed error is a redacted authorization-expired code. `NewLab` remains available for M0 trials without a snapshot and still accepts **only explicitly granted loopback addresses**; the GUI calls neither constructor.

Synthetic tests with two `httptest` servers show that an allowed request succeeds while a redirect to a second origin present in network grants but absent from the project never contacts that server. An independent dial fence allows only the two owned test endpoints. Another test holds a response until expiry and verifies cancellation, the error and accounting for the one started attempt. A zero snapshot is rejected. The `project`/`transport` race suite passed locally; CI includes both on native targets.

**Limits:** project checks still cover exact origins and expiry only. Method, path, exclusion, CIDR/real-network and rate policies, plus discovery/UI integration, remain missing. The existing broker deliberately remains loopback-only; this block neither authorizes external scanning nor establishes general SSRF protection. Next steps are to complete policy limits, persist projects and design a production transport with per-request checks.
