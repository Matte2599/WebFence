# M3 — Test accounts and cross-role check in the desktop

[Italiano](../it/M3-AUTH-DESKTOP.md) · [Managed run](M3-CROSS-ROLE-RUN.md) · [Sessions](M3-SESSIONS.md) · [Roadmap](../ROADMAP.md)

The Qt Widgets/MIQT scan window opens **Test accounts and cross-role access** for the selected project. This workflow is limited to **loopback** transport with pinned IPs and the already configured GET policy, budget and pacing. The operator enters exact POST login, GET verification and GET private-resource URLs, two distinct accounts with different verification markers, the owner's exact private body and form/cookie field names. They separately confirm the POST, resource ownership and the other account's lack of access. The three confirmations reset when a run starts and when the dialog closes: every new run requires them again. No request starts without the confirmations.

Two optional fields declare the hidden CSRF token name and pre-session cookie name. When supplied, the run reads an HTML page from the exact login URL for each account and sends token and cookie only to the already confirmed POST. Every request uses the shared budget and pinned IPs; redirects are not followed.

Passwords in masked fields are used for **one run only**; fields are cleared when a run starts, the dialog closes and the window is disposed. An in-memory secret source returns copies to the manager and zeroes its owned bytes after the run; Qt, Go or OS copies may remain in memory. Passwords are not saved in the project, keychain, ledger or report. Closing the dialog or pressing Cancel cancels an active run. This check and the M1 scan cannot start together from the same window. Outcome and redacted code stay in the dialog only, without URLs, bodies, accounts or credentials. Failed login, interruption and ambiguous responses are never presented as “no issue”.

**Local verification on September 27, 2026:** IT/EN Qt offscreen self-test on the `httptest` server also used by the synthetic scan. Closing with unsubmitted passwords, missing confirmations and a second run without renewed confirmations: zero requests and cleared fields. Two confirmed logins and a private resource reproduced by the other account: 11 loopback requests under one run and code `cross_role_private_body_reproduced` without sensitive data. The full Go suite, targeted race detector, `go vet` and local documentation links passed; branch/merged CI must be checked per commit.

The later CSRF trial in the IT/EN Qt self-test completed two logins with separate pre-sessions and 13 loopback requests, again confirming the redacted outcome. The Go suite and targeted race detector passed; branch and merged CI must be checked for the relevant commit.

The session service used by this dialog also confirms bounded cookie rotation, with the extra validity request charged to the run budget; core and managed cross-role fixtures cover that variant. The dialog self-test still uses stable cookies.

This is a narrow trial, not a general authorization test. It does not cover public targets, other CSRF flows, MFA/OIDC login, bearer tokens, authenticated JavaScript browsing, persistent multiple roles, report state or real-world accuracy measurement. The second and third M3 criteria remain open.
