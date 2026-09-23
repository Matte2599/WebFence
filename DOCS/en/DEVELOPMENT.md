# Development, desktop and localization

[Italiano](../it/DEVELOPMENT.md) · [Index](../README.md)

## Current state and starting point

First offline M0 prototype: native window, on-demand synthetic dataset, filters, virtualized table, evidence detail, explicit copy and persistent language. No crawler, network engine, database, CVE, signing or AI. Read the [M0 report](M0-DESKTOP.md) and [UX direction](UX.md).

Prerequisites: Go **1.27.1**, Fyne **2.8.1** pinned in `go.mod`, C compiler and graphics libraries. macOS requires Xcode/Command Line Tools; Debian derivatives require `gcc libgl1-mesa-dev xorg-dev libxkbcommon-dev libwayland-dev`; Windows requires 64-bit GCC/MinGW-w64 on PATH. Go dependencies are downloaded on the first toolchain run; the application does not scan or download anything.

```sh
git clone https://github.com/Matte2599/WebFence.git
cd WebFence
go mod download
go run ./cmd/webfence
```

To build on macOS/Linux and run automated checks:

```sh
go build -trimpath -o bin/ ./cmd/webfence
go mod verify
go vet -tags ci ./...
go test -tags ci -race ./...
git diff --check
```

On Windows the same build command produces `bin/webfence.exe`; run it from PowerShell or Explorer. The `ci` tag uses the software graphics driver in tests: it **does not test the native window or a screen reader**. Local verification and CI results are separated in the report.

## Implemented structure

```text
cmd/webfence/           desktop entry point
internal/demo/         pure synthetic fixtures, no I/O
internal/i18n/         embedded IT/EN JSON catalogs
internal/ui/           Fyne workspace and interaction tests
scripts/package-macos.sh
.github/workflows/ci.yml
DOCS/                  bilingual documentation
```

The `demo` package is not an analysis engine and its records are not findings. Future domain/network/storage packages will be introduced with testable use cases; do not create empty placeholders. The core will remain independent of Fyne.

## Systems and packaging

Confirmed requirement: **Apple Silicon macOS only; Windows 10 and 11 x86-64; Debian and derivatives on x86-64 and ARM64**. No 32-bit architectures. Minimum macOS/Debian versions remain subject to testing. CI builds on ARM64 macOS, x64 Windows Server and x64/ARM64 Ubuntu; it does not establish interactive compatibility with Windows 10/11 or every Debian distribution.

On Apple Silicon macOS:

```sh
sh scripts/package-macos.sh
open dist/WebFence.app
```

The local `0.0.1` bundle is only for M0 evaluation; it is not a release. The script validates the plist, does not install certificates, sign with a distribution identity or notarize. Any signature inserted by the toolchain is not Developer ID. `dist/` and `bin/` are excluded from Git. Simple cross-compilation with CGO disabled is not promised.

To reproduce the Fyne accessibility experiment on macOS, first close the prototype and rebuild:

```sh
FYNE_BUILD_TAGS=accessibility sh scripts/package-macos.sh
```

The experimental bridge **has not passed the M0 gate**. Normal builds do not enable it; the final toolkit selection remains open. To restore a normal build, close the app and rerun the script without that variable. See [observed limitations](M0-DESKTOP.md).

The only persistent application preference is `ui.language`, managed by Fyne in the per-user app directory for `io.github.Matte2599.WebFence`, not the repository. Examples, filters and selection are not saved. “Clear examples” removes in-memory records, not content already copied to the clipboard. Keychain, database and their deletion flows remain unimplemented.

## IT/EN

String catalogs with stable keys and complete Italian/English coverage. Use the system language initially if supported, otherwise English; provide a persistent desktop selector. Report language is separately configurable. Do not concatenate translated sentences; format plurals, numbers and dates in presentation. Persistence uses UTC; status codes and IDs stay canonical.

Rule text includes title, impact, explanation and remediation in both languages. Original evidence is neither translated nor altered; translations are separate annotations. English fallback prevents blocking but missing translations in released features are defects.

Every PR changing requirements or behavior updates matching IT/EN documents. Language must not change fingerprints, severity, counts, scope or decisions. A translation of a signed report is a new artifact requiring signing.

## Release and operations

Before release: relevant risk tests, dependency/secret scans, SBOM, license inventory, checksums, package signing, IT/EN notes and rollback procedure. Create a verified backup before data migration; binary rollback does not automatically reverse schema changes. Rule/model updates are separately versioned and do not alter running scans.

Keep redacted local logs with run IDs, duration, limits and errors; no default telemetry. On insufficient disk, stale feeds, locked keys, missing runtime or OOM, identify the affected component and remaining capabilities. Do not silently turn degraded operation into success.

M0 update: [practical Qt/Fyne comparison](GUI-COMPARISON.md) and [proposed ADR-002](ADR-002-GUI.md). The Qt experiment is separate; the author’s choice and adoption gates remain open.
