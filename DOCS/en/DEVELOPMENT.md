# Development, desktop and localization

[Italiano](../it/DEVELOPMENT.md) · [Index](../README.md)

## Current state and starting point

The repository contains documentation and editor/Git configuration only. There is no `go.mod`, build, installer, server, CLI or engine test suite. To obtain the documents:

```sh
git clone https://github.com/Matte2599/WebFence.git
cd WebFence
```

Read the [README](../../README.en.md), [roadmap](../ROADMAP.md) and [CONTRIBUTING](../../CONTRIBUTING.md). No `webfence scan` command is available: do not add executable instructions until the path is implemented and verified.

## Proposed future structure

```text
cmd/webfence/          desktop application
cmd/webfence-cli/      possible later CLI
internal/app/         use cases
internal/ui/          GUI and localization
internal/policy/      authorization and scope
internal/scan/        scheduler, discovery and checks
internal/intelligence/
internal/ai/
internal/report/
internal/storage/
testdata/             synthetic fixtures
DOCS/                 bilingual specifications
```

Create packages when a useful first function exists, not as empty containers. The core does not import the GUI toolkit; checks do not directly access arbitrary networking or filesystems.

## Systems and installation

Targets: macOS, Windows and Linux. The support matrix depends on real OS/architecture tests; the design machine does not establish compatibility elsewhere. Go/Fyne requires an appropriate graphics/native toolchain: see [Fyne documentation](https://docs.fyne.io/started/). Do not assume `CGO_ENABLED=0` or simple cross-compilation for the entire desktop.

M0 must test macOS bundles, Windows packaging and the selected Linux format, graphics dependencies, keychain and accessibility. Application signing/notarization and report signing are separate systems. No distribution certificates are present. Go, GUI, browser and library versions will be pinned after the prototype, with lockfiles and reproducible CI.

The app uses per-user system data directories, not the source directory; exports go to operator-selected destinations. No routine administrator privileges. Models, browsers and large snapshots are optional verified packages; disclose downloads and disk requirements before installation. Uninstallation does not silently delete data: provide an explicit choice.

## IT/EN

String catalogs with stable keys and complete Italian/English coverage. Use the system language initially if supported, otherwise English; provide a persistent desktop selector. Report language is separately configurable. Do not concatenate translated sentences; format plurals, numbers and dates in presentation. Persistence uses UTC; status codes and IDs stay canonical.

Rule text includes title, impact, explanation and remediation in both languages. Original evidence is neither translated nor altered; translations are separate annotations. English fallback prevents blocking but missing translations in released features are defects.

Every PR changing requirements or behavior updates matching IT/EN documents. Language must not change fingerprints, severity, counts, scope or decisions. A translation of a signed report is a new artifact requiring signing.

## Release and operations

Before release: relevant risk tests, dependency/secret scans, SBOM, license inventory, checksums, package signing, IT/EN notes and rollback procedure. Create a verified backup before data migration; binary rollback does not automatically reverse schema changes. Rule/model updates are separately versioned and do not alter running scans.

Keep redacted local logs with run IDs, duration, limits and errors; no default telemetry. On insufficient disk, stale feeds, locked keys, missing runtime or OOM, identify the affected component and remaining capabilities. Do not silently turn degraded operation into success.
