# Reports, evidence and signing

[Italiano](../it/REPORTING.md) · [Index](../README.md)

Status: [M2 reports](M2-REPORTS.md) implement bilingual JSON/HTML export, JCS, JWS, native key storage and offline verification for M1 runs; the [M2 desktop](M2-VALIDATION.md) exposes a limited guided workflow. The sections below also describe the future contract; richer findings, PDF and linked corrections remain planned.

## Content

A report includes project and environment, authorized targets, excluded scope, UTC interval, engine/rule version, profile and budgets, CVE snapshot, AI model where applicable, scan status and coverage. Skipped, limited, failed and blocked checks must be visible alongside findings.

Each finding includes a stable ID, IT/EN title, category, applicable CWE/CVE, severity separate from confidence, verification state, asset/route, redacted identity context, prerequisites, evidence, minimal reproduction steps, impact and remediation. CVSS requires version, vector and source; unknown scores remain unknown. Operational priority and technical severity are separate fields.

Minimize evidence before export: exclude secrets, cookies and unnecessary personal content. Keep hashes and an omission description; the hash identifies what was included, not the removed content. Do not render captured pages as active HTML in the desktop or exports.

## Formats and sequence

Structured JSON and static HTML in M2; PDF in M6 after rendering QA. Selected language changes text, not IDs or outcomes. A bilingual bundle lists IT and EN artifacts separately. The PDF is not automatically a PAdES document or qualified signature: legal signature requirements need separate design.

1. Freeze a results snapshot; generate final artifacts.
2. Create a manifest with schema version, report/run/project IDs, signer-declared UTC time, language, declared identity, key ID and files listed by relative path, size and SHA-256.
3. Canonicalize the manifest with [JCS, RFC 8785](https://www.rfc-editor.org/rfc/rfc8785). Reject duplicate keys, invalid numbers and invalid strings; use strings for long numeric identifiers.
4. Sign the canonical bytes as an embedded payload in a [JWS, RFC 7515](https://www.rfc-editor.org/rfc/rfc7515.html). The implemented profile uses `Ed25519` from [RFC 9864](https://www.rfc-editor.org/rfc/rfc9864.html), with algorithm, `kid` and document type in the protected header, through `jwx/v4` already selected in [ADR-004](ADR-004-STORAGE-SIGNATURE.md).
5. Distribute artifacts and JWS; any readable manifest copy must match the verified payload exactly. The signature is excluded from the manifest to avoid self-reference.

## Verification and trust

The offline verifier must accept only the expected profile and a public key trusted outside the bundle. `kid` is a selector, not identity proof. Reject `none`, unexpected algorithms, keys fetched from report URLs, unknown critical headers, absolute paths, traversal, duplicates and missing or modified files. An unlisted added file is not signed and must be reported as such.

Verify JWS, schema, manifest correspondence and file hashes before showing “valid signature and trusted signer”. Distinguish invalid signature, unknown key, revoked key and outdated revocation status. A public key included in the bundle may assist transport but cannot establish trust.

## Key management and limitations

Generate keys per installation/operator; keep private keys in a keychain or encrypted container, never a universal embedded key. The configured operator signs, not automatically Matteo Luigi Feroldi. Support secure export, rotation, revocation and retention of old public keys. If the key is unavailable, produce only an explicitly unsigned export.

A valid signature detects alteration and identifies a trusted key; it does not prove every observation correct, the site secure or the local clock trustworthy. Trusted timestamps and current revocation status are not guaranteed offline. Corrections, later translations and retests produce new signed reports referencing their predecessor without modifying it.
