# M3 — Contextual cross-role check: first core

[Italiano](../it/M3-CROSS-ROLE.md) · [Sessions](M3-SESSIONS.md) · [Roadmap](../ROADMAP.md)

`internal/checks.CheckCrossRole` implements rule `AUTH-CROSSROLE-001` revision 1 on **two distinct, already authenticated test identities** bound to the same broker/run. The operator must confirm a private resource URL, an exact, non-secret body specific to its owner **and that the other identity is not entitled to access it**. The check re-verifies both sessions, reads the resource anonymously, then as the owner and finally as the other identity. Every request uses the loopback broker with scope, policy, pinned IPs and budgets; no credential is returned to the caller. The method follows the two-session comparison in [OWASP WSTG-ATHZ-02](https://wstg.owasp.org/latest/4-Web_Application_Security_Testing/05-Authorization/02-Bypassing_Authorization_Schema/), in a GET-only subset.

The rule emits a `finding` only if both the owner and other identity receive `200` with the same exact private body, while the anonymous response does not contain it. A marker already public, a different owner response, denied access for the other identity, an expired session or a network error yields **`inconclusive`**, never a “no issue” verdict. The result contains only rule ID/revision, outcome and evidence code; URLs, bodies and identities are absent. The caller owns session closure and credentials.

**Local synthetic fixtures on September 27, 2026:** six scenarios: reproduced cross-role access (1 `finding`), denied access, public marker with `200`, `403` or inside a larger response, and different owner response (5 `inconclusive`, no negative verdict). The race-detector test passed. These counts verify behavior on constructed cases; they do not estimate real sensitivity or specificity. Branch and merged CI must be checked separately.

The check does not prove that a different response is safe, infer ownership from OpenAPI or cover enumerated IDs, bearer tokens, multiple desktop roles, public grants or persisted reports. The M3 cross-role access and rule-expansion items remain open.
