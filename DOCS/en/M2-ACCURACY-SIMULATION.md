# M2 — Local CVE accuracy simulation

[Italiano](../it/M2-ACCURACY-SIMULATION.md) · [Matching](M2-MATCHING.md) · [M2 validation](M2-VALIDATION.md)

## Method

`TestSyntheticAccuracySimulation` uses **30 labeled synthetic cases** without network access, a CVE database or external systems. The labels distinguish 20 cases with a known product/version assumption (`affected` or `unaffected`) from 10 indeterminate cases that require abstention. Cases cover NVD/CVE, inclusive and exclusive boundaries, exact versions, `changes`, weak signals, incomparable versions, NVD AND, platforms, modules, package identity, CPE qualifiers and overlapping ranges. A separate test checks that the CPE part (`a`, `o`, `h`) is declared for NVD when a full CPE is absent.

A positive conclusion is `applicable`/`verified`; a negative one is `not_applicable`; `candidate` and `unknown` are abstentions. The corpus has explicit expected outcomes and fails if an assessment changes. Reproduce with `go test ./internal/intelligence -run 'TestSyntheticAccuracySimulation|TestNVDProductPartIsExplicit' -v`.

| Corpus measure | Result |
| --- | ---: |
| Total cases | 30 |
| Cases with affected/unaffected ground truth | 20 |
| Correct positive / negative conclusions | 8 / 9 |
| Conclusive false positives / false negatives | 0 / 0 |
| Abstentions on known-truth cases | 3 |
| Indeterminate cases correctly left `unknown` | 10 |
| Conclusive coverage on known-truth cases | 17/20 (85%) |

## Review-driven fixes

- NVD requires an explicit CPE part when the signal has only vendor/product: a text identity does not show whether the product is an application, operating system or hardware.
- CVE entries with unproven platform, module, package or CPE constraints remain `unknown`.
- All overlapping CVE version entries are evaluated; conflicting outcomes remain `unknown`. Out-of-range status changes and unrecognized change statuses are rejected.

This is a **regression simulation over constructed cases**, not a statistical estimate of accuracy on real CVEs. It does not measure feed quality, product identity in the field, prevalence, recall for real vulnerabilities or the validity of operator-attested backports. The analysis follows the published [CVE Record Format](https://cveproject.github.io/cve-schema/schema/docs/) and [NVD configuration documentation](https://nvd.nist.gov/vuln/Vulnerability-Detail-Pages).
