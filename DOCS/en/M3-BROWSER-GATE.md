# M3 — First block: browser request admission

[Italiano](../it/M3-BROWSER-GATE.md) · [Roadmap](../ROADMAP.md) · [Architecture](ARCHITECTURE.md)

`internal/browser.Gate` is the first M3 policy boundary, **not a runnable browser**. It requires a managed run scope, an explicit GET/HEAD path policy and request, concurrency and duration limits. The caller must apply `Admit` to every navigation, redirect, subresource and JavaScript request before networking and release the lease on completion. Every attempt, including a denial, consumes the budget; errors and counters contain no URLs or credentials. Revocation, expiry and cancellation also interrupt a request waiting for a slot.

The initial policy admits only GET/HEAD documents, subresources and `fetch` requests. It blocks workers and WebSockets until an adapter proves containment of those channels too. Origin and path checks reuse `project.RunScope` and `scope.RequestPolicy`: servers named by site content cannot expand scope. Requests with unsupported methods or schemes are denied.

The gate opens no sockets, does not know the connected IP, cannot stop a browser process from bypassing it and does not limit memory or process count. It is **not connected to the GUI or to a browser** and does not enable dynamic or authenticated scans. Before enabling a browser, an adapter, mandatory traversal of the pinned broker for every request, a verified process limit and negative tests for redirects, subresources, workers, WebSockets and alternate channels are required. An application interceptor alone does not replace an independent egress boundary.

Synthetic tests in this block cover valid admission, third-party origins, excluded and ambiguous paths, denied methods, local schemes, workers/WebSockets, budget, concurrency, close and revocation. They contact no external targets and do not yet measure SPA/API coverage.
