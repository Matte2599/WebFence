# ADR-017 — Broker-mediated CDP in the Linux lab

[Italiano](../it/ADR-017-CDP-BROKER-BOUNDARY.md) · [Lab](M3-BROWSER-CDP-LAB.md) · [ADR-016](ADR-016-LINUX-CDP-TRIAL.md) · [M3 status](M3-VALIDATION.md)

Date: 2026-09-27. Status: **experimental trial, synthetic fixtures only**.

## Context and decision

The first CDP trial served hard-coded responses inside the helper: it preserved HTTP origin in the renderer and demonstrated inheritance of the Linux filter, but did not exercise WebFence policies. The experimental path is now connected to a local `httptest` origin through a declared project/run, gate, broker with a pinned loopback IP and authenticated Unix proxy. The parent opens eight proxy connections before starting the helper; the child receives them as descriptors and installs seccomp before Chromium. The child's HTTP client can use only these descriptors and does not follow redirects on its own. CDP intercepts every request and returns the proxy's status, bounded body and essential response type to the renderer.

The fixture verifies four local target contacts: document, script, `fetch` and an endpoint returning an outside redirect. The gate denies the outside resource without contacting the target; the broker rejects the redirect after the initial response. Target counts, gate/broker budgets and redacted observations are checked. Unknown requests consume the CDP budget. The networkless CI container adds aggregate memory and PID quotas to the trial.

This is not a product browser adapter. The lab remains Linux/container only, with Chromium `--no-sandbox`, one fixture, HTTP only and an ephemeral proxy credential in memory. Filesystem sandboxing, containment outside the container, TLS/HTTPS, cookies and sessions, revocation during a page, user profiles and desktop integration need new trials. The Go/Qt Widgets/MIQT desktop and ADR-015 OS-isolation requirement are unchanged; the first M3 item remains open.
