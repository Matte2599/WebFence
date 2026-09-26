# M2 — Intelligence and report alpha validation

[Italiano](../it/M2-VALIDATION.md) · [Roadmap](../ROADMAP.md) · [Development](DEVELOPMENT.md)

Technical status: all four M2 criteria are implemented in the core and Qt desktop. Closure also requires green CI on the branch and the integrated `main` commit; read outcomes from the GitHub runs for the relevant commit rather than inferring them from local tests. The trials below are synthetic and do not establish real-corpus matching quality, support on every minimum OS version, certified accessibility or distribution readiness.

| Criterion | Technical evidence and limit |
| --- | --- |
| CVE/NVD adapters and cache | `internal/intelligence` syncs explicit NVD windows and CVE IDs to a separate SQLite cache with provenance, hashes, watermark and transactional rollback. The GUI opens source network connections only after an explicit action, to the official endpoints. No catalog is bundled or downloaded on startup; local `fresh/stale` uses a 24-hour threshold, not a completeness promise. |
| Conservative matching | `Assess` requires operator-supplied vendor/product/version and references; `candidate`, `applicable`, `verified`, `not_applicable`, `unknown` remain distinct. The GUI accepts inventory/manual/banner signals and confidence, but does not turn an M1 header into a CVE or yet expose backport/verification attestation forms. GUI assessments live in the session for the selected run until export. |
| JSON/HTML, signing and verification | The `.wfr` bundle contains static IT/EN artifacts, a JCS manifest with hashes and an Ed25519 JWS; the private key uses the native credential store. The GUI creates an operator key, exports signed/unsigned and verifies against the local trust registry. The technical CLI also offers rotation, revocation and public descriptor export/import with an independently confirmed fingerprint. Offline verification does not consult current revocation or trusted timestamps. |
| Security regressions | Local fixtures exercise unavailable feeds and interrupted sync without losing the last valid snapshot; ambiguous versions remain `unknown`. Modified bundles, unexpected files, bad signatures and unknown/revoked keys are rejected. The Qt self-test runs a loopback scan, fetches a synthetic CVE from a local server, explicitly matches it and exports a bilingual unsigned bundle. |

The report preserves the run's authorization revision and only the redacted ledger. Whole-site coverage is not attested: it shows the known minimum of unexecuted seeds and leaves other areas unknown. NVD and CVE have separate report states, with offline mode distinct from freshness at export. CLI export without a cache snapshot marks them `unavailable`, never zero CVEs.

## Full M2 test

The local milestone check uses `go mod verify`, `go test -race` on all core packages, `go test ./...`, `go vet ./...`, builds of all three commands, the offscreen Qt self-test, soak test, Python support tests, IT/EN consistency, relative links and `git diff --check`. CI adds builds, self-tests and development packaging on macOS, Windows and Ubuntu ARM64/x86-64. The self-test does not replace VoiceOver, NVDA, Orca or real hardware trials.

**Local trial on September 26, 2026, Apple Silicon/macOS 26.6.2:** the checks above passed; `go test -race ./internal/desktop` with C++17 flags, the Qt self-test with a synthetic CVE assessment and the 10-second soak (four cycles) also passed. The Python suite ran 79 tests with four environment-dependent skips. Branch and final-merge outcomes are separate from this local trial and must be checked in the corresponding CI runs.

No real NVD/CVE sync or scans of external targets were performed to close M2. The source and scanned site are distinct network boundaries: the feed receives no project inventory. CVE accuracy, live endpoint availability, target load feedback, M0/M1 assistive limits and professional legal review remain support/distribution risks, not outcomes implicitly passed by M2.

## Extra review after M2

The [local matching simulation](M2-ACCURACY-SIMULATION.md) adds 30 labeled synthetic cases: 17 correct conclusions among 20 cases with known truth, three abstentions, and ten indeterminate cases left `unknown`. This measure is limited to the constructed corpus; it does not change M2 closure or establish accuracy on real CVEs. The review also required an explicit CPE part for NVD without a full CPE and abstention for unproven CVE qualifiers or conflicting version entries.
