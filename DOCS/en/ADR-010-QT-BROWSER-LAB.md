# ADR-010 — Qt WebEngine in the M3 browser lab

[Italiano](../it/ADR-010-QT-BROWSER-LAB.md) · [M3 lab](M3-BROWSER-LAB.md) · [ADR-002](ADR-002-GUI.md) · [ADR-009](ADR-009-BROWSER-HTTP-BOUNDARY.md)

Date: 2026-09-26. Status: **adopted for the M3 lab**, not for a distributable desktop feature.

## Decision and evidence

The synthetic lab uses Qt WebEngine 6.11.2 through MIQT 0.14.0. This preserves the author's chosen Qt stack and provides an [off-the-record profile](https://doc.qt.io/qt-6/qwebengineprofile.html), a [request interceptor](https://doc.qt.io/qt-6/qwebengineurlrequestinterceptor.html) and [Qt Network proxy support](https://doc.qt.io/qt-6/qtwebengine-overview.html#proxy-support). The lab creates a local HTTP fixture, exposes it as a `.test` origin through a pinned loopback IP and uses the authenticated M3 proxy. A page loads a script, executes `fetch` and attempts a third-party subresource; the local trial observed four admitted GETs through the proxy/broker (document, script, API and redirect source) and a blocked subresource. The broker rejects the redirect destination at an excluded origin. The command has no external target input.

The lab has a dedicated build tag and is not included in the desktop or packages. macOS 26 CI checks it separately; results must be read from the run for the relevant commit. The local trial does not measure Chromium sandbox security or replace other-platform trials.

## Limits and review

The prototype uses an ephemeral profile and interceptor in the test executable's process. It does not yet impose a process/memory limit, an independent egress firewall or HTTPS mediation. Adopting Qt WebEngine for the product requires a separate helper, IPC that does not expose the proxy credential, containment tests on macOS/Windows/Linux, packaging and a review of additional license materials. If these constraints cannot be met, the driver must be reconsidered in a new ADR. The lab does not close the first M3 item.
