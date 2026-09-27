# ADR-016 — Linux Chromium/CDP trial

[Italiano](../it/ADR-016-LINUX-CDP-TRIAL.md) · [Trial](M3-BROWSER-CDP-LAB.md) · [Follow-up ADR-017](ADR-017-CDP-BROKER-BOUNDARY.md) · [ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md)

Date: 2026-09-27. Status: **experimental trial; no product runtime selected**.

## Context and decision

The experimental Qt scheme does not preserve HTTP origin semantics, and the baseline Ubuntu Qt version does not execute `fetch` on that scheme. A repeatable trial is needed that preserves a real HTTP origin while the browser process cannot open its own connections. The desktop remains Go and Qt Widgets/MIQT.

The first version added a separate Linux lab, built only with `m3cdplab`. The Go supervisor started a helper with resource limits and a deadline; the helper installed the seccomp filter before creating headless Chromium. The CDP connection used inherited pipes, not a TCP port. `Fetch.requestPaused` served hard-coded synthetic responses for an HTTP document, JavaScript and `fetch`, and blocked a resource and redirect outside the origin. The lab counted requests, bounded frames and duration, and checked `/proc` for `seccomp` and `no_new_privs` inherited by Chromium. CI ran it in a container without networking, with a read-only filesystem and aggregate memory/PID quotas. [ADR-017](ADR-017-CDP-BROKER-BOUNDARY.md) records the subsequent gate/broker connection.

This decision **does not adopt Chromium in the product**. The lab uses `--no-sandbox` to run unprivileged inside the container: its Linux filter remains active, but Chromium's internal renderer sandbox is disabled. It does not provide file/process protection suitable for hostile pages. HTTPS, cookies, revocation, distribution and browser licensing are unresolved. Any target use first needs a complete OS boundary, repeatable negative trials and dependency review.

Primary sources: [Chromium DevTools pipe](https://chromium.googlesource.com/chromium/src/+/HEAD/content/public/browser/devtools_agent_host.h), [CDP Fetch domain](https://chromedevtools.github.io/devtools-protocol/tot/Fetch/), [Chrome headless mode](https://developer.chrome.com/docs/automation-and-testing/headless).
