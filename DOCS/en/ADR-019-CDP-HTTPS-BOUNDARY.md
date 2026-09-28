# ADR-019 — Broker-mediated HTTPS in the Linux CDP lab

[Italiano](../it/ADR-019-CDP-HTTPS-BOUNDARY.md) · [Lab](M3-BROWSER-CDP-LAB.md) · [ADR-009](ADR-009-BROWSER-HTTP-BOUNDARY.md) · [ADR-017](ADR-017-CDP-BROKER-BOUNDARY.md) · [M3 status](M3-VALIDATION.md)

Date: 2026-09-28. Status: **adopted for the experimental Linux fixture only**.

## Context and decision

The initial proxy denies `CONNECT`: an opaque tunnel would prevent the gate and broker from checking each path, method and redirect. The CDP lab already intercepts requests inside the Chromium process and uses Unix sockets connected by the parent. We extend that path without adding a MITM CA to the browser: the helper sends an absolute-form HTTP request with a `https://` URL over an inherited socket; **only the Unix-socket proxy** admits this form. The TCP listener still denies it, and every form of `CONNECT` remains forbidden.

The gate checks URL, type and budget; the broker opens TLS to the pinned IP, verifies certificate and hostname, and rechecks policy and redirects. The helper opens no target network socket, does not choose TLS trust and forwards no renderer cookies or headers. For the loopback fixture, `NewAuthorizedLabWithPolicyTLSRoots` receives a copy of the synthetic server's ephemeral trust roots; the public constructor retains system trust. CDP presents the `https://` origin to the renderer without terminating TLS inside it.

## Verification and limits

The lab runs HTTP and HTTPS variants with a document, script and `fetch` under one run each; the HTTPS variant also checks `isSecureContext`. An out-of-scope image and outside redirect are stopped; the direct TCP canary receives `EPERM` with both absent container networking and `bridge` networking. A separate test admits the fixture certificate, rejects the wrong hostname and an outside origin without target contact, and checks that the TCP listener rejects absolute HTTPS. No external target is used.

This is an experimental Linux/container boundary, using Chromium `--no-sandbox` and a fixture certificate. It does not enable HTTPS in the desktop browser or prove safety for hostile pages, sessions, quotas outside the container or macOS/Windows runtimes. The first M3 item remains open under [ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md).
