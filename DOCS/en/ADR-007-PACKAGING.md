# ADR-007 — Debian and Windows trial packages

[Italiano](../it/ADR-007-PACKAGING.md) · [Index](../README.md)

Date: 2026-09-23. Status: implemented; platform CI verification recorded below. M0 development packages, not supported releases. Distribution, complete licensing and assistive trials remain in the [matrix](M0-VALIDATION.md).

## Decision

**Debian:** native amd64/arm64 `.deb`, built in an official Go 1.27.1 image based on Debian 12. Multiarch images pinned by digest; APT packages updated from Debian repositories, actual versions recorded. Debian 12 is a candidate technical baseline, not a new author-approved support promise. Qt remains a system dependency: no private glibc or Qt copy. `dpkg-shlibdeps` derives constraints from actual ELF symbols. Debian QtGui includes XCB/offscreen; `qt6-qpa-plugins` contains other backends and is not unnecessarily added. Wayland is suggested through `qt6-wayland`, still awaiting trials.

**Windows:** portable UCRT64 x86-64 ZIP, no installer, administrator privileges or Authenticode signature. `windeployqt6` collects Qt plugins; recursive PE import inspection adds DLLs from the same UCRT64 prefix. Only x86-64 PE files are accepted. System/API-set imports recorded; missing/ambiguous DLLs and packages without notices fail packaging. Import inspection does not cover every possible dynamic load: package execution and actual desktops are also required.

## Provenance and licensing

`package-go-notices.py` reads modules **actually linked** in the binary, checks version/hash against the Go graph and copies top-level module notices and the Go license. It rejects unreviewed local replacements. The whole `go.mod` is not treated as GUI contents. These are initial metadata/notices, not a complete SBOM or legal review.

The `.deb` contains LICENSE, IT/EN READMEs, Go metadata, native build versions, declared revision and source-input hashes. The container binary uses `-buildvcs=false`: external provenance and hashes distinguish this from embedded Git revision data. The dirty flag must describe the actual context; CI uses the current checkout, local modified trials declare dirty=true.

The ZIP records each DLL with copied/original hashes, its owning MSYS2 package, `pacman -Qi` metadata and copies of its notices. Selected notices are hash-compared with their members in cached MSYS2 binary archives and rechecked after ZIP extraction: [43 DLLs, 22 packages and 74 notices verified in CI](../evidence/windows-notice-linkage-2026-09-24.md). Qt files rewritten by windeployqt may differ from their originals. Dependency licenses are not replaced by WebFence licensing. Complete embedded-component inventory, corresponding sources and distribution materials remain pending. CI builds and tests packages without publishing a release or automatically uploading binary artifacts.

SHA-256 values for the 22 original binary archives are now [bound to checksums published on MSYS2 package pages](../evidence/windows-binary-checksum-lock-2026-09-24.md). The reviewed lock is included in the ZIP, and the extracted-archive trial rechecks its relationship to `native-build.json`. New versions require explicit review; this check does not replace PGP signatures, embedded-code inventory or legal review.

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

The Debian runtime container starts from a slim image without Go, gcc or qmake. It installs the `.deb` through APT and runs self-tests as an unprivileged user, first offscreen and then XCB in Xvfb with Openbox. A separate trial runs the normal GUI in a private Xvfb/D-Bus session, reads the AT-SPI tree in English and Italian and checks the table sequence 10,000 → 1 → 0 → 10,000 with a valid first cell after reset; [procedure and evidence](../evidence/debian-atspi-2026-09-24.md). Python, AT-SPI and D-Bus are dependencies of the **test container only**, not the `.deb`. It purges the package and checks executable/launcher removal and preservation of a synthetic user preference. Logs and checksum export only if all stages pass. This checks declared dependencies and package behavior in the container, not an actual Debian desktop, Wayland, Orca announcements, GPU or physical installation.

First XCB trial: focus tests failed in an X server without a window manager. The trial now waits for a dedicated window manager; the app self-test waits up to three seconds for asynchronous window activation before keyboard checks and explicitly fails if activation never occurs. Normal interactive execution is unchanged.

Windows CI uses Windows Server, not Windows 10/11. Sanitized PATH and execution from an extracted ZIP provide stronger evidence than building alone, but are not a completely clean machine or NVDA trial. Actual outcomes are recorded below.

Sources: [dpkg-shlibdeps](https://manpages.debian.org/bookworm/dpkg-dev/dpkg-shlibdeps.1.en.html), [Debian QtGui files](https://packages.debian.org/bookworm/arm64/libqt6gui6/filelist), [Qt Windows deployment](https://doc.qt.io/qt-6/windows-deployment.html), [MSYS2 UCRT64 Qt](https://packages.msys2.org/packages/mingw-w64-ucrt-x86_64-qt6-base).

Local ARM64 trial on 2026-09-23: `.deb` build, installation in the runtime without toolchain, unprivileged offscreen and XCB/Openbox self-tests and purge passed. Exported logs/checksum verified. This local trial does not establish equivalence to actual desktops. Windows builds use the GUI subsystem to avoid an extra console.

CI `ea2896a`, [run 35876057107](https://github.com/Matte2599/WebFence/actions/runs/35876057107): macOS/Linux builds/tests and Debian amd64/arm64 packages passed. Windows passes build/self-test but packaging stops because ICU stores LICENSE in `share/icu/<version>/LICENSE`. Collection corrected to include explicitly named notices under `share`, still owned by the installed package; missing notices continue to fail packaging. [MSYS2 ICU inventory](https://packages.msys2.org/packages/mingw-w64-ucrt-x86_64-icu). Subsequent verification recorded below. Added Installed-Size to the `.deb`; synthetic purge-test preference created by the same unprivileged user.

Windows verified on commit `32655b0` ([CI 35877356234](https://github.com/Matte2599/WebFence/actions/runs/35877356234)): complete ZIP, corrected ICU notices collection, offscreen and native Windows backend self-tests passed with system-only PATH and a spaced/Unicode directory. Runner ZIP SHA-256: `9c5226fbb452fe8f0f2329496f0bc80cfff01532215df7d4dedb943735f5c6f1`; ephemeral artifact not published. This does not establish Windows 10/11 or NVDA behavior.

Final task outcome: **all six jobs passed** on code `32655b0` ([run](https://github.com/Matte2599/WebFence/actions/runs/35877356234)). Debian amd64 and arm64 passed build, install, offscreen/XCB self-tests and purge. Ephemeral CI `.deb` hashes:

| Architecture | SHA-256 |
| --- | --- |
| amd64 | `50c539bc43ab9810e7c7a87723d8b2acbf731462ca3ec711e3d6ec8504d1cbd1` |
| arm64 | `a0ae5104727feca727d75c214021253fffa464f56a152a39f8909a5cad924405` |

Hashes identify this run; timestamps and APT/MSYS2 packages may change between builds. Bit-for-bit reproducibility is not claimed. M0-02 remains partial for the gates in the matrix.

## Offline documentation in packages

The shared `scripts/package-project-docs.py` collector preserves IT/EN READMEs, LICENSE, DOCS, memory, contributor/security instructions and locally referenced materials. macOS, Windows and Debian use it before publishing an artifact. Windows documentation is under the ZIP’s `WebFence` directory; Debian uses `/usr/share/doc/webfence`, with LICENSE identical to the conventional `copyright` file; macOS uses `Contents/Resources/notices`. The Cocoa patch in Linux/Windows documentation is reference material, not a claim that those binaries use it.

Packaging checks that local Markdown links resolve to included files, without network access. It does not verify remote URLs or heading anchors, or turn documented commands into product capabilities. Existing documents are not overwritten; errors fail staging. To recheck a packaged documentation tree: `python3 scripts/package-project-docs.py PATH --check`.

Fixes a defect in earlier Windows/Debian packages: READMEs linked to absent DOCS, and Debian retained LICENSE only as `copyright`. The Docker context now includes documentation too; Debian source-input hashes cover it. Extracted-ZIP and installed-Debian-package trials check IT/EN documentation entry points and the license.

Local collector verification: 68 Markdown documents and 519 local links; intentionally removed guide detected, existing documents preserved. macOS bundle signature/Cocoa self-test passed; ARM64 `.deb` built, installed and purged with passing offscreen/XCB tests. Documentation extracted from the `.deb`: 68 Markdown files/521 valid links, LICENSE identical to copyright. Local `.deb` SHA-256: `e5c175563b2f6d5479ec873644fd20b7601fd12bcf635fbb6ddceae71c8f094a`. Build from `a4c621c` with declared modifications. [CI `f99b3e7`, run 35900777988](https://github.com/Matte2599/WebFence/actions/runs/35900777988): all six jobs passed, included documentation verified in Windows and Debian logs.

The first installed-package trial found that `bookworm-slim` excludes documentation through dpkg. Only the test container now explicitly retains `/usr/share/doc/webfence`; the `.deb` does not change the user’s dpkg policy. Systems configured to omit documentation remain under administrator control.

## Matching MSYS2 binary packages

Windows packaging now requires exactly one cached binary archive of the installed version of each package owning included DLLs. Retain `/var/cache/pacman/pkg` after installing dependencies. Missing/ambiguous archives or mismatched versions fail packaging before ZIP replacement; dependencies are not automatically upgraded.

`scripts/msys2_binary_metadata.py` streams archives without extracting or executing code, matches each original DLL against its archive member and retains `.PKGINFO`/`.BUILDINFO`. It compares name, version, source package and architecture, recording archive and recipe hashes. [BUILDINFO v2](https://man.archlinux.org/man/BUILDINFO.5.en) defines `pkgbuild_sha256sum`; for split packages the DLL owner must occur in the list of produced packages. Matching uses the original DLL: `windeployqt` may modify the packaged copy.

The manifest adds `binary_package_member` per DLL and `binary_package` per owner, with metadata and hashes; retains `distribution_ready=false`. Toolchain `zstd.exe` only decompresses archives during the build. Per-archive limits: 512 MiB compressed, 2 GiB declared expanded, 100,000 members and 1 MiB per metadata file; unsafe paths, duplicate metadata and links replacing required evidence are rejected. The cache is trusted build input, not scanner/AI input.

This establishes correspondence between cached archives and installed DLLs; it does not verify pacman signatures, source availability or license completeness. The hash-identified recipe can be acquired and compared separately. Local reader trial against the real Qt package and 16 Python regressions passed; complete CI verification is recorded below.

First CI `ceef622` ([run 35903124589](https://github.com/Matte2599/WebFence/actions/runs/35903124589)): Windows reaches packaging but fails because the Qt archive is absent. Logs show the pinned MSYS2 action runs `pacman -Scc` after saving its cache. Set `cache: false` for that action only: retains downloaded packages in the job for verification, at the cost of downloading them again in subsequent jobs. Go caching unchanged; archive checks not relaxed. [Corrective CI `054bb2d`](https://github.com/Matte2599/WebFence/actions/runs/35904059667) completed: all six jobs passed. Windows verifies 43 DLLs from 22 packages, with 21 distinct source packages; ZIP and GUI tests passed. [Recipe table and hashes](../evidence/windows-native-provenance-2026-09-23.md).

Windows packaging also supports [verified MSYS2 source collection and attachment](M0-WINDOWS-SOURCES.md#include-sources-in-the-zip), with explicit options and an extracted-ZIP trial. Distribution obligations still require review.
