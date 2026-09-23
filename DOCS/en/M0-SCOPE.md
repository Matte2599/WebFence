# M0 — First origin boundary check

[Italiano](../it/M0-SCOPE.md) · [Index](../README.md) · [Engine specification](SCANNING.md)

Date: 2026-09-23. Implemented `internal/scope`, independent of Qt with no additional dependencies. **This is the first URL comparison layer, not a complete safe transport or proof of the target owner's authorization.** The desktop remains offline; the HTTP laboratory exists only in tests.

## Current contract

`scope.New(origins)` creates an immutable policy for concurrent reads. At least one explicit HTTP(S) origin is required; a partially invalid configuration cannot produce a usable policy. The zero value denies everything. `Check(rawURL)` returns a fresh `url.URL` when its origin is allowed, otherwise a stable error: `scope_invalid_url`, `scope_invalid_configuration` or `scope_origin_not_allowed`. These package errors contain no input, query or credentials; caller wrappers must preserve redaction.

An origin includes scheme, host and port. Scheme and DNS hostname are compared in lowercase; default ports become explicit 80/443; IPv6 is normalized with `net/netip`. HTTPS does not authorize HTTP, nor does one port authorize others. No implicit wildcard, subdomain or DNS alias. Configuration accepts only an origin and optional trailing slash: paths or queries are rejected, rather than stripped and silently broadening permission.

URLs must be absolute and no larger than 16 KiB. Userinfo, fragments (including an empty `#`), backslashes, unencoded whitespace/control characters, malformed UTF-8, Unicode hosts, trailing DNS dots, IPv6 zones, IPv4-mapped IPv6 and alternative IPv4 forms are rejected. IPv4 requires four canonical decimal octets; IPv6 requires brackets. Explicit ports must be decimal 1–65535 without leading zeros. DNS names use ASCII alphanumeric labels with internal hyphens; already encoded IDNA A-labels can be declared explicitly, without automatic Unicode conversion.

Escaped paths and original queries remain intact, including order, repeated parameters and `..` segments: origin comparison is separate from resource normalization. The caller must resolve a relative link against the appropriate base and check the resulting absolute destination again, including every redirect. The policy performs no DNS resolution and opens no sockets.

## Laboratory and verification

`internal/scope/lab_test.go` creates two `httptest` servers on literal loopback IPs and temporary ports. Only the first origin is in the policy. A **test-only** HTTP adapter checks the initial request and each redirect; an independent dial fence permits connections exclusively to the two lab addresses. Proxies disabled, 3-second client timeout, a three-response redirect-chain limit and fixture reads limited to 1 KiB; no public DNS, secrets or external targets.

The suite exercises allowed requests, relative redirects, direct access and redirects to an excluded port, a redirect to `.invalid` and a bounded redirect loop. The second server must receive zero requests and the dial fence must receive no external attempts: rejection must happen earlier. Fixtures do not assess website vulnerabilities.

Pure tests cover distinct origins, ambiguous input, configuration that cannot silently broaden scope, path/query preservation, independence from caller-owned mutable objects, zero value and concurrent reads. The fuzz test checks that accepted input stays within the declared origin and remains stable on roundtrip, without networking.

Local verification: race-enabled suite passed with 98.6% package statement coverage; a bounded 20-second fuzz run passed (696,840 reported executions). These are observations from this run, not security guarantees or scanner benchmarks. CI runs the package tests on all four runners and a bounded fuzz session on Linux x86-64.

**CI completed:** [run 35859069506](https://github.com/Matte2599/WebFence/actions/runs/35859069506), code `ed71bec`: all four jobs passed (macOS ARM64, Windows Server x86-64, Ubuntu x86-64/ARM64), including race-enabled scope tests; bounded Linux x86-64 fuzzing, build/vet/Qt self-test and macOS bundle passed. This does not establish Windows 10/11 or Debian assistive desktop behavior.

## Limitations and next step

This check may recognize an explicitly listed private/loopback origin, but **does not grant permission to connect to that network**. Before enabling product traffic, IP/CIDR policy, DNS and connection-pinned destination addresses, TLS verification, proxy isolation, authorization/expiry, methods and paths/exclusions, shared budgets, cancellation and response handling are required. The lab does not establish DNS rebinding protection, complete SSRF prevention, browser/WebSocket containment or actual private-network behavior. Its adapter must not be reused as the production broker.

This completes part of the M0 “synthetic lab and initial scope/network tests” gate. GUI, packaging, storage, signing and keychain gates remain open; neither M0 completion nor a scanning release is claimed.

References: [net/url](https://pkg.go.dev/net/url), [net/netip](https://pkg.go.dev/net/netip), [HTTP client and redirects](https://pkg.go.dev/net/http#Client), [httptest](https://pkg.go.dev/net/http/httptest). Compatibility with the future browser's parsers requires dedicated tests.
