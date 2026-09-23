# M0 — Packaging investigation

[Italiano](../it/M0-PACKAGING.md) · [Validation](M0-VALIDATION.md) · [Index](../README.md)

Date: 2026-09-23. Development artifacts only; no supported release or completed distribution review.

## Inspected macOS artifact

The first inspected `dist/WebFence.app` was a development bundle predating the current commit; the following data are historical, and later bundle verification appears below. `go version -m` identifies revision `c4ce9cab56635db726e590ad0c2c09f20d0f18ee` with `vcs.modified=true`, Go 1.27.1 and MIQT 0.14.0. Main executable SHA-256: `de663edcda2eceddafb5e020a9b944949371f1bb20c569178d7f03c6a4048436`. This identifies the inspected binary, not a signed release manifest.

A read-only scan of every Mach-O file, excluding symlink duplicates, found **28 binaries/libraries/plugins**. `otool -l` reports minimum macOS **14.0 for 9** and **26.0 for 19**, including the main binary. QtCore itself declares 14.0; glib and ICU in this bundle declare 26.0. Thus this artifact requires macOS 26 despite Qt's broader upstream platform support. Changing Info.plist or the Go executable's deployment target cannot lower requirements embedded in its bundled libraries.

`otool -L` found no non-system absolute dependency paths after excluding each library's own `LC_ID_DYLIB` identity (`otool -D`). Some Qt framework identities still contain Homebrew paths; those are not an external dependency edge by themselves. This check does not prove all runtime plugin loading, code-signature behavior or clean-machine launch. Existing local ad hoc signature validation passed; Developer ID signing and notarization are not provided.

For a lower supported minimum, build or acquire every native dependency for that baseline, set consistent deployment flags, inspect all resulting load commands, and test on that OS. Do not patch minimum-version load commands to conceal incompatible code. The author has been asked whether the intended minimum should be macOS 13, 15 or 26; no answer or support promise is presumed.

## Inventory starting point

Observed frameworks: QtCore, QtGui, QtWidgets, QtDBus. Separate bundled dylibs include libb2, D-Bus, double-conversion, FreeType, GLib/GThread, Graphite2, HarfBuzz, ICU, gettext/libintl, libjpeg-turbo, libpng, md4c, PCRE2 and Zstandard. Qt plugins and embedded third-party code also require attribution; counting separate files is not a complete dependency inventory.

The installed Homebrew Qt 6.11.2 tree supplies both `sbom.spdx.json` and `share/qt/sbom/qtbase-6.11.2.spdx`. These are useful upstream inputs, **not a WebFence SBOM**: they include build tools and components absent from the app, and do not by themselves identify every copied library version. Retain artifact hashes, build provenance, the actual linked Go module list, source archive hashes, patches and notices when generating a package. `go.mod` lists foundations that are not all linked into the current GUI; do not present its entire graph as the binary's contents.

Before distributing a package, assemble the actual component licenses/notices and required corresponding-source/replacement materials, and resolve compatibility through the [legal review](LEGAL-REVIEW.md). Homebrew formula license expressions describe a whole formula, not necessarily each embedded component. No dependency's license is replaced by WebFence's license; no commercial Qt purchase or legal approval has occurred.

## Linux and Windows packages

Debian 12 provides Qt 6.4.2; Debian 13 provides Qt 6.8.2 at inspection time. The native `.deb` is now built and tested in Debian 12, with derived ELF dependencies and system-provided Qt. Both amd64/arm64 passed install, offscreen/XCB self-tests and purge in the runtime container without a toolchain ([CI](https://github.com/Matte2599/WebFence/actions/runs/35876057107)). This is not a physical desktop, Wayland or Orca trial. [ADR-007](ADR-007-PACKAGING.md) documents the procedure and limits.

The Windows CI uses MSYS2 UCRT64 on Windows Server. The executable requires Qt plugins/DLLs and compiler/runtime dependencies; the exe alone is insufficient. The ZIP script collects the runtime and CI includes execution without MSYS2 on PATH; outcomes are tracked in [ADR-007](ADR-007-PACKAGING.md). Actual Windows 10/11 desktop and assistive trials remain pending. Qt 6.11 lists Windows 10 1809+ and Windows 11; Qt documents 6.12 as the last branch supporting Windows 10. This is an upstream constraint, not completed WebFence verification or a promise of indefinite maintenance.

Sources: [Qt supported platforms](https://doc.qt.io/qt-6/supported-platforms.html), [Debian 12 Qt](https://packages.debian.org/bookworm/libqt6core6), [Debian 13 Qt](https://packages.debian.org/trixie/libqt6core6t64), local Mach-O metadata and installed Qt SBOMs. Package versions and host metadata were inspected on the stated date.

## Build process update

The macOS script now stages the bundle on the host temporary filesystem, verifies the ad hoc signature, copies completely onto the destination volume and publishes by rename; it retains the old artifact until replacement and restores it if publication fails. This avoids repeated rewrites on an external disk and a partial final bundle during the build. Local build and both codesign checks passed. An intentionally injected compiler error failed as expected, preserved the previous executable hash and cleaned staging. No speed benchmark is claimed.

The macOS bundle now rebuilds the Qt 6.11.2 Cocoa plugin with a temporary assistive-crash correction: [ADR-006](ADR-006-QT-COCOA.md). CMake, Ninja and MoltenVK/Vulkan headers are also required (`brew install cmake ninja molten-vk vulkan-headers`); Qt sources are downloaded and hash-verified. Other Qt versions are rejected until reassessed. Unpackaged binaries continue using installed Qt.

Updated local bundle verified on 2026-09-23: Go revision `953ed2daf5061b4044f7cd9d8ff72b133153d260`, `vcs.modified=false`. Executable SHA-256 `4af00ede347509677b52c55bd8a355a77faef6d9879327868ae325ef124b0abf`; corrected Cocoa plugin `acc31e86a786050c2e7280c5be09df30718262d7b4514dc5d82f65e8d13d5511`. Build and ad hoc signature verified; [four-target CI passed](https://github.com/Matte2599/WebFence/actions/runs/35873709410). These hashes identify local artifacts, not a guarantee of bit-for-bit reproducible builds. The new bundle minimum OS was not lowered.

Added `.deb`/Windows ZIP packaging and separate-runtime checks: [ADR-007 and commands](ADR-007-PACKAGING.md). [CI `32655b0` passed, six jobs](https://github.com/Matte2599/WebFence/actions/runs/35877356234); development packages, not supported releases.

## Automated macOS collection

`package-macos-notices.py` runs against the staged bundle before final signing. Requires Python 3.9+ and Apple developer tools. For every Mach-O it checks a single ARM64 slice and finds a unique name/UUID match among installed Homebrew kegs in Qt's runtime closure. Includes actually matched versions, installed recipes/patches, receipts, upstream SBOMs and available notices. UUID is a provenance hint, not cryptographic proof. Unknown/ambiguous native files fail the build; dependencies listed in the receipt but absent from the app are not inventoried as shipped.

`Contents/Resources/notices` also collects actual linked Go modules, LICENSE and IT/EN documentation, the Cocoa patch, its LGPL/GPL texts and rebuild script. Hashes are labelled **before final signing**, which may change binaries; do not present them as signed-bundle checksums. Final signing protects the added resources.

The manifest sets `distribution_ready: false`. The local probe matched 28 total Mach-O files (executable, rebuilt Cocoa and 26 originals) to 15 Homebrew packages. GLib supplies no installed notices; installed Qt notices concern CMake tools and do not suffice for Qt runtime. Complete source archives, embedded components, further notices and replacement materials remain to collect/verify. Upstream SBOMs are retained as sources, not declared product SBOMs. Script success alone grants no distribution approval.

Verified rejection of an unrelated Mach-O component and preservation of an existing inventory. Local bundle integration completed: ad hoc signature verified, 28 records/15 packages checked, bundled documentation links valid. Local build from code `e683437` with declared modifications; change CI still to verify. The GLib recipe also references a Homebrew patch absent from the keg: it remains among source materials to acquire.
