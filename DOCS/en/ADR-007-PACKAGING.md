# ADR-007 — Debian and Windows trial packages

[Italiano](../it/ADR-007-PACKAGING.md) · [Index](../README.md)

Date: 2026-09-23. Status: implementation under verification. M0 development packages, not supported releases. Distribution, complete licensing and assistive trials remain in the [matrix](M0-VALIDATION.md).

## Decision

**Debian:** native amd64/arm64 `.deb`, built in an official Go 1.27.1 image based on Debian 12. Multiarch images pinned by digest; APT packages updated from Debian repositories, actual versions recorded. Debian 12 is a candidate technical baseline, not a new author-approved support promise. Qt remains a system dependency: no private glibc or Qt copy. `dpkg-shlibdeps` derives constraints from actual ELF symbols. Debian QtGui includes XCB/offscreen; `qt6-qpa-plugins` contains other backends and is not unnecessarily added. Wayland is suggested through `qt6-wayland`, still awaiting trials.

**Windows:** portable UCRT64 x86-64 ZIP, no installer, administrator privileges or Authenticode signature. `windeployqt6` collects Qt plugins; recursive PE import inspection adds DLLs from the same UCRT64 prefix. Only x86-64 PE files are accepted. System/API-set imports recorded; missing/ambiguous DLLs and packages without notices fail packaging. Import inspection does not cover every possible dynamic load: package execution and actual desktops are also required.

## Provenance and licensing

`package-go-notices.py` reads modules **actually linked** in the binary, checks version/hash against the Go graph and copies top-level module notices and the Go license. It rejects unreviewed local replacements. The whole `go.mod` is not treated as GUI contents. These are initial metadata/notices, not a complete SBOM or legal review.

The `.deb` contains LICENSE, IT/EN READMEs, Go metadata, native build versions, declared revision and source-input hashes. The container binary uses `-buildvcs=false`: external provenance and hashes distinguish this from embedded Git revision data. The dirty flag must describe the actual context; CI uses the current checkout, local modified trials declare dirty=true.

The ZIP records each DLL with copied/original hashes, its owning MSYS2 package, `pacman -Qi` metadata and copies of its notices. Qt files rewritten by windeployqt may differ from their originals. Dependency licenses are not replaced by WebFence licensing. Complete embedded-component inventory, corresponding sources and distribution materials remain pending. CI builds and tests packages without publishing a release or automatically uploading binary artifacts.

## Available commands

On a Docker host, build and test Debian for the native architecture:

```sh
docker build --progress=plain --file packaging/debian/Dockerfile \
  --build-arg SOURCE_REVISION="$(git rev-parse HEAD)" \
  --build-arg SOURCE_DIRTY=true \
  --output type=local,dest=dist/debian-test .
```

`SOURCE_DIRTY=true` explicitly marks a development trial; use false only with a verified clean context. Outputs include `.deb`, checksums and test logs. In a prepared Debian environment: `sh scripts/package-debian.sh`; requires Go, CGO/g++, Qt 6 development files, pkg-config, dpkg-dev, desktop-file-utils and Python 3. This is not cross-compilation.

In an MSYS2 UCRT64 terminal, after `go build -ldflags "-H=windowsgui -s -w" -o bin/webfence.exe ./cmd/webfence`, with UCRT64 Python installed:

```sh
python3 scripts/package-windows.py "$(cygpath -w /ucrt64)" "$(cygpath -w "$PWD/bin/webfence.exe")"
```

From PowerShell 7: `./scripts/test-windows-package.ps1 -Archive ./dist/webfence-windows-amd64.zip`. The test uses a temporary directory containing spaces/Unicode and child processes whose PATH contains only Windows directories; personal PATH is unchanged. Tests offscreen and the native Windows plugin. No customer or target data is included.

## Checks and limitations

The Debian runtime container starts from a slim image without Go, gcc or qmake. It installs the `.deb` through APT and runs self-tests as an unprivileged user, first offscreen and then XCB in Xvfb with Openbox. It purges the package and checks executable/launcher removal and preservation of a synthetic user preference. Logs and checksum export only if all stages pass. This checks declared dependencies and package behavior in the container, not an actual Debian desktop, Wayland, Orca, GPU or physical installation.

First XCB trial: focus tests failed in an X server without a window manager. The trial now waits for a dedicated window manager; the app self-test waits up to three seconds for asynchronous window activation before keyboard checks and explicitly fails if activation never occurs. Normal interactive execution is unchanged.

Windows CI uses Windows Server, not Windows 10/11. Sanitized PATH and execution from an extracted ZIP provide stronger evidence than building alone, but are not a completely clean machine or NVDA trial. Record native/CI outcomes after actual execution.

Sources: [dpkg-shlibdeps](https://manpages.debian.org/bookworm/dpkg-dev/dpkg-shlibdeps.1.en.html), [Debian QtGui files](https://packages.debian.org/bookworm/arm64/libqt6gui6/filelist), [Qt Windows deployment](https://doc.qt.io/qt-6/windows-deployment.html), [MSYS2 UCRT64 Qt](https://packages.msys2.org/packages/mingw-w64-ucrt-x86_64-qt6-base).

Local ARM64 trial on 2026-09-23: `.deb` build, installation in the runtime without toolchain, unprivileged offscreen and XCB/Openbox self-tests and purge passed. Exported logs/checksum verified. Windows and Debian x86-64 await CI; no equivalence to actual desktops assumed. Windows builds use the GUI subsystem to avoid an extra console.
