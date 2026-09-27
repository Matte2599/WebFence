# M3 — Browser request observations

[Italiano](../it/M3-DYNAMIC-OBSERVATIONS.md) · [Qt lab](M3-BROWSER-LAB.md) · [Proxy](M3-BROWSER-PROXY.md) · [Roadmap](../ROADMAP.md)

`browser.NewObservedProxy` and `NewObservedProxyWithSession` retain at most 256 observations **in memory**, for one proxy/run instance. They record only GET/HEAD requests admitted by the gate and completed by the broker, with method, resource type, status and initial/final paths. Queries, headers, cookies, bodies and the proxy credential are absent from observations. Once the quota is full or a path exceeds 1,024 bytes, the omitted count increases. `Observations` returns a copy; callers cannot alter the internal record.

The Qt fixture runs JavaScript that starts `fetch('/app/api')`: the parent sees that route in the proxy, while an external-origin redirect and an excluded subresource do not become observations. The session variant uses the already verified cookie only in the parent and records the path without its query. Observations are untrusted data: they are not automatically scheduled as seeds and must not be persisted in the ledger without a redaction rule. The proxy remains HTTP and the browser is not integrated into the desktop yet.

**Local verification on September 27, 2026:** race-detector tests for capacity, copying, stripped query, denied requests, session and credential-bearing redirect; Qt macOS lab on a synthetic fixture. Branch and merged CI must be checked per commit. This block proves discovery of JavaScript requests, not crawling with DOM navigation or interaction: the third M3 item remains open.
