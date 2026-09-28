# M3 — Managed cross-role check execution

[Italiano](../it/M3-CROSS-ROLE-RUN.md) · [Rule](M3-CROSS-ROLE.md) · [Sessions](M3-SESSIONS.md) · [Roadmap](../ROADMAP.md)

`checks.RunCrossRole` binds the first `AUTH-CROSSROLE-001` check to **one managed run**. The plan declares a project, origin/pinned IP grant, GET policy, limits, exact login/verification URLs, two test accounts and one private resource. The operator separately confirms the login POST, resource and other role's lack of access. The two identities need distinct IDs, usernames, secret references and validity markers. Secrets come only from `session.SecretSource`; the orchestrator does not retain them.

The default mode and Qt dialog use `CrossRoleRunLoopback`. `CrossRoleRunPinnedPublic` must be selected explicitly and requires `SessionRoutes.PublicConfirmed` in addition to POST confirmation: it selects the [HTTPS broker with pinned public IPs](ADR-018-AUTHENTICATED-PUBLIC-TRANSPORT.md), one grant and its tighter limits. The plan rejects unknown modes, loopback IPs in public mode and non-loopback IPs in local mode. The [Qt dialog](M3-AUTH-DESKTOP.md) now offers this mode with a separate public confirmation and the tighter limits.

Before networking it validates the plan, current authorization, in-scope resource and route policy. One broker applies pinned IPs, budget, pacing, body size and revocation to both logins and the check. Sessions, broker and run close on every exit. A failed or ambiguous login yields `inconclusive`, never a negative outcome; interruption keeps an inconclusive outcome with its cause. The result contains only rule ID/revision, outcome and a redacted code. It is not written to the M1 database or M2 report.

A later fixture with confirmed rotation for both identities produces the expected finding with 15 requests under the same budget; rotation errors instead make dependent checks inconclusive.

**Local synthetic fixtures on September 27, 2026:** reproduced cross-role access and denied access (11 managed requests each), missing second secret, out-of-scope/unconfirmed plan and exhausted budget. Package race-detector tests passed; no external target. The later [desktop workflow](M3-AUTH-DESKTOP.md) exposes a two-account loopback trial.

**Core extension on September 28, 2026:** a run with explicit public mode constructs the HTTPS broker and fails inconclusively on denied synthetic DNS, without opening a socket. Missing confirmation, unknown mode or wrong IP type is rejected before networking. The broker's TLS trials and complete loopback two-account tests are separate: an end-to-end two-account trial on an authorized public staging target is still missing. Other CSRF flows, MFA/OIDC login and real accuracy measurements remain open.
