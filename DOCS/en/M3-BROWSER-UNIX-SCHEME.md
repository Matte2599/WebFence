# M3 — Browser trial with Unix socket and Qt scheme

[Italiano](../it/M3-BROWSER-UNIX-SCHEME.md) · [ADR-013](ADR-013-BROWSER-UNIX-SCHEME.md) · [Qt lab](M3-BROWSER-LAB.md) · [M3 status](M3-VALIDATION.md)

`NewObservedUnixProxy` accepts a private local directory (`0700`), opens `broker.sock` and serves requests through the existing gate and broker. It publishes no TCP endpoint. The test checks an allowed GET, denial of an outside origin, redacted observations and rejection of a nonprivate directory. It is not a sandbox for the calling process.

On macOS with Qt 6.11.2, the optional lab starts a second supervised helper. Its configuration arrives on `stdin` and constrains the origin, secret and socket. A Qt scheme serves a document, script and `fetch` for three synthetic paths; each request crosses the Unix proxy and gate/broker/target counts are checked. The first helper still verifies the HTTP proxy, DOM navigation, subresources and blocked redirects/outside origins. No external target is contacted.

Local verification: `go test -race ./internal/browser -run 'TestUnixProxy|TestProxy' -count=1`, `CGO_CXXFLAGS='-O0 -g0 -std=c++17' go test -tags=m3browserlab ./experiments/m3-browser -count=1` and `CGO_CXXFLAGS='-O0 -g0 -std=c++17' QT_QPA_PLATFORM=offscreen go run -tags=m3browserlab ./experiments/m3-browser` passed on Apple Silicon/macOS 26.6.2 on September 27, 2026. Branch and merge CI will be recorded in the [M3 matrix](M3-VALIDATION.md) after they run.

The Qt scheme path cannot work on the Linux job's Qt 6.4 because `FetchApiAllowed` is absent; its HTTP trial remains. The scheme does not preserve complete HTTP semantics and does not prevent the browser from using its own TCP sockets. Independent isolation, HTTPS, aggregate quotas, Windows and desktop integration remain open.
