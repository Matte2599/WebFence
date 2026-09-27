# ADR-012 — Explicit POST and confined cookie for M3 test accounts

[Italiano](../it/ADR-012-TEST-SESSIONS.md) · [M3 sessions](M3-SESSIONS.md) · [ADR-009](ADR-009-BROWSER-HTTP-BOUNDARY.md)

Date: 2026-09-27. Status: **adopted for the synthetic M3 core**; it does not enable public login or authenticated browser use in the desktop.

## Decision

Login is a different operation from the M1 GET crawler. It requires a managed run and `LoginConfirmed`, an exact POST URL, pinned origin/IP, a bounded form body and a budget shared with GETs. It never follows a redirect carrying credentials. An in-memory manager selects one host-only cookie without a broad Domain attribute and with an explicit path; it checks the cookie only for the same scheme/origin and allowed path. A preceding anonymous GET must differ from the expected authenticated response. Identity and project stay bound to one specific broker, and the browser proxy neither reads client cookies nor returns `Set-Cookie`.

This is a conservative application of the cookie origin and path rules described by [RFC 6265](https://www.rfc-editor.org/rfc/rfc6265). Cookie Path alone is not a security boundary: the broker and run policy repeat checks before networking. Login and verification failures are separated from check outcomes.

The September 27 extension permits one extra anonymous GET only to the exact login URL after run confirmation. One declared hidden token can be added to POST; one declared host-only pre-session cookie may accompany it only on that POST. The authenticated cookie must be new. Page content cannot widen scope, method or submitted fields. This specializes the existing boundary rather than crawling the form.

The September 27 session revision applies the same strict selection to cookies returned by authenticated GETs. A new value for the same name/path is accepted only after an additional identity verification; requests within one session are serialized and the deadline is never extended. A second rotation during verification, an ambiguous cookie or a changed response closes the session. The extra verification uses the run budget.

## Limits

The form requires simple username/password fields and an exact validity marker; it does not discover other CSRF or federated flows automatically. This module does not persist the secret, but Go/OS memory copies cannot be reliably erased. The authenticated broker is loopback-only and the session is connected to the loopback GUI dialog, but not to the Qt helper. Continuous rotation on each verification or changes of name/path remain unsupported. Further trials are needed before M3 closure.
