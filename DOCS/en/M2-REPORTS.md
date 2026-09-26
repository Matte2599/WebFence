# M2 — Verifiable reports

[Italiano](../it/M2-REPORTS.md) · [Roadmap](../ROADMAP.md) · [Contract](REPORTING.md)

Status: **third M2 core block**. `internal/reporting` exports a finished SQLite run to a new `.wfr` ZIP bundle with JSON and static HTML in Italian and English. The report uses the authorization revision associated with the run, the redacted M1 ledger and coverage counts; it does not reconstruct URLs, headers, bodies or checks that were not recorded. Unexplored areas remain explicitly unknown. CVE assessments and cache status are included only when supplied as an explicit snapshot: the run-only CLI export marks them unavailable.

The `webfence-manifest-v1` manifest lists the four artifacts with sizes and SHA-256 values, is canonicalized under JCS and signed with the existing restricted Ed25519 JWS profile. `manifest.json` bytes equal the signed payload in `manifest.jws`. The bundle contains no private key; export does not overwrite an existing file. An explicitly requested unsigned export is labeled `unsigned` in both reports and is rejected by the verifier as signed evidence.

## Local technical use

Build `go build -o bin/webfence-report ./cmd/webfence-report` and `go build -o bin/webfence-verify ./cmd/webfence-verify`. Paths must be absolute. The following commands show syntax, not an executed scan.

```text
webfence-report keygen -trust /path/trust.json -name "Operator"
webfence-report export -db /path/projects.sqlite -run RUN_ID -out /path/report.wfr -trust /path/trust.json -kid KEY_ID
webfence-report public -trust /path/trust.json -kid KEY_ID -out /path/public-key.json
webfence-report trust-import -trust /other/trust.json -public /path/public-key.json -fingerprint SHA256_CONFIRMED_INDEPENDENTLY
webfence-verify -bundle /path/report.wfr -trust /other/trust.json
```

`rotate -trust … -kid …` stops signing with the old key while preserving its public key for historical verification; `revoke -trust … -kid …` makes local verification of its reports fail. `export -unsigned` is an explicit choice, for example when the keychain is unavailable; signed mode fails closed. The private key remains in the native credential store, outside the public trust file. Importing a public descriptor requires a SHA-256 fingerprint obtained through an independent channel: receipt of the file alone does not establish trust.

## Verification and limits

`webfence-verify` runs offline with a trust registry outside the bundle. It checks the JWS profile, canonical manifest, all expected files and their hashes; it rejects duplicates, unexpected paths, additions, modifications and unknown or revoked keys. It returns the identity and status in the local registry, including its last update time. It does not consult a revocation service: current trust status and clock accuracy are not guaranteed. A signature does not certify scan correctness, authorization or target security.

Synthetic tests cover bilingual export, HTML escaping, historical authorization revision, unsigned bundles, artifact/manifest/JWS tampering, extra files, wrong/untrusted/revoked keys, rotation and an unavailable keychain. No client reports were exported and no external targets contacted. The [M2 desktop](M2-VALIDATION.md) adds export and verification from the selected run; the CLI alpha is not a supported distribution package.
