# ADR-013 — Unix socket and Qt scheme in the browser lab

[Italiano](../it/ADR-013-BROWSER-UNIX-SCHEME.md) · [Lab](M3-BROWSER-UNIX-SCHEME.md) · [ADR-011](ADR-011-BROWSER-HELPER.md) · [M3 status](M3-VALIDATION.md)

Date: 2026-09-27. Status: **adopted only for the synthetic lab**.

## Decision

A second macOS lab path connects the parent and Qt helper through a Unix socket in a private directory, using the same gate, broker and ephemeral secret as the HTTP proxy. The helper registers `wfsite` as a Qt scheme before creating the application and forwards only three fixture GET resources to the parent. This exercises a document, external script and `fetch` without a TCP listener for that path. The existing HTTP proxy trial remains in place for redirect and out-of-scope checks.

The Qt `FetchApiAllowed` flag, available since Qt 6.6, is required for `fetch` on a custom scheme. MIQT 0.14.0 does not name the flag, so the lab uses the Qt value `0x100`. The Ubuntu 24.04 job has Qt 6.4; it runs the existing HTTP trial and Unix proxy tests, but not this Qt path.

## Boundaries and limitations

The Unix socket does not prevent Qt or its child processes from opening TCP connections: it is not an egress boundary. The scheme handler allows only three fixture paths and does not mediate an arbitrary website. `QWebEngineUrlRequestJob::reply` exposes MIME and body but cannot faithfully reproduce HTTP status and headers; the scheme also changes the web origin. Therefore this path is not yet adopted in the desktop and is not presented as a replacement for the HTTP(S) proxy.

The first M3 item still requires an OS mechanism denying every direct browser-group egress route, faithful policy-bound HTTP(S) mediation, memory/process quotas and negative trials on target platforms. That mechanism will need its own ADR and tests; Chromium flags and the Qt interceptor are insufficient.
