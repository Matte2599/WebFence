# ADR-008 — Public IP pinning and route policy

[Italiano](../it/ADR-008-PINNED-PUBLIC-TRANSPORT.md) · [Controlled visits](M1-CONTROLLED-CRAWL.md) · [ADR-003](ADR-003-TRANSPORT.md)

Date: 2026-09-25. Status: **adopted for the experimental M1 core**; distribution and production scanning are not approved.

## Context and decision

The M0 broker tests HTTP boundaries on loopback but cannot exercise integration with authorized public origins. Hostname-only scope does not stop DNS rebinding or redirects into internal networks. We extend the broker with `NewAuthorizedPublic`: every exact origin must be in the authorization snapshot and have explicit public IP pins; every DNS answer must contain only those IPs. A bound lifecycle is mandatory; `RunCrawl` obtains it from the store. Individual runs are sequential, with budgets and per-origin pacing. An independent method/path-prefix policy is rechecked on every hop.

A grant does not prove target ownership. Only the operator can attest authorization and choose routes without unwanted effects. Private/VPN targets, special-address overrides, proxies, cookies, authentication and browsers are unsupported for now. This deliberately restrictive choice means internal targets need a separate authorization and egress design.

## Enforced boundaries

The public constructor limits grants, IPs per grant, attempts, duration and minimum interval. IPv4-mapped IPv6 addresses, zones, non-global-unicast, private, loopback, link-local and special-use ranges are rejected. The special-range table is a conservative snapshot of the [IANA IPv4](https://www.iana.org/assignments/iana-ipv4-special-registry/) and [IANA IPv6](https://www.iana.org/assignments/iana-ipv6-special-registry/) registries as of September 25, 2026; update it with tests when the registries change. It is not proof that an IP is reachable or belongs to the target.

Every hop rechecks authorization, origin, route, budget, DNS and peer; the connection goes directly to the selected IP while the origin hostname remains in Host and TLS verification. One attempt/chain at a time prevents DNS or TLS latency from reordering starts around the rate limit. Network errors exposed to reports are redacted codes. The path policy rejects ambiguous routes rather than guessing how a server interprets them. Results and the HTML surface remain ephemeral.

## Consequences and review

The single-chain limit reduces speed but keeps the initial pace controllable. Rate limiting is broker-local: two processes can add their traffic. Explicit public IP pins need maintenance when DNS changes; this can sharply reduce coverage on CDN domains. This block includes no live Internet test, real Windows/Debian test or system firewall trial. Before calling the transport production-ready, add coordination across runs/processes, target load feedback, an authorization/grant UI, secure persistence, egress control independent of the broker and trials on real platforms. The [M1 roadmap](../ROADMAP.md) keeps these open.
