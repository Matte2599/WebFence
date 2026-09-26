# M2 — Advanced desktop features

[Italiano](../it/M2-GUI-EXTENSION.md) · [M2 validation](M2-VALIDATION.md) · [Matching](M2-MATCHING.md) · [Reports](M2-REPORTS.md)

The selected run's **CVE and reports** dialog has three tabs: **CVE sources**, **Assessment**, **Reports and keys**. Cache updates remain manual and attestations are operator statements, not automatic CVE checks.

## Assessment

- A signal has vendor, product, version, method, confidence and a non-secret reference. NVD without a full CPE also requires the `application`/`operating system`/`hardware` part. An optional full CPE 2.3 replaces the separate identity fields; its version must equal the supplied version.
- The backport form requires explicit confirmation, an HTTPS advisory and a valid reference. The verification form requires confirmation, a manual method or a safe check **already performed**, and a valid reference. Both attestations cannot be confirmed together. The app does not open the advisory.
- Changes to CVE, source, run, identity/version or signal quality clear the form attestations. This prevents a confirmation from following a different record. Existing assessments for the selected run remain in dialog memory only until session end or run change.
- A verification attempt with invalid data remains `unknown` in the core instead of silently appearing applicable. The GUI shows IT/EN state and reason descriptions while report codes remain stable.

## Reports and keys

The GUI creates private keys in the native credential store, exports signed or explicitly unsigned reports and verifies bundles offline. It can also **rotate** an active key, **revoke** a key, export a public JSON descriptor and import one with a SHA-256 fingerprint obtained through an independent channel. Rotation and revocation request confirmation. An imported public key can verify reports but cannot sign without the matching local private key. The trust registry is not an online, current revocation service.

## Extra local verification

The Qt self-test uses one run and two synthetic CVE/NVD records over loopback. It exercises both NVD identity modes, valid and conflicting attestations, stale-form reset, unsigned and signed bilingual export, offline verification, generation/rotation/revocation with an isolated synthetic credential store, import with wrong and correct fingerprints and a short key ID, and preserving the selected signing key across language changes. Core `go test -race`, `go test ./...`, `go vet ./...`, Python tests and soak remain general regressions.

Rendering was inspected on native macOS Cocoa and Qt offscreen in IT/EN: three tabs, the top/bottom of scrollable forms, and normal **820×740** and compact **620×560** logical-point windows. Local synthetic screenshots were inspected for overlaps, unreachable controls and clipped text; form growth and row wrapping resolved the initially narrow Cocoa fields. This check does not certify freedom from defects on Windows/Linux, every DPI scale, theme or assistive technology; CI checks native flows on its configured platforms.

In one local Cocoa repeat, pre-existing M0 focus/keyboard assertions were intermittent while the macOS foreground changed; a repeat without contention passed. The M2 flow stayed green. This fluctuation is not evidence of reliable focus in real use.
