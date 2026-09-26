# M2 — Conservative CVE matching

[Italiano](../it/M2-MATCHING.md) · [Roadmap](../ROADMAP.md) · [Cache](M2-INTELLIGENCE-CACHE.md) · [Simulation](M2-ACCURACY-SIMULATION.md)

Status: **second M2 core block**. `internal/intelligence.Assess` correlates one cached NVD or CVE record with **one explicit product/version signal**. It does not infer installed products from M1 scan headers, prove exploitability or automatically create CVE findings in the GUI.

## States and provenance

An assessment preserves the CVE ID, source (`nvd` or `cve`), record SHA-256, signal method and opaque reference, qualitative confidence and a language-independent reason code. `candidate` means the version matches with a low- or medium-confidence signal; `applicable` requires high-confidence inventory or manual identification. `verified` additionally requires an explicit manual or safe-check attestation with a reference; M2 runs no automatic CVE-specific check. `not_applicable` needs high-confidence product/version evidence and an explicitly unaffected condition. `unknown` remains the result for rejected/withdrawn sources, unidentified products, incomparable versions, complex configurations or incomplete data.

A backport can change `applicable` to `not_applicable` **only** with an explicit operator attestation, reference and HTTPS advisory. The engine never opens the URL, and the attestation is not independent vendor verification. An unconfirmed backport or one applied to a weak signal yields `unknown`. CVE and NVD remain separate assessments; disagreements are not hidden.

## Supported subset

- NVD: direct OR configurations with unescaped CPE 2.3, explicit vendor/product identity and CPE part (`a`, `o`, `h`), matching or `*` extra fields, and inclusive/exclusive ranges of dotted numeric versions. The part may come from a full CPE or a separate signal field; without it the outcome is `unknown`. AND, negation, recursive children, escaped fields and unproven platform conditions yield `unknown`.
- CVE JSON: `containers.cna.affected`, explicit vendor/product, exact versions and `versionType=semver` ranges with three numeric components, including evaluation of sorted `changes`. Conflicting overlapping ranges, out-of-range status changes, prereleases, distro, RPM, Python, Maven and custom ranges, or unproven platform, module, package or CPE constraints yield `unknown`; versions are never compared lexicographically.
- A banner remains low confidence even if its input claims high confidence; it cannot produce `verified` or a negative conclusion. Duplicate JSON members, mismatched IDs and inconsistent hashes are rejected.

## Verification and limits

The [local simulation](M2-ACCURACY-SIMULATION.md) publishes the method, matrix and limits of 30 synthetic cases. `go test -race ./internal/intelligence` contacts no external source or target. No real-corpus matching accuracy or support for every CPE/CVE version scheme is claimed. The [M2 desktop](M2-VALIDATION.md) accepts manual vendor/product/version signals for a selected run; until the CPE part field is added, an NVD signal without a full CPE remains `unknown`. Backport and verification attestations remain core APIs without a GUI form.

Primary sources: [CVE Record Format](https://cveproject.github.io/cve-schema/schema/docs/), [NVD CPE FAQ](https://nvd.nist.gov/general/faq-sections/cpe-faqs), [NVD API](https://nvd.nist.gov/developers/vulnerabilities).
