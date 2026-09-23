# ADR-003 — M0 transport confined to the laboratory

[Italiano](../it/ADR-003-TRANSPORT.md) · [Index](../README.md) · [URL scope](M0-SCOPE.md)

Date: 2026-09-23. Status: **adopted for the M0 lab**, not production transport approval. Implementation: `internal/transport`. No added dependencies; no GUI integration.

## Problem and decision

Origin comparison alone cannot control a connection: changing DNS, mixed DNS answers, redirects and concurrent requests may bypass checks applied only to the initial URL. M0 must exercise these boundaries before enabling a scanner.

`NewLab(ctx, grants, limits, resolver)` creates a broker restricted to explicitly listed loopback IPs. Each grant binds one HTTP(S) origin to exact IPs; the port comes from the origin. Configuration is copied, duplicate origins are rejected and one origin’s addresses do not authorize another. No wildcard/CIDR, mapped IPv4 or IPv6 zones. Non-loopback private, public, link-local and metadata addresses are also rejected. This restriction is enforced in code, not just documentation.

This enables actual HTTP/TLS tests without opening the product to external targets. Results provide a foundation for M1; the broker must not be described as a scanner or complete SSRF protection.

## Connection and DNS

The resolver is an explicit trusted dependency: it must honor context and return caller-owned addresses. Tests use a synthetic resolver, without public DNS. Literal IPs bypass resolution. For a hostname, every answer must belong to that origin’s grant; empty answers, failures and mixed sets are rejected before opening a socket.

Each HTTP hop resolves again, deterministically selects one allowed IP and connects directly to `IP:port`, also verifying the returned peer. The dialer performs no second resolution. HTTP Host and TLS name remain those of the original URL; certificates and hostnames are verified, with minimum TLS 1.2. Fixture CAs are injected only by package tests; the API offers no TLS-verification bypass.

Only HTTP/1 is used, with a new connection per hop: no pooling, automatic retries or alternate-IP attempts. This simplifies the relationship between attempts, DNS validation and budgets. The connection cost is not a final optimization decision: pooling would require new checks of the same boundaries.

Environment proxies are disabled. `Fetch(ctx, url)` sends GET only; it accepts no headers or credentials, stores no cookies and sends no Referer. Redirects 301/302/303/307/308 are handled explicitly; every new destination goes through origin and IP checks again. GET may still have effects: this interface is for owned fixtures, not permission to scan arbitrary sites.

## Budgets and stopping

Mandatory limits cover shared total attempts, concurrency of entire Fetch chains, redirects, body size, Fetch duration and run duration. There are no implicit unlimited values. Prototype caps are 16 concurrent Fetch calls, 10 redirects and 8 MiB per body; response headers are limited to 32 KiB.

An atomic reservation precedes DNS and connection setup. DNS/network failures consume an attempt; each allowed redirect needs another. Out-of-origin URLs and already cancelled calls consume no new attempts. Fetch duration includes slot waiting, DNS, connection setup, TLS, redirects and reading; run duration starts at construction and does not renew for each call.

`Close()` is idempotent, cancels the run and active work, and prevents new calls. Caller context can also interrupt waiting, DNS, dial or reading. The actual dialer receives the Fetch context: the Go transport’s internal dial context alone may be detached from the request. The test checks that a blocked dial terminates.

The broker requests `Accept-Encoding: identity` and rejects compressed responses: no implicit decompression. It reads at most body limit + 1 byte; exceeding the limit never returns partial content as success. Intermediate redirect bodies are closed without capture. This limits application reads, not exact bytes transferred on the wire. No total disk quota, persistence, rate limiting or durable queue is introduced.

Errors are stable codes, scope errors or `context.Canceled`/`DeadlineExceeded`. Original DNS/TLS/socket errors are redacted. Returned headers and bodies remain untrusted data requiring future redaction and retention policies.

## Completed checks

Local race-enabled tests and `go vet` passed. The lab covers:

- One controlled DNS lookup per hop, pinned destination IP and preserved Host; DNS changes between requests and after redirects rejected.
- Empty, failed or mixed DNS answers; private/metadata/mapped/excluded IPs never contacted; separate grants and copied configuration.
- Concurrent calls within the shared attempt budget, counted redirect hops, bounded loops and an excluded server never contacted.
- Ignored environment proxies, no cookies/credentials/Referer; body/header limits and rejected compression.
- Local HTTPS with trusted fixture CA; untrusted certificates and wrong names rejected without disabling verification.
- Cancellation during DNS, dial, reading and waiting; pre-cancelled/expired runs issue no requests; Fetch timeout and permanent closure.
- IPv6 loopback HTTP where available and rejection of a connected peer differing from the approved endpoint. The IPv6 test reports a skip if the OS lacks IPv6 loopback.

A dial fence independent of the code under test restricts tests to owned endpoints. Public and metadata addresses occur only as synthetic input, never actual destinations.

**CI completed:** [run 35861040736](https://github.com/Matte2599/WebFence/actions/runs/35861040736), code `8f189eb`: all four jobs passed, macOS ARM64, Windows Server x86-64 and Ubuntu x86-64/ARM64. Race-enabled transport tests, scope tests and Qt regressions, build/vet and macOS bundle passed; bounded scope fuzzing ran on Linux x86-64. The IPv6 test was explicitly run and passed locally on macOS without skipping; aggregate CI is not used to claim assistive or IPv6 testing on every requested desktop.

## Consequences and remaining limitations

This task covers the M0 lab and **initial** scope/network test gate. M0 remains open for GUI/accessibility, packaging and other foundation decisions. Real traffic still requires authorization/expiry, internal-service exclusions, IP/CIDR and public/private-network policies, methods and paths, rate limiting, redaction, recovery and scan lifecycle integration. The lab does not establish Internet DNS behavior, enterprise proxies, browser/WebSocket containment, actual private-network behavior or resilience to every OS failure.

Verified references: [Go HTTP transport](https://pkg.go.dev/net/http#Transport), [resolver](https://pkg.go.dev/net#Resolver.LookupNetIP), [contexts](https://pkg.go.dev/context), [TLS verification](https://pkg.go.dev/crypto/tls#Config).
