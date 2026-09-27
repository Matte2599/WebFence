# M3 — Explicit visits to observed `fetch` routes

[Italiano](../it/M3-OBSERVED-CRAWL.md) · [Observations](M3-DYNAMIC-OBSERVATIONS.md) · [Managed runs](M1-MANAGED-RUNS.md) · [Roadmap](../ROADMAP.md)

`scanner.RunObservedCrawl` accepts up to 256 ephemeral proxy observations and **replays only up to 32 indices explicitly selected by the operator**, after separate confirmation. It admits only completed `2xx` GET `fetch` requests with identical initial/final paths and no query. Replay uses the observed path without a query, so it can have different semantics from the original JavaScript request. It does not replay POST, cookies, credentials, bodies, links or redirects.

Observations and paths are untrusted input, not authorization. The origin is declared separately; the service rejects duplicate selections, ambiguous paths and plans with implicit seeds or links. A fresh `RunCrawl` rechecks current authorization, scope, GET policy, pinned IP grants and budget before networking. Results and the ledger contain only redacted codes and counts. This core service is not connected to the GUI and never starts the browser or replay on its own.

**Local synthetic fixtures on September 27, 2026:** two selected JSON routes visited under one run, unselected route never visited, private body absent from the result; missing confirmation, disallowed method/type/status, changed path, query and invalid grant rejected without requests; redirect to an unselected route stopped after one attempt. The full Go suite, targeted race detector, `go vet` and local documentation links passed; branch/merged CI must be checked per commit. This is not a real coverage or accuracy measurement.

DOM navigation and interaction, an isolated product browser, safe handling of dynamic queries/parameters and API-specific checks remain open. The third M3 item is not closed.
