# M3 — Second block: confined HTTP proxy

[Italiano](../it/M3-BROWSER-PROXY.md) · [Gate](M3-BROWSER-GATE.md) · [ADR-009](ADR-009-BROWSER-HTTP-BOUNDARY.md) · [Roadmap](../ROADMAP.md)

`internal/browser.NewProxy` opens an ephemeral listener **only on `127.0.0.1`**, with at most 32 simultaneous connections, headers up to 32 KiB and read/write timeouts. It requires a random 256-bit proxy credential generated for each instance and kept in memory only. After authentication, it admits absolute-form HTTP requests using GET/HEAD without a body. Every request traverses the [M3 gate](M3-BROWSER-GATE.md) and then the M1 broker, with scope, policy, pinned DNS/IP, budget, pacing and redirects checked again. Proxy responses and errors do not include URLs in their messages.

The proxy rejects HTTPS `CONNECT`, WebSocket upgrades, request bodies, declared workers and unsupported methods. It forwards no client headers, including credentials and cookies, to the target; the broker creates its own request without authentication headers. The response includes only headers needed for rendering and basic restrictions, excludes `Set-Cookie`, and uses `Cache-Control: no-store` to avoid uncounted reuse. The proxy does not implement an authenticated session or expand scope based on page content.

**This block does not launch or configure a browser.** A browser could ignore the proxy or use alternate channels: an isolated runtime, independent egress, interception of all resource types, process/memory limits and tests on every platform are still needed. HTTPS, authentication and WebSockets are unsupported by this proxy. Do not treat it as proof of M3 containment or use it for external targets.

`httptest` exercises loopback only and checks a valid request, HEAD, headers/CSP and no forwarding of cookies or credentials, proxy authentication, third-party origin, excluded path, worker, WebSocket, POST and out-of-scope redirect. These are neither real-browser tests nor SPA/API coverage measurements.
