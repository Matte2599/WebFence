# M3 — Third block: Qt WebEngine lab

[Italiano](../it/M3-BROWSER-LAB.md) · [ADR-010](ADR-010-QT-BROWSER-LAB.md) · [Proxy](M3-BROWSER-PROXY.md) · [Roadmap](../ROADMAP.md)

`experiments/m3-browser` is a repeatable trial **using only fixtures created by the command**. Run it with Qt 6 WebEngine/MIQT 0.14.0 and C++17 flags:

```sh
CGO_CXXFLAGS='-O0 -g0 -std=c++17' QT_QPA_PLATFORM=offscreen go run -tags=m3browserlab ./experiments/m3-browser
```

The program accepts no target URLs or arguments. A `.test` origin is connected only to its `httptest` server through the broker with a pinned `127.0.0.1` IP. Qt WebEngine uses an off-the-record profile and the authenticated loopback HTTP proxy. The interceptor blocks disallowed origins/paths/methods, workers and WebSockets; the proxy and broker recheck actual requests. The fixture includes a document, script, `fetch`, out-of-scope image and out-of-scope redirect. The test succeeds only when document/script/API and redirect source reached the local target without authentication headers, the four counted attempts agree, the image was denied and the redirect destination produced a browser error. The JavaScript outcome is a fixture signal, not trustworthy evidence from an arbitrary page.

**Local trial on September 26, 2026, Apple Silicon/macOS 26.6.2, Qt WebEngine 6.11.2:** the command returned `PASS`. The macOS 26 CI job is separate and must be checked on the branch and merged commit. No equivalent Windows or Linux trial is documented yet. The lab is not a GUI feature and lacks process/memory limits, independent egress, HTTPS and sessions; it does not satisfy the M3 browser criterion.
