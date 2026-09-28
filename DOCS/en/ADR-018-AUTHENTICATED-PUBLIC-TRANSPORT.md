# ADR-018 — Test sessions on public HTTPS grants

[Italiano](../it/ADR-018-AUTHENTICATED-PUBLIC-TRANSPORT.md) · [M3 sessions](M3-SESSIONS.md) · [ADR-008](ADR-008-PINNED-PUBLIC-TRANSPORT.md)

Date: 2026-09-27. Status: **adopted for experimental core transport; desktop selection added later**.

## Decision

`NewAuthorizedPublicWithSession` is separate from the anonymous M1 public transport. It requires a managed run, one exact-origin grant with pinned public IPs, a GET policy for verification, exact login and verification URLs, confirmation of the POST and a **second explicit confirmation** of credential transmission to the public origin. The origin must use HTTPS; the broker verifies certificate and hostname with TLS 1.2 or newer. The budget is at most 128 attempts in 15 minutes, sequentially and at least 500 ms apart. Login and authenticated GET redirects are never followed. DNS, peer, authorization, route, revocation and budget are checked for each request.

This avoids implicitly widening `NewAuthorizedPublic`, which serves anonymous visits, and does not implicitly enable the desktop dialog; the later [GUI block](M3-AUTH-DESKTOP.md) requires a separate confirmation and tighter limits. The operator must retain authorization evidence and declare URLs and IPs: the software cannot prove target ownership. The existing session manager can use the broker, retaining ephemeral cookies and secrets and positive identity verification; the core two-account run can now explicitly select this broker, while the desktop dialog selects public mode explicitly.

## Tests and limits

A local TLS fixture simulates a public origin with pinned DNS and peer; it checks POST, cookie-bearing GET, a stopped redirect, out-of-scope URL and wrong peer. It makes no Internet requests. Trials on an authorized staging target, complex login flows, authenticated browsing, aggregate target-load control across processes and independent egress remain open.
