# Qt desktop development

[Italiano](../it/DEVELOPMENT.md) · [Index](../README.md)

## Running and checking

Qt Widgets/MIQT is the author-selected main GUI. Go **1.27.1** and MIQT **0.14.0** are pinned in the module. The desktop offers M0 synthetic examples, an [M1 alpha](M1-VALIDATION.md) with bounded HTTP scans and an [M2 flow](M2-VALIDATION.md) for the CVE cache, explicit assessments and reports. AI is not active. [M0 desktop status and verification](QT-DESKTOP.md).

The first binding compilation can take several minutes; later builds benefit from Go’s cache. macOS CI compiles C++ wrappers with `-O0 -g0` to limit first-build cost; build and bundle share flags and cache. Installed Qt libraries remain the Homebrew package binaries. The local script defaults to `-O2 -g`; CI checks functionality, not release performance.

Requires a C++17 compiler, CGO, pkg-config and Qt 6 Core/Gui/Widgets. On Apple Silicon macOS install Xcode/Command Line Tools and `brew install qtbase pkgconf`; on Debian/Ubuntu use `sudo apt-get install g++ pkg-config qt6-base-dev`.

```sh
go mod download
export CGO_CXXFLAGS='-O2 -g -std=c++17'
go run ./cmd/webfence
```

```sh
go mod verify
go test -race ./internal/demo ./internal/i18n ./internal/preferences ./internal/scope ./internal/project ./internal/storage ./internal/transport ./internal/scanner ./internal/foundation ./internal/signature ./internal/credentials ./internal/intelligence ./internal/reporting
go vet ./...
go build -o bin/webfence ./cmd/webfence
QT_QPA_PLATFORM=offscreen ./bin/webfence --self-test
git diff --check
```

The self-test uses real Qt without a display and a temporary directory for language and DB; copying is intercepted to preserve the clipboard. It also checks an M1 scan, a synthetic CVE fetched from a local `httptest` server, an explicit assessment and unsigned M2 export: **no external target**. It is not a screen-reader test. Fyne's `ci` tag is no longer needed. `go test ./...` needs Qt dependencies to compile the desktop.

On Windows x86-64 use MSYS2 **UCRT64**, Go on PATH and matching tools: `mingw-w64-ucrt-x86_64-gcc`, `mingw-w64-ucrt-x86_64-pkgconf`, `mingw-w64-ucrt-x86_64-qt6-base`. In UCRT64 use the same variables and build with `go build -ldflags "-H=windowsgui -s -w" -o bin/webfence.exe ./cmd/webfence`; run `./bin/webfence.exe --self-test` with `QT_QPA_PLATFORM=offscreen`. Qt DLLs and plugins must be available; the exe alone is not a distributable package. CI uses [setup-msys2](https://github.com/msys2/setup-msys2); MSVC is not the CGO compiler in this setup.

Windows packaging in CI retains the [libwinpthread](https://packages.msys2.org/packages/mingw-w64-ucrt-x86_64-libwinpthread) package already reviewed in the binary lock `packaging/windows/msys2-binary-lock.json`: after the MSYS2 update it restores the pinned version along with the [winpthreads](https://packages.msys2.org/packages/mingw-w64-ucrt-x86_64-winpthreads) package that requires it, verifying both published SHA-256 values. This prevents an upstream release from silently changing ZIP contents; updating the version requires explicit checksum and signature review.

## Structure and data

- `cmd/webfence`: desktop entry and `--self-test` option.
- `internal/desktop`: Qt workspace, model value ownership and self-test.
- `internal/demo`: pure fixtures without networking.
- `internal/i18n`: embedded IT/EN JSON catalogs.
- `internal/preferences`: language only, independent of toolkit.
- `internal/project` and `internal/storage`: authorization model, [SQLite v4](M1-PROJECT-STORE.md) with redacted runs/observations and [local revocation](M1-MANAGED-RUNS.md).
- `internal/scope` and `internal/transport`: [route policy and public IP-pinned broker](M1-CONTROLLED-CRAWL.md), used by the M1 desktop.
- `internal/scanner`: [first HTTP check](M1-HEADER-LAB.md), [HTML discovery](M1-DISCOVERY-LAB.md) and [bounded crawler](M1-CONTROLLED-CRAWL.md) with persistent results. Tests open no external network connections.
- `internal/intelligence` and `internal/reporting`: [M2 cache/matching](M2-INTELLIGENCE-CACHE.md) and [verifiable bundles](M2-REPORTS.md), independent of Qt. `cmd/webfence-report` and `cmd/webfence-verify` are technical helpers buildable from source.

Future engine packages remain Qt-independent. Do not use the fixture cache as a retention design for real data. Fyne and the old Qt laboratory remain in Git history, not in the current build.

In the desktop, `WebFence/ui-language`, `WebFence/projects.sqlite`, `WebFence/intelligence.sqlite` and `WebFence/report-trust.json` are under `os.UserConfigDir()` (macOS: `~/Library/Application Support`; Linux: `$XDG_CONFIG_HOME` or `~/.config`; Windows: `%AppData%`). The project DB may have `.lock`, `-wal` and `-shm` files; the public trust registry may have `.lock`. Private keys stay in the native credential store. Language uses a temporary file and rename; projects, M1 results and the CVE cache persist, while GUI assessments live in the session until export. Errors are visible in the GUI. Old Fyne preferences are neither imported nor deleted. M0 examples, filters and selection are not saved. Explicitly copied clipboard contents are not erased by “Clear examples”.

For an M1 trial, open **M1 Scan**, create a project with exact origin, owner, non-secret authorization reference, expiry and confirmation. Select it, enter an authorized seed, allowed/excluded prefixes, loopback or manually pinned public IP mode, budget, pages and depth. Links are visited only when opted in. Start, review progress and coverage, then reopen the app to read saved results. Use only owned or explicitly authorized targets; the self-test uses loopback only. The GUI does not yet provide renewal/revocation or backup; see the [APIs and limits](M1-PROJECT-STORE.md).

For M2, select a finished run in the M1 window and open **CVE and reports**. Choose a UTC NVD window or CVE ID and update the cache only by explicit action; enter a non-secret vendor, product, version and inventory reference for local matching. An assessment is not an active CVE check. Create a signer key in the native credential store, choose a new `.wfr` path and export; unsigned mode is explicit. **Verify** uses the local trust registry; importing another system's key requires an independently confirmed fingerprint through the [CLI procedure](M2-REPORTS.md). Export shows incomplete coverage and cache mode/freshness; no feed is fetched on startup.

The Language menu and selector switch IT/EN; shortcuts are **Ctrl+1/Ctrl+2** (**Cmd+1/Cmd+2** on macOS). **Ctrl+O/Cmd+O** loads examples. Full keyboard and screen-reader validation remains open.

The View menu adds search (Ctrl/Cmd+F), results (F6), evidence (Ctrl/Cmd+Shift+E) and advanced details (Ctrl/Cmd+Shift+D). See [keyboard, focus and assistive limitations](QT-DESKTOP.md).

## Platforms and bundle

M1 targets: Apple Silicon ARM64 macOS 26+, Windows 10 1809+/11 x86-64 and Ubuntu 24.04 LTS x86-64/ARM64; no 32-bit. Other Debian distributions/derivatives remain a compatibility goal. The bundle declares macOS 26.0 but has not been tried on a real 26.0 system; the Windows ZIP and `.deb` packages are tested in CI, not on every minimum-version workstation. Limits and future Windows 10 maintenance are in the [platform policy](M1-SUPPORT-POLICY.md) and [ADR-002](ADR-002-GUI.md). CI builds do not establish assistive use on the required systems.

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

In the historical M0 block, `internal/scope` contained the immutable origin policy and the HTTP lab existed only in `lab_test.go` ([M0 contract](M0-SCOPE.md)). The M1 desktop can make authorized requests through the controlled broker.

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

Driver and library selected in [ADR-004](ADR-004-STORAGE-SIGNATURE.md). `internal/foundation` contains SQLite tests on temporary files; `internal/signature` exposes byte signing/verification, now used by [M2 reports](M2-REPORTS.md) with JCS and native key storage. The GUI offers the essential workflow; rotation, revocation and public-key import remain in the CLI helper.

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
