# M1 — First HTTP check in the lab

[Italiano](../it/M1-HEADER-LAB.md) · [Managed runs](M1-MANAGED-RUNS.md) · [Scanning](SCANNING.md) · [Roadmap](../ROADMAP.md)

`scanner.RunHeaderLab(ctx, store, plan)` is the first core path from **saved project → managed run → authorized transport → result**. It uses only explicitly supplied seed URLs and loopback IP grants; it is not connected to the desktop and is not a production scanner. The caller must choose safe fixture paths: even GET can affect an application.

The plan requires a project ID, 1 to 32 distinct seed URLs, up to 32 grants with at most 16 IPs each, a resolver and mandatory broker limits. The component copies URLs and grants on entry, opens the current revision with `Store.BeginRun`, checks **all** seeds before sending a byte, and constructs `NewAuthorizedLab` with the managed scope. It performs one sequential GET per seed, without retries. The broker rechecks authorization, redirects, DNS/IP, TLS, budgets and timeouts on each request. Revocation, expiry and cancellation stop the run. An out-of-scope or duplicate plan URL prevents all run traffic.

## Rule and result

Initial rule `HTTP-XCTO-001` revision 1 observes `X-Content-Type-Options` on `2xx` responses with `Content-Type: text/html`. One `nosniff` value yields `observed`; absence yields `not_observed`; ambiguous/unrecognized values or an unparseable MIME type yield `inconclusive`. Non-HTML responses or inapplicable status codes yield `skipped`. A transport error yields `error` and stops further seeds. **`not_observed` is an observation, not an automatic vulnerability or severity.** This rule does not yet create product findings, CVEs or recommendations.

The `Report` contains project ID, authorization revision, planned/completed seed counts, HTTP attempts used (including redirects), checks with seed index, evidence code and redacted stop reason. `SeedsComplete` means only that **every explicit seed** returned a response: it does not indicate site coverage or safety. If an error occurs after one or more responses, a partial report is returned with the error. URLs, queries, raw headers, cookies and bodies are not retained in the result. Reports are not yet persisted or recovered after a crash.

Tests use only loopback `httptest` servers: `nosniff` present, absent, ambiguous value, nonapplicable MIME, invalid MIME, status 500, partial budget, out-of-scope seed, redirect to an unauthorized second origin, revocation during a request, and caller slice mutation after validation. The race detector exercises the path. No external target was contacted.

The next block adds [bounded HTML discovery](M1-DISCOVERY-LAB.md) on seed responses, without visiting links or automatically submitting forms. M1 still needs method/path/exclusion policy, rate limiting, a production broker, persistent run data and quota/recovery, IT/EN desktop results and visible incomplete coverage. The [human prerequisite plan](M1-PREREQUISITES.md) remains open.
