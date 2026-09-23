# Quality, benchmarks and release criteria

[Italiano](../it/QUALITY.md) · [Index](../README.md)

Status: engine quality plan, without scanning benchmarks. Initial offline GUI tests and accessibility limitations are recorded in the [M0 report](M0-DESKTOP.md). Professional-coverage ambitions are not claims of equivalence to commercial products.

## Corpus

Use owned or explicitly authorized, reproducible and isolated labs. Each rule needs a vulnerable case, fixed case, non-applicable case and ambiguous case. Add redirects, changing DNS, IPv6, expired authentication, simulated WAFs, compressed bodies, endless pagination and network limits. Fixtures must not contact public systems during tests.

Separate development and evaluation by application/template family, not URLs within the same application. Version data, labels, rules, engine and hardware. For AI, retain model, prompt, parameters, costs and output; repeat unstable cases and do not use the same LLM as the sole label source.

## Metrics

| Metric | Definition |
| --- | --- |
| Precision | `TP / (TP + FP)`, by family and verification level |
| Recall | `TP / (TP + FN)` against known corpus cases, not all real vulnerabilities |
| Coverage | Discovered endpoints/contexts and executed applicable checks; show numerator and denominator |
| Stability | Disagreements across repetitions and their causes |
| Cost | Requests, duration, RAM, CPU, disk and any AI tokens/spend |
| Retest | Correctly verified fixes, false “fixed” results and inconclusive outcomes |

A zero denominator means “not computable”. Blocked checks are not true negatives. Deduplicate under a published rule before computing metrics. Report sample size and uncertainty; a percentage from a few cases is not a reliable benchmark.

## Proposed gates

- Every check has positive/negative tests and a verifiable severity/confidence rationale.
- No scope, isolation, secret, import or signature test violates its boundary; any failure blocks the affected release.
- No false `fixed_verified` in the retest corpus's failure/ambiguity cases.
- Initial precision target of at least 95% for high-confidence findings, per family on a declared corpus; also publish sample size and uncertainty interval. This is not an achieved result.
- Publish recall by family; do not promote a rule by concealing uncovered cases or artificially reducing the denominator.
- Cancellation target: stop new dispatches within 1 second; in-flight requests respect cancellation and configured timeouts. Measure both core and browser paths.
- Responsive UI on a large corpus, keyboard navigation and readability at different DPI settings; test assistive technologies on supported systems.
- Complete IT/EN coverage for released features; verify signatures after rendering and localization too.

These gates are project criteria, not mathematical security guarantees or achieved metrics.

## Future verification layers

Unit tests for scope, version matching, fingerprints and states. Integration tests for transport, SQLite, crash/recovery, keychain and provider contracts. Fuzz URL parsers, imports, manifests and decoding. Desktop end-to-end tests cover scanning, cancellation, restart, export, retest and data deletion. Use the race detector and dependency analysis for Go code.

Compare with Invicti/Acunetix only with appropriate licensing and comparable configuration, applications, time, credentials and counting rules; respect benchmarking terms. Do not present third-party results as WebFence measurements.
