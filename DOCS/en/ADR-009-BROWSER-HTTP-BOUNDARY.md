# ADR-009 — Local HTTP boundary for the M3 browser

[Italiano](../it/ADR-009-BROWSER-HTTP-BOUNDARY.md) · [M3 proxy](M3-BROWSER-PROXY.md) · [ADR-008](ADR-008-PINNED-PUBLIC-TRANSPORT.md)

Date: 2026-09-26. Status: **adopted for the M3 core prototype**; a browser runtime and external-target use are not approved.

The later [ADR-012](ADR-012-TEST-SESSIONS.md) extends the core with an optional proxy bound to a test session; the anonymous boundary decided here remains unchanged.
[ADR-019](ADR-019-CDP-HTTPS-BOUNDARY.md) later adds absolute-form HTTPS on the CDP lab's Unix socket only, without enabling `CONNECT`.

## Context and decision

The planned browser produces navigations, resources and JavaScript requests independently of the M1 crawler. Direct browser networking would bypass Go policy and pinned IPs. The first M3 network path is therefore an authenticated local HTTP proxy: after the [gate](M3-BROWSER-GATE.md), the M1 broker performs every GET/HEAD with scope, path, DNS, peer, budget, pacing and redirect checks. The proxy listens on an ephemeral `127.0.0.1` port, requires a random per-instance credential and forwards no client headers or cookies.

This block rejects `CONNECT` and every method with a body. An HTTPS tunnel would prevent the proxy from checking every path and method inside it without another boundary; adding a MITM CA now would introduce certificate management and risks unjustified while there is no browser. The broker follows admitted redirects and rejects out-of-scope ones before another connection.

## Consequences and next decision

The proxy permits HTTP path testing against local fixtures but cannot force a future browser to use it. Its credential protects the listener from unauthorized local callers; it is not a process sandbox. This first block lacks a runtime, independent egress, process/memory limits, HTTPS, sessions, cookies and alternate-path verification; the later HTTPS trial is bounded by ADR-019 above. Choosing between Qt WebEngine and a separate driver, together with the product HTTPS model and OS constraints, requires trials on macOS, Windows and Linux. This ADR closes no M3 roadmap item.
