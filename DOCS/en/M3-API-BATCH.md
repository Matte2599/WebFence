# M3 — Explicit static API visits

[Italiano](../it/M3-API-BATCH.md) · [OpenAPI import](M3-OPENAPI-IMPORT.md) · [Controlled visits](M1-CONTROLLED-CRAWL.md) · [Roadmap](../ROADMAP.md)

`scanner.RunAPICrawl` accepts an OpenAPI JSON document and **1 to 32 GET paths explicitly selected by the operator**. It imports the document offline under a brief managed run and accepts only static paths classified as candidates by the importer. The origin, GET policy, pinned IP grants and budgets come from the declared plan, never from document `servers` or `$ref`. Duplicate, absent, templated, authenticated, excluded or other-method paths are rejected before networking.

A fresh managed run passes the seeds to `RunCrawl`, which rechecks authorization, scope, policy and pinned transport before the first request. Visits are sequential, without following links or redirects or submitting forms; a redirect stops the run as partial. The M1 crawler's request and body limits still apply. The ledger stores redacted codes and counts, without URLs or response bodies. An authorization revision between import and start requires the new check; import grants no independent access.

**Local verification on September 27, 2026:** an `httptest` fixture with two selected JSON routes, inert external `servers`, two GET requests and a redacted ledger; denied or ambiguous selections without requests; package race detector. Branch and merged CI must be checked per commit. The [desktop](M3-API-DESKTOP.md) later integrates multiple selection. It performs no API JSON-specific security checks, parameterized routes, OpenAPI authentication or JavaScript crawling. The third M3 criterion remains open.
