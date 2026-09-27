# M3 — Third block: Qt WebEngine lab

[Italiano](../it/M3-BROWSER-LAB.md) · [ADR-010](ADR-010-QT-BROWSER-LAB.md) · [ADR-011](ADR-011-BROWSER-HELPER.md) · [Proxy](M3-BROWSER-PROXY.md) · [Roadmap](../ROADMAP.md)

`experiments/m3-browser` is a repeatable trial **using only fixtures created by the command**. Run it with Qt 6 WebEngine/MIQT 0.14.0 and C++17 flags:

```sh
CGO_CXXFLAGS='-O0 -g0 -std=c++17' QT_QPA_PLATFORM=offscreen go run -tags=m3browserlab ./experiments/m3-browser
```

The program accepts no target URLs or arguments. A `.test` origin is connected only to its `httptest` server through the broker with a pinned `127.0.0.1` IP. Qt WebEngine uses an off-the-record profile and the authenticated loopback HTTP proxy. The interceptor blocks disallowed origins/paths/methods, workers and WebSockets; the proxy and broker recheck actual requests. The fixture includes a document, script, a `fetch`, a scripted click on a DOM link to a second page, an external script and a new `fetch`, an out-of-scope image and an out-of-scope redirect. The test requires seven actual requests with equal broker and local-target counts and no authentication headers; the gate may also count requests denied before the broker. It confirms the image and redirect blocks and records six admitted observations, including the route loaded after the click. The JavaScript outcome is a fixture signal, not trustworthy evidence from an arbitrary page.

Since the fourth block, the lab starts Qt in a [supervised helper](ADR-011-BROWSER-HELPER.md), which also applies [Unix resource limits](M3-BROWSER-RESOURCE-LIMITS.md). The proxy credential travels over `stdin` only after process supervision is installed. The parent bounds runtime and output, terminates descendants and evaluates redacted synthetic flags. Run the supervisor and child-configuration tests with:

```sh
go test -race ./internal/browser -run '^TestRunHelper' -count=1
CGO_CXXFLAGS='-O0 -g0 -std=c++17' go test -tags=m3browserlab ./experiments/m3-browser
```

**Local trial on September 27, 2026, Apple Silicon/macOS 26.6.2, Qt WebEngine 6.11.2:** the command with the helper and DOM click returned `PASS` in four earlier repeats and after the new Unix limits; tagged Go tests passed. The macOS 26 CI job is separate and must be checked on the branch and merged commit. No equivalent Qt trial is documented yet on Windows. The lab is not a GUI feature and lacks independent egress, macOS memory or aggregate process/memory quotas on macOS/Linux, HTTPS and sessions; it does not satisfy the M3 browser criterion.

CI now includes a Linux Ubuntu 24.04 job with Qt WebEngine and a virtual display: the configuration test and synthetic fixture passed on branch `0da4e66` ([run 36281325352](https://github.com/Matte2599/WebFence/actions/runs/36281325352)). Check the merged commit separately. The local macOS trial was repeated after adding the job. This is not a trial of independent egress containment.

The first Linux attempt compiled the helper but returned `browser_helper_failed` during execution. The supervisor environment allowlist omitted `XAUTHORITY`, needed by the authenticated X11 virtual display; the same fixture passed after it was forwarded. That outcome supports the diagnosis without proving egress isolation.

The Windows runtime remains open: the desktop uses MIQT/MinGW, while [Qt WebEngine 6.11 does not compile with MinGW](https://doc.qt.io/qt-6/qtwebengine-platform-notes.html). A separate helper with a compatible Qt toolchain is a proposal requiring an ADR and trials, not an implemented capability.

CI for branch `4f7e2c0` showed a fifth attempt counted by the gate but denied before the broker: four requests reached the broker and target. The lab now compares those exact forwarded counts and requires the gate to count at least every forwarded request; extra attempts consume budget but are not treated as target traffic. This change needs fresh green CI.

The [CI for merge `cf504cf`](https://github.com/Matte2599/WebFence/actions/runs/36282644664) exposed a macOS lab timing failure: a fixed five-second timer stopped the page before the `fetch` calls. Fix `24a7523` passed final CI. The [DOM merge CI for `fd6be19`](https://github.com/Matte2599/WebFence/actions/runs/36289947405) then reached the second page without running its inline script and second `fetch` within 12 seconds: five target requests and four observations. The fixture now loads an external script from the second page and waits for both dynamic signals for at most 20 seconds, under the 30-second parent limit. Ten local macOS repetitions passed; fresh CI must be checked per commit.
