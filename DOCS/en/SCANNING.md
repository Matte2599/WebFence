# Scanning engine and profiles

[Italiano](../it/SCANNING.md) · [Index](../README.md)

Status: production engine not implemented. The first [origin check](M0-SCOPE.md), [controlled transport](ADR-003-TRANSPORT.md), [M1 operator-declared authorization snapshot](M1-PROJECT-AUTHORIZATION.md), a [first HTTP check on explicit seeds](M1-HEADER-LAB.md), [observational HTML discovery](M1-DISCOVERY-LAB.md) and [controlled visits with pinned public grants](M1-CONTROLLED-CRAWL.md) are available in the M1 desktop. The [M3 browser gate](M3-BROWSER-GATE.md) generates no traffic yet. Methodological reference: [OWASP WSTG](https://owasp.org/projects/web-security-testing-guide), with identifiers pinned to the version used by rules.

## Pipeline

`authorization → discovery → check planning → execution → evidence → normalization → correlation → report`.

The plan records rule versions, configuration, environment, credential references and intelligence snapshot. Normalize URLs without merging paths or parameters the application distinguishes. Deduplication considers origin, method, route, parameter location and authentication context, not just URL.

Initial discovery covers HTTP(S) links and observed forms without automatic submission. Explicit API specification imports, JavaScript browsers, login workflows and multiple identities arrive in M3. Imported OpenAPI documents are untrusted: their server entries do not expand scope.

The M1 core can sequentially visit HTTP links with `RunCrawl` only by explicit opt-in and within the [implemented policy limits](M1-CONTROLLED-CRAWL.md); forms stay observational. The profile table below remains a product proposal, not automatic configuration of that core.

## Scope

Scope includes scheme, host, ports, authorized CIDRs where applicable, paths, methods, exclusions and expiry. No implicit wildcards, automatic subdomain expansion or trust in links found on a site. Third-party resources are recorded as uncovered; permission to load a dependency does not authorize active testing against it.

Every new connection and redirect checks DNS, IPv4/IPv6, the actual connected address and egress policy, avoiding a second uncontrolled resolution. Block private networks, loopback, link-local and cloud metadata by default; a local lab can explicitly allow exact hosts and ports. Control endpoints, stores and AI runtimes remain excluded as targets. Providers and feeds have separate connection policies.

## Proposed profiles

Initial values for controlled testing, to be measured: these are neither implemented limits nor guarantees of no side effects.

| Profile | Operations | Global / per-origin concurrency | Requests per second per origin | Request budget / duration |
| --- | --- | --- | --- | --- |
| Observation | Analyze imported traffic; no new target traffic | 0 / 0 | 0 | 0 / 15 min |
| Cautious | Discovery and low-impact checks; no form submission | 2 / 1 | 1 | 500 / 15 min |
| Standard | Selected active checks and authorized test identities | 4 / 2 | 2 | 3,000 / 60 min |
| Deep | Greater coverage and budgets, explicitly enabled | 8 / 4 | 4 | 10,000 / 120 min |

Shared initial limits: depth 5, decompressed body 2 MiB, request timeout 15 s, at most 5 redirects and 2 compatible retries. Count attempts, retries, redirects and browser requests, not just unique URLs. Hitting a limit produces explicitly partial coverage. AI has a separate budget.

Depth and intrusiveness are separate axes. No profile automatically enables deletions, purchases, email sending, persistent uploads, brute force, denial of service or data extraction. Even GET can have effects: exclude logout and known actions, use staging and test accounts. The product cannot promise to recognize every effect of an unknown endpoint.

## Rule contract

Each check declares a stable ID, revision, prerequisites, impact, budget, compatible targets, expected evidence, possible false positives, applicable CWE/WSTG mappings and IT/EN text. It uses only the controlled transport. It returns one of `observed`, `not_observed`, `inconclusive`, `skipped`, `error`, with reasons and evidence references.

Initial families: TLS/HTTP configuration, cookies, exposures and limited contextual checks. XSS, injection and cross-identity access checks are introduced with positive and negative fixtures and minimal evidence, without promising coverage of an entire category. A missing header does not automatically imply high severity.

The stop mechanism responds to cancellation, overall timeout, quota, repeated target failures and overload signals. No automatic retry of non-repeatable actions; cancelled and partial runs retain a redacted summary.
