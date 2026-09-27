# M3 — Third block: Qt WebEngine lab

[Italiano](../it/M3-BROWSER-LAB.md) · [ADR-010](ADR-010-QT-BROWSER-LAB.md) · [ADR-011](ADR-011-BROWSER-HELPER.md) · [Proxy](M3-BROWSER-PROXY.md) · [Roadmap](../ROADMAP.md)

`experiments/m3-browser` is a repeatable trial **using only fixtures created by the command**. Run it with Qt 6 WebEngine/MIQT 0.14.0 and C++17 flags:

```sh
CGO_CXXFLAGS='-O0 -g0 -std=c++17' QT_QPA_PLATFORM=offscreen go run -tags=m3browserlab ./experiments/m3-browser
```

The program accepts no target URLs or arguments. A `.test` origin is connected only to its `httptest` server through the broker with a pinned `127.0.0.1` IP. Qt WebEngine uses an off-the-record profile and the authenticated loopback HTTP proxy. The interceptor blocks disallowed origins/paths/methods, workers and WebSockets; the proxy and broker recheck actual requests. The fixture includes a document, script, a `fetch`, a scripted click on a DOM link to a second page, a new `fetch`, an out-of-scope image and an out-of-scope redirect. The test requires six actual requests with equal broker and local-target counts and no authentication headers; the gate may also count requests denied before the broker. It confirms the image and redirect blocks and records five admitted observations, including the route loaded after the click. The JavaScript outcome is a fixture signal, not trustworthy evidence from an arbitrary page.

Since the fourth block, the lab starts Qt in a [supervised helper](ADR-011-BROWSER-HELPER.md). The proxy credential travels over `stdin` only after process supervision is installed. The parent bounds runtime and output, terminates descendants and evaluates redacted synthetic flags. Run the new supervisor and child-configuration tests with:

```sh
go test -race ./internal/browser -run '^TestRunHelper' -count=1
CGO_CXXFLAGS='-O0 -g0 -std=c++17' go test -tags=m3browserlab ./experiments/m3-browser
```

**Local trial on September 27, 2026, Apple Silicon/macOS 26.6.2, Qt WebEngine 6.11.2:** the command with the helper and DOM click returned `PASS` in four repeats; tagged Go tests and `go vet` passed. The macOS 26 CI job is separate and must be checked on the branch and merged commit. No equivalent Qt trial is documented yet on Windows. The lab is not a GUI feature and lacks independent egress, hard macOS/Linux process/memory limits, HTTPS and sessions; it does not satisfy the M3 browser criterion.

CI now includes a Linux Ubuntu 24.04 job with Qt WebEngine and a virtual display: the configuration test and synthetic fixture passed on branch `0da4e66` ([run 36281325352](https://github.com/Matte2599/WebFence/actions/runs/36281325352)). Check the merged commit separately. The local macOS trial was repeated after adding the job. This is not a trial of independent egress containment.

The first Linux attempt compiled the helper but returned `browser_helper_failed` during execution. The supervisor environment allowlist omitted `XAUTHORITY`, needed by the authenticated X11 virtual display; the same fixture passed after it was forwarded. That outcome supports the diagnosis without proving egress isolation.

The Windows runtime remains open: the desktop uses MIQT/MinGW, while [Qt WebEngine 6.11 does not compile with MinGW](https://doc.qt.io/qt-6/qtwebengine-platform-notes.html). A separate helper with a compatible Qt toolchain is a proposal requiring an ADR and trials, not an implemented capability.

CI for branch `4f7e2c0` showed a fifth attempt counted by the gate but denied before the broker: four requests reached the broker and target. The lab now compares those exact forwarded counts and requires the gate to count at least every forwarded request; extra attempts consume budget but are not treated as target traffic. This change needs fresh green CI.

The [CI for merge `cf504cf`](https://github.com/Matte2599/WebFence/actions/runs/36282644664) exposed a macOS lab timing failure: a fixed five-second timer stopped the page after document and script, before the `fetch` calls. The lab now exits once document, API, blocked redirect and blocked subresource are all observed, with a 12-second deadline and a 30-second parent limit. Five local macOS repetitions passed; the fix still needs green branch and merged CI.
