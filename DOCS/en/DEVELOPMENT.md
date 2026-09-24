# Qt desktop development

[Italiano](../it/DEVELOPMENT.md) · [Index](../README.md)

## Running and checking

Qt Widgets/MIQT is the author-selected main GUI. Go **1.27.1** and MIQT **0.14.0** are pinned in the module. The desktop remains an offline example prototype: no scanning or persistent projects **in the GUI**, CVE or AI. The core has an [M1 project store](M1-PROJECT-STORE.md) and a [first loopback-only HTTP check](M1-HEADER-LAB.md), separate from the GUI. [Desktop status and verification](QT-DESKTOP.md).

The first binding compilation can take several minutes; later builds benefit from Go’s cache. macOS CI compiles C++ wrappers with `-O0 -g0` to limit first-build cost; build and bundle share flags and cache. Installed Qt libraries remain the Homebrew package binaries. The local script defaults to `-O2 -g`; CI checks functionality, not release performance.

Requires a C++17 compiler, CGO, pkg-config and Qt 6 Core/Gui/Widgets. On Apple Silicon macOS install Xcode/Command Line Tools and `brew install qtbase pkgconf`; on Debian/Ubuntu use `sudo apt-get install g++ pkg-config qt6-base-dev`.

```sh
go mod download
export CGO_CXXFLAGS='-O2 -g -std=c++17'
go run ./cmd/webfence
```

```sh
go mod verify
go test -race ./internal/demo ./internal/i18n ./internal/preferences ./internal/scope ./internal/project ./internal/storage ./internal/transport ./internal/scanner ./internal/foundation ./internal/signature ./internal/credentials
go vet ./...
go build -o bin/webfence ./cmd/webfence
QT_QPA_PLATFORM=offscreen ./bin/webfence --self-test
git diff --check
```

The self-test uses real Qt without a display and a temporary directory for language; copying is intercepted to preserve the clipboard. It also checks save failures, long text and language actions. It is not a screen-reader test. Fyne's `ci` tag is no longer needed. `go test ./...` now needs Qt dependencies to compile the desktop; pure packages can be checked separately as above.

On Windows x86-64 use MSYS2 **UCRT64**, Go on PATH and matching tools: `mingw-w64-ucrt-x86_64-gcc`, `mingw-w64-ucrt-x86_64-pkgconf`, `mingw-w64-ucrt-x86_64-qt6-base`. In UCRT64 use the same variables and build with `go build -ldflags "-H=windowsgui -s -w" -o bin/webfence.exe ./cmd/webfence`; run `./bin/webfence.exe --self-test` with `QT_QPA_PLATFORM=offscreen`. Qt DLLs and plugins must be available; the exe alone is not a distributable package. CI uses [setup-msys2](https://github.com/msys2/setup-msys2); MSVC is not the CGO compiler in this setup.

## Structure and data

- `cmd/webfence`: desktop entry and `--self-test` option.
- `internal/desktop`: Qt workspace, model value ownership and self-test.
- `internal/demo`: pure fixtures without networking.
- `internal/i18n`: embedded IT/EN JSON catalogs.
- `internal/preferences`: language only, independent of toolkit.
- `internal/project` and `internal/storage`: authorization model, SQLite metadata-only store and [managed runs with local revocation](M1-MANAGED-RUNS.md), still disconnected from the GUI.
- `internal/scanner`: [first laboratory HTTP check](M1-HEADER-LAB.md) on explicit seeds, with no persistent data or external networking.

Future engine packages remain Qt-independent. Do not use the fixture cache as a retention design for real data. Fyne and the old Qt laboratory remain in Git history, not in the current build.

In the desktop, the only saved preference is `WebFence/ui-language` under `os.UserConfigDir()` (macOS: `~/Library/Application Support`; Linux: `$XDG_CONFIG_HOME` or `~/.config`; Windows: `%AppData%`). Writes use a temporary file and rename in the same directory. Errors are visible in the GUI; session language remains usable. Old Fyne preferences are neither imported nor deleted. Examples, filters and selection are not saved. Explicitly copied clipboard contents are not erased by “Clear examples”.

The Language menu and selector switch IT/EN; shortcuts are **Ctrl+1/Ctrl+2** (**Cmd+1/Cmd+2** on macOS). **Ctrl+O/Cmd+O** loads examples. Full keyboard and screen-reader validation remains open.

The View menu adds search (Ctrl/Cmd+F), results (F6), evidence (Ctrl/Cmd+Shift+E) and advanced details (Ctrl/Cmd+Shift+D). See [keyboard, focus and assistive limitations](QT-DESKTOP.md).

## Platforms and bundle

Requirements: Apple Silicon macOS, Windows 10/11 x86-64, Debian/derivatives x86-64 and ARM64; no 32-bit. The author chose macOS 26 as the temporary minimum; the local bundle declares 26.0 but has not yet been tested on a real 26.0 system. Windows/Debian minima and Windows 10 maintenance remain to be defined/verified under [ADR-002](ADR-002-GUI.md). CI builds do not establish assistive use on the required systems.

```sh
sh scripts/package-macos.sh
open dist/WebFence.app
```

The script produces the local `0.0.1` bundle, validates the plist and uses `macdeployqt` for libraries/plugins. Requires Homebrew Qt and Apple Silicon. No Developer ID signature or notarization; clean-machine testing remains open. `bin/` and `dist/` are Git-ignored. WebFence's license remains unchanged; Qt/MIQT have separate rights and obligations to verify before distribution.

In the private staging bundle the script removes external `LC_RPATH` entries, then checks that explicit Mach-O dependencies resolve in the bundle or Apple system libraries before signing. Recheck the published artifact with `python3 scripts/check-macos-linkage.py dist/WebFence.app`; [scope and evidence](../evidence/macos-linkage-closure-2026-09-24.md).

## IT/EN

String catalogs with stable keys and complete Italian/English coverage. Use the system language initially if supported, otherwise English; provide a persistent desktop selector. Report language is separately configurable. Do not concatenate translated sentences; format plurals, numbers and dates in presentation. Persistence uses UTC; status codes and IDs stay canonical.

Rule text includes title, impact, explanation and remediation in both languages. Original evidence is neither translated nor altered; translations are separate annotations. English fallback prevents blocking but missing translations in released features are defects.

Every PR changing requirements or behavior updates matching IT/EN documents. Language must not change fingerprints, severity, counts, scope or decisions. A translation of a signed report is a new artifact requiring signing.

## Release and operations

Before release: relevant risk tests, dependency/secret scans, SBOM, license inventory, checksums, package signing, IT/EN notes and rollback procedure. Create a verified backup before data migration; binary rollback does not automatically reverse schema changes. Rule/model updates are separately versioned and do not alter running scans.

Keep redacted local logs with run IDs, duration, limits and errors; no default telemetry. On insufficient disk, stale feeds, locked keys, missing runtime or OOM, identify the affected component and remaining capabilities. Do not silently turn degraded operation into success.

## First scope layer (M0)

`internal/scope` contains the immutable origin policy; the HTTP lab exists only in `lab_test.go`. [Contract and limitations](M0-SCOPE.md). The desktop performs no requests.

```sh
go test -race -cover ./internal/scope
go test ./internal/scope -run '^$' -fuzz '^FuzzCheck$' -fuzztime=20s -parallel=4
```

## M0 lab transport

`internal/transport.NewLab` uses explicit loopback origins/IPs, a controlled resolver and mandatory limits. It is not connected to the desktop. [Decision, contract and limitations](ADR-003-TRANSPORT.md).

```sh
go test -race -cover ./internal/transport
```

## SQLite and JWS: M0 foundations

Driver and library selected in [ADR-004](ADR-004-STORAGE-SIGNATURE.md). `internal/foundation` contains SQLite tests on temporary files only; `internal/signature` exposes byte signing/verification, without reports, JCS or keychain access. Neither is connected to the GUI.

```sh
go test -race ./internal/foundation ./internal/signature
go test ./internal/signature -run '^$' -fuzz '^FuzzVerify$' -fuzztime=20s -parallel=4
```

## Credentials: M0 tests

[ADR-005](ADR-005-CREDENTIALS.md) documents backends, limits and checks. Ordinary unit tests do not write personal credentials. For native macOS tests (separate temporary keychain) and Windows tests (synthetic item with random namespace):

```sh
go test -race -tags=keychainintegration ./internal/credentials -count=1 -v
```

On Linux use only the isolated launcher, with `dbus-run-session`, `gnome-keyring-daemon`, `gdbus` and `rg` installed. Do not manually set the isolation flag on a personal bus: the test locks the test service collection.

```sh
sh scripts/test-keychain-linux.sh
```

Optional Windows source collection also requires GnuPG on PATH: it verifies the eight signatures of locked versions offline using only the public keys in the repository. [Procedure and limits](M0-WINDOWS-SOURCES.md#offline-verification-of-eight-detached-signatures). The ordinary GUI build does not require GnuPG.

The macOS bundle now rebuilds the Qt 6.11.2 Cocoa plugin with a temporary assistive-crash correction: [ADR-006](ADR-006-QT-COCOA.md). CMake, Ninja and MoltenVK/Vulkan headers are also required (`brew install cmake ninja molten-vk vulkan-headers`); Qt sources are downloaded and hash-verified. Other Qt versions are rejected until reassessed. Unpackaged binaries continue using installed Qt.

Added `.deb`/Windows ZIP packaging and separate-runtime checks: [ADR-007 and commands](ADR-007-PACKAGING.md). [CI `32655b0` passed, six jobs](https://github.com/Matte2599/WebFence/actions/runs/35877356234); development packages, not supported releases.

macOS packaging also requires Python 3.9+ to collect provenance/notices before signing: [materials and known gaps](M0-PACKAGING.md). The check rejects unmatched or ambiguous native libraries.

Optional source-archive and notice collection from a macOS inventory: [procedure, requirements and tests](M0-SOURCE-MATERIALS.md). Not automatically run by the product build.

Set `WEBFENCE_NATIVE_SOURCE_MATERIALS` to a completed collection directory to attach reverified source notices before macOS signing; see the same procedure. Archives remain separate.

Offline documentation included in all packages through a shared collector, with linked-file verification: [procedure and limits](ADR-007-PACKAGING.md#offline-documentation-in-packages).
