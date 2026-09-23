# Qt desktop development

[Italiano](../it/DEVELOPMENT.md) · [Index](../README.md)

## Running and checking

Qt Widgets/MIQT is the author-selected main GUI. Go **1.27.1** and MIQT **0.14.0** are pinned in the module. This remains an offline example prototype: no scanner, persistent projects, CVE or AI. [Status and verification](QT-DESKTOP.md).

Requires a C++17 compiler, CGO, pkg-config and Qt 6 Core/Gui/Widgets. On Apple Silicon macOS install Xcode/Command Line Tools and `brew install qtbase pkgconf`; on Debian/Ubuntu use `sudo apt-get install g++ pkg-config qt6-base-dev`.

```sh
go mod download
export CGO_CXXFLAGS='-O2 -g -std=c++17'
go run ./cmd/webfence
```

```sh
go mod verify
go test -race ./internal/demo ./internal/i18n ./internal/preferences
go vet ./...
go build -o bin/webfence ./cmd/webfence
QT_QPA_PLATFORM=offscreen ./bin/webfence --self-test
git diff --check
```

The self-test uses real Qt without a display and a temporary directory for language; copying is intercepted to preserve the clipboard. It also checks save failures, long text and language actions. It is not a screen-reader test. Fyne's `ci` tag is no longer needed. `go test ./...` now needs Qt dependencies to compile the desktop; pure packages can be checked separately as above.

On Windows x86-64 use MSYS2 **UCRT64**, Go on PATH and matching tools: `mingw-w64-ucrt-x86_64-gcc`, `mingw-w64-ucrt-x86_64-pkgconf`, `mingw-w64-ucrt-x86_64-qt6-base`. In UCRT64 use the same variables and build with `go build -o bin/webfence.exe ./cmd/webfence`; run `./bin/webfence.exe --self-test` with `QT_QPA_PLATFORM=offscreen`. Qt DLLs and plugins must be available; the exe alone is not a distributable package. CI uses [setup-msys2](https://github.com/msys2/setup-msys2); MSVC is not the CGO compiler in this setup.

## Structure and data

- `cmd/webfence`: desktop entry and `--self-test` option.
- `internal/desktop`: Qt workspace, model value ownership and self-test.
- `internal/demo`: pure fixtures without networking.
- `internal/i18n`: embedded IT/EN JSON catalogs.
- `internal/preferences`: language only, independent of toolkit.

Future engine packages remain Qt-independent. Do not use the fixture cache as a retention design for real data. Fyne and the old Qt laboratory remain in Git history, not in the current build.

The only saved preference is `WebFence/ui-language` under `os.UserConfigDir()` (macOS: `~/Library/Application Support`; Linux: `$XDG_CONFIG_HOME` or `~/.config`; Windows: `%AppData%`). Writes use a temporary file and rename in the same directory. Errors are visible in the GUI; session language remains usable. Old Fyne preferences are neither imported nor deleted. Examples, filters and selection are not saved. Explicitly copied clipboard contents are not erased by “Clear examples”.

The Language menu and selector switch IT/EN; shortcuts are **Ctrl+1/Ctrl+2** (**Cmd+1/Cmd+2** on macOS). **Ctrl+O/Cmd+O** loads examples. Full keyboard and screen-reader validation remains open.

## Platforms and bundle

Requirements: Apple Silicon macOS, Windows 10/11 x86-64, Debian/derivatives x86-64 and ARM64; no 32-bit. Minimum versions and Windows 10 maintenance remain subject to [ADR-002](ADR-002-GUI.md). CI builds do not establish assistive use on the required systems.

```sh
sh scripts/package-macos.sh
open dist/WebFence.app
```

The script produces the local `0.0.1` bundle, validates the plist and uses `macdeployqt` for libraries/plugins. Requires Homebrew Qt and Apple Silicon. No Developer ID signature or notarization; clean-machine testing remains open. `bin/` and `dist/` are Git-ignored. WebFence's license remains unchanged; Qt/MIQT have separate rights and obligations to verify before distribution.

## IT/EN

String catalogs with stable keys and complete Italian/English coverage. Use the system language initially if supported, otherwise English; provide a persistent desktop selector. Report language is separately configurable. Do not concatenate translated sentences; format plurals, numbers and dates in presentation. Persistence uses UTC; status codes and IDs stay canonical.

Rule text includes title, impact, explanation and remediation in both languages. Original evidence is neither translated nor altered; translations are separate annotations. English fallback prevents blocking but missing translations in released features are defects.

Every PR changing requirements or behavior updates matching IT/EN documents. Language must not change fingerprints, severity, counts, scope or decisions. A translation of a signed report is a new artifact requiring signing.

## Release and operations

Before release: relevant risk tests, dependency/secret scans, SBOM, license inventory, checksums, package signing, IT/EN notes and rollback procedure. Create a verified backup before data migration; binary rollback does not automatically reverse schema changes. Rule/model updates are separately versioned and do not alter running scans.

Keep redacted local logs with run IDs, duration, limits and errors; no default telemetry. On insufficient disk, stale feeds, locked keys, missing runtime or OOM, identify the affected component and remaining capabilities. Do not silently turn degraded operation into success.
