# M3 — Managed cross-role check execution

[Italiano](../it/M3-CROSS-ROLE-RUN.md) · [Rule](M3-CROSS-ROLE.md) · [Sessions](M3-SESSIONS.md) · [Roadmap](../ROADMAP.md)

`checks.RunCrossRole` binds the first `AUTH-CROSSROLE-001` check to **one loopback-only managed run**. The plan declares a project, origin/pinned loopback IP grant, GET policy, limits, exact login/verification URLs, two test accounts and one private resource. The operator separately confirms the login POST, resource and other role's lack of access. The two identities need distinct IDs, usernames, secret references and validity markers. Secrets come only from `session.SecretSource`; the orchestrator does not retain them.

Before networking it validates the plan, current authorization, in-scope resource and route policy. One broker applies pinned IPs, budget, pacing, body size and revocation to both logins and the check. Sessions, broker and run close on every exit. A failed or ambiguous login yields `inconclusive`, never a negative outcome; interruption keeps an inconclusive outcome with its cause. The result contains only rule ID/revision, outcome and a redacted code. It is not written to the M1 database or M2 report.

**Local synthetic fixtures on September 27, 2026:** reproduced cross-role access and denied access (11 managed requests each), missing second secret, out-of-scope/unconfirmed plan and exhausted budget. Package race-detector tests passed; no external target. Branch and merged CI must be checked per commit. The desktop does not yet offer the account/check workflow; public grants, CSRF/MFA/OIDC login and real accuracy measurements remain open.
