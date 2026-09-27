# ADR-016 — Linux Chromium/CDP trial

[Italiano](../it/ADR-016-LINUX-CDP-TRIAL.md) · [Trial](M3-BROWSER-CDP-LAB.md) · [ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md) · [M3 status](M3-VALIDATION.md)

Date: 2026-09-27. Status: **experimental trial; no product runtime selected**.

## Context and decision

The experimental Qt scheme does not preserve HTTP origin semantics, and the baseline Ubuntu Qt version does not execute `fetch` on that scheme. A repeatable trial is needed that preserves a real HTTP origin while the browser process cannot open its own connections. The desktop remains Go and Qt Widgets/MIQT.

A separate Linux lab is added, built only with `m3cdplab`. The Go supervisor starts a helper with resource limits and a deadline; the helper installs the seccomp filter before creating headless Chromium. The CDP connection uses inherited pipes, not a TCP port. `Fetch.requestPaused` serves synthetic responses for an HTTP document, JavaScript and `fetch`, and blocks a resource and redirect outside the origin. The lab counts requests, bounds frames and duration, and checks `/proc` for `seccomp` and `no_new_privs` inherited by Chromium. CI runs it in a container without networking, with a read-only filesystem and aggregate memory/PID quotas.

This decision **does not adopt Chromium in the product**. The lab uses `--no-sandbox` to run unprivileged inside the container: its Linux filter remains active, but Chromium's internal renderer sandbox is disabled. It does not provide file/process protection suitable for hostile pages, nor connect CDP to the gate, broker, managed runs or GUI. Responses are hard-coded in the fixture; HTTPS, cookies, revocation, distribution and browser licensing are unresolved. Any target use first needs a complete OS boundary and broker-mediated adapter, with repeatable negative trials and dependency review.

Primary sources: [Chromium DevTools pipe](https://chromium.googlesource.com/chromium/src/+/HEAD/content/public/browser/devtools_agent_host.h), [CDP Fetch domain](https://chromedevtools.github.io/devtools-protocol/tot/Fetch/), [Chrome headless mode](https://developer.chrome.com/docs/automation-and-testing/headless).
