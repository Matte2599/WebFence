# ADR-012 — Explicit POST and confined cookie for M3 test accounts

[Italiano](../it/ADR-012-TEST-SESSIONS.md) · [M3 sessions](M3-SESSIONS.md) · [ADR-009](ADR-009-BROWSER-HTTP-BOUNDARY.md)

Date: 2026-09-27. Status: **adopted for the synthetic M3 core**; it does not enable public login or authenticated browser use in the desktop.

## Decision

Login is a different operation from the M1 GET crawler. It requires a managed run and `LoginConfirmed`, an exact POST URL, pinned origin/IP, a bounded form body and a budget shared with GETs. It never follows a redirect carrying credentials. An in-memory manager selects one host-only cookie without a broad Domain attribute and with an explicit path; it checks the cookie only for the same scheme/origin and allowed path. A preceding anonymous GET must differ from the expected authenticated response. Identity and project stay bound to one specific broker, and the browser proxy neither reads client cookies nor returns `Set-Cookie`.

This is a conservative application of the cookie origin and path rules described by [RFC 6265](https://www.rfc-editor.org/rfc/rfc6265). Cookie Path alone is not a security boundary: the broker and run policy repeat checks before networking. Login and verification failures are separated from check outcomes.

## Limits

The form requires simple username/password fields and an exact validity marker; it does not discover CSRF or federated flows automatically. This module does not persist the secret, but Go/OS memory copies cannot be reliably erased. The authenticated broker is loopback-only and the session is not connected to the GUI or Qt helper yet. Cookies updated after login are not captured. Further trials are needed before M3 closure.
