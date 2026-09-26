# M3 — Offline OpenAPI import: core inventory

[Italiano](../it/M3-OPENAPI-IMPORT.md) · [Roadmap](../ROADMAP.md) · [Scanning](SCANNING.md)

`internal/apiimport.Import` reads an OpenAPI 3.0.x or 3.1.x JSON document in memory and returns an ordered operation inventory. This is a **core block**, not yet a GUI importer or a scan plan. Import opens no connections, visits no routes and resolves no `$ref`.

The operator must separately supply an exact origin already admitted by `project.RunScope` and a valid `scope.RequestPolicy`. Document `servers` do not change the origin. Only static GET/HEAD operations without declared OpenAPI security requirements and admitted by origin/path policy receive a candidate URL. Other operations remain in the inventory with an exclusion code: template requiring explicit values, authentication, method, policy, ambiguous path or unresolved reference. A candidate is never executed automatically: future explicit selection must still pass through the broker with IP grants, budgets, pacing and per-request checks.

The parser rejects duplicate JSON keys, non-UTF-8 input, unsupported syntax or version, and documents above 1 MiB, depth 32, 20,000 tokens, 512 paths or 1,024 operations. It does not interpret schemas, parameters, examples, `callbacks`, links or complex security requirements; template paths are not expanded. The inventory is ephemeral and may contain sensitive path names: it must not be copied into the ledger or reports without a redaction contract.

**Local verification on September 27, 2026:** synthetic fixtures for ignored external `servers`, origin and path policy, security, `$ref`, templates, non-read methods, duplicate keys and size/depth bounds. No external target was contacted. This block does not close the M3 item that also includes dynamic crawling, contextual checks and cross-role access testing.
