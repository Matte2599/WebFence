# Product vision and requirements

[Italiano](../it/PRODUCT.md) · [Index](../README.md)

Status: proposed specification, 2026-09-23.

## Problem and intended outcome

A working application is not necessarily a secure one. WebFence aims to make a repeatable cycle accessible: define authorized scope, discover reachable surfaces, collect evidence, prioritize problems, apply fixes and validate them. AI-assisted development motivates the project; classifying code as “human” or “AI” is not a prerequisite for using it.

The product is primarily DAST: it observes running applications. It does not promise access to all source code, internal dependencies or every business workflow. SBOM import and source analysis are separate extensions. Business-logic and authorization flaws often require multiple identities and operator-supplied context.

## Users and primary workflow

Initial users: individual developers and independent professionals, with free use including paid client assessments. Companies must obtain written authorization. The product will be a native desktop application; software licensing and authorization to assess a target remain separate requirements.

1. Create a project and record target ownership, authorization, expiry and allowed environments.
2. Define origins, ports, paths, exclusions, test identities and budgets.
3. Review the planned operations and start a scan with an appropriate profile.
4. Inspect inventory, progress, errors, evidence and coverage limitations.
5. Export a signed report and retain an immutable baseline.
6. After remediation, run a retest and regression assessment; compare results and coverage.
7. Export or delete the project and derived data.

## Traceable requirements

| ID | Requirement | Minimum verification | Phase |
| --- | --- | --- | --- |
| WF-01 | Enforce scope and authorization on every request | Block out-of-scope redirects and DNS destinations | M1 |
| WF-02 | Independently bounded analysis profiles | Measure load and cancellation against fixtures | M1 |
| WF-03 | Traditional checks with reproducible evidence | Vulnerable cases and corresponding fixed cases | M1–M3 |
| WF-04 | Updated CVE cache available offline | Interrupted sync preserves the last valid snapshot | M2 |
| WF-05 | Optional remote or local AI | Same pipeline remains usable with AI disabled | M5 |
| WF-06 | Signed IT/EN reports | Offline verification and rejection of altered artifacts | M2 |
| WF-07 | Persistent memory and deletion | Retain projects after restart; remove derived data | M1–M4 |
| WF-08 | Fix validation and regressions | “Inconclusive” for expired sessions or unreachable endpoints | M4 |
| WF-09 | Bilingual UI, CLI and reports | Translation-key parity and language-independent results | M1–M6 |
| WF-10 | Browser and authentication | Cover dynamic content without leaving scope | M3 |

## Initial product boundaries

The first alpha covers HTTP(S), bounded crawling and low-impact checks in a single installation. The useful MVP also includes verifiable reports and post-fix comparisons; it must not be called complete before M4. AI follows a measurable traditional baseline.

The MVP excludes a WAF, real-time protection, EDR agent, destructive exploitation, automatic production fixes, indiscriminate Internet scans, multi-tenant SaaS and compliance guarantees. Retesting is an explicit operation; retaining a project does not authorize continuous monitoring.

## Open decisions

- Establish supported operating systems through real tests, not compilation alone.
- Validate Fyne as the Go desktop GUI and select browser driver, AI providers and local model through prototypes and evaluations.
- Establish minimum hardware, benchmark corpus, private security contact and commercial terms.

Priorities and dependencies are in the [roadmap](../ROADMAP.md). No release date is committed.
