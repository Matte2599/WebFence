# M3 — Feasibility of the browser boundary by platform

[Italiano](../it/M3-BROWSER-PLATFORM-FEASIBILITY.md) · [ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md) · [M3 status](M3-VALIDATION.md) · [Roadmap](../ROADMAP.md)

Local trials on September 27, 2026, on Apple Silicon/macOS 26.6.2, using only synthetic pages and loopback addresses. The probe programs and ad hoc signed bundles are temporary and outside the repository: these results must be reproduced with versioned material before a product choice. No external target was contacted.

| Trial | Observed outcome | Limit of inference |
| --- | --- | --- |
| Minimal App Sandbox bundle without `network.client`, Go program opening TCP to `127.0.0.1:9` | `operation not permitted`; the signed control without sandbox runs. | Confirms TCP denial in the trial bundle, not a working browser engine. |
| Qt WebEngine 6.11.2, synthetic HTML page | Without App Sandbox, `LOADED true`; in the App Sandbox bundle, after bundling the Qt offscreen plugin, Chromium fails at `bootstrap_check_in org.chromium.Chromium.MachPortRendezvousServer.<pid>: Permission denied (1100)`. `QTWEBENGINE_DISABLE_SANDBOX=1` also fails. | Another packaging arrangement has not been ruled out; [Qt documentation](https://doc.qt.io/qt-6/qtwebengine-platform-notes.html) warns that App Sandbox interferes with Chromium initialization. |
| Native WebKit with a `wfsite` scheme, synthetic document, script and `fetch` | Passes without App Sandbox; the App Sandbox bundle without `network.client` times out without a loaded page; with `network.client` it passes; removing permission makes it time out again. | Outbound permission allows arbitrary TCP connections according to [Apple](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.security.network.client); it is not a page egress boundary. Not every WebKit configuration was explored. |
| Linux ARM64 container and Ubuntu 24.04 job | The [seccomp filter](M3-BROWSER-LINUX-NETWORK.md) denies direct INET sockets while Unix IPC works; the branch's Linux Qt lab passed. | Unix sockets to other local services remain reachable; the job's Qt 6.4 does not test scheme `fetch`. |
| Windows 10/11 | No browser runtime trial. [Qt WebEngine](https://doc.qt.io/qt-6/qtwebengine-platform-notes.html) does not compile with the desktop's MinGW toolchain. | The supervisor's Job Object does not replace a Windows browser and network trial. |

Baseline Ubuntu 24.04.0 uses Linux kernel 6.8 ([Canonical](https://documentation.ubuntu.com/kteam-docs/public/reference/hwe-kernels.html)); Landlock restriction of pathname Unix socket resolution is available only from ABI 9 ([Linux kernel](https://cdn.kernel.org/doc/html/latest/userspace-api/landlock.html)). HWE updates can change the installed kernel, so capability must be detected at runtime. This matrix motivates [ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md), but does not close the first M3 item.
