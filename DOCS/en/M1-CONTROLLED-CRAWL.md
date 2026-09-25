# M1 — Controlled HTTP visits

[Italiano](../it/M1-CONTROLLED-CRAWL.md) · [ADR-008](ADR-008-PINNED-PUBLIC-TRANSPORT.md) · [Roadmap](../ROADMAP.md)

`scanner.RunCrawl` connects a SQLite project to a managed run, a policy-bound HTTP broker and a bounded BFS queue. The basic workflow is now exposed in the M1 GUI, but this is not a production scanner. No external target was contacted during development or testing of this block. The operator must own or have explicit permission for the origins, addresses and activities; the project declaration does not independently prove that permission.

## Operating contract

- `CrawlPlan` requires a project ID, 1–32 explicit seeds, `loopback` or `pinned_public` mode, exact IP grants per origin, a `scope.RequestPolicy`, limits, at most 256 pages, depth 0–5 and explicit `FollowLinks`. Every seed is copied and preflighted **before** traffic. A changed authorization revokes the run within the same store/process.
- `scope.NewRequestPolicy` requires at least one `GET`/`HEAD` method and an allowed path prefix. Exclusions win. Matching respects segment boundaries: `/docs` includes `/docs/a`, not `/document`. This policy **does not** widen origin scope. Paths containing percent encoding, double slash, backslash, `;`, `.`/`..` segments or control characters are conservatively rejected; query meaning is not filtered. Callers must exclude routes with side effects, including GET logout or mutations.
- The crawler issues only `GET`; `HEAD` is available only through an explicit `Broker.FetchMethod` call if the policy permits it. No POST, form submission, cookies, credentials, JavaScript or automatic retries. HTML links are followed only with `FollowLinks`; form actions are observed but never scheduled. Every candidate and redirect hop rechecks origin, method/path, DNS, IP and budget.
- The queue is sequential and deduplicates canonical URLs and final redirect destinations. Page/depth limits or exclusions yield partial-coverage counts. `QueueDrained` means only that the admitted queue finished, **not** that the site was covered. The report contains project ID, revision, codes, counts and the first HTTP rule status; it contains no URLs, queries, headers or bodies. Exact references live only in memory during the run.

## Transport and pacing

`transport.NewAuthorizedLabWithPolicy` keeps loopback grants. `transport.NewAuthorizedPublic` requires a lifecycle-bound scope, explicit public grants for each origin, a valid policy, one concurrent request/chain, at least **100 ms** between attempts per origin, no more than 10,000 attempts and a run duration of at most four hours. Caller-selected stricter limits still apply. An attempt is reserved before DNS; DNS failures, redirects and canceled pacing waits consume attempts. Pacing occurs after resolution and before connection; a canceled time slot is not reused.

On every hop, **all** resolver answers must be in the origin's grant. The broker selects one deterministic IP, connects directly to `IP:port`, verifies the actual peer, keeps the original Host and SNI and verifies TLS; proxies and pooling are disabled. Public mode rejects private, loopback, link-local, metadata and special-use ranges even if granted. The classifier is a conservative snapshot of the [IANA IPv4](https://www.iana.org/assignments/iana-ipv4-special-registry/) and [IANA IPv6](https://www.iana.org/assignments/iana-ipv6-special-registry/) registries reviewed on September 25, 2026; it does not promise global routability. See [ADR-008](ADR-008-PINNED-PUBLIC-TRANSPORT.md).

## Limits and evidence

Pacing is per broker/run. The store permits one persistent crawler run per instance and a file lock excludes a second process on the same DB; other programs or separate DBs are not coordinated. Public IP grants require operator selection and maintenance; the GUI accepts manually pinned IPs without auto-suggesting them. Redacted persistence, a DB quota, interrupted-run marking on restart and desktop integration now exist. Target load feedback, browser, authentication and a system egress firewall remain absent. The path policy alone cannot determine whether a GET has side effects. Do not start external visits using documentation examples. [M1 verification and gates](M1-VALIDATION.md).

Synthetic tests exercise prefixes/exclusions, ambiguous escapes, methods, preflight, redirects, mixed DNS, special IPs, peer pinning, pacing, cancellation/revocation, page/depth limits, ignored forms and report redaction. The public-path test uses a loopback `httptest` server and a fake dialer simulating a public peer: it is **not** a live public-network test. The M1 human trial plan remains [open](M1-PREREQUISITES.md).
