# M0 — Packaging investigation

[Italiano](../it/M0-PACKAGING.md) · [Validation](M0-VALIDATION.md) · [Index](../README.md)

Date: 2026-09-23. Development artifacts only; no supported release or completed distribution review.

## Inspected macOS artifact

The existing local `dist/WebFence.app` is a development bundle, not the current repository commit. `go version -m` identifies revision `c4ce9cab56635db726e590ad0c2c09f20d0f18ee` with `vcs.modified=true`, Go 1.27.1 and MIQT 0.14.0. Main executable SHA-256: `de663edcda2eceddafb5e020a9b944949371f1bb20c569178d7f03c6a4048436`. This identifies the inspected binary, not a signed release manifest.

A read-only scan of every Mach-O file, excluding symlink duplicates, found **28 binaries/libraries/plugins**. `otool -l` reports minimum macOS **14.0 for 9** and **26.0 for 19**, including the main binary. QtCore itself declares 14.0; glib and ICU in this bundle declare 26.0. Thus this artifact requires macOS 26 despite Qt's broader upstream platform support. Changing Info.plist or the Go executable's deployment target cannot lower requirements embedded in its bundled libraries.

`otool -L` found no non-system absolute dependency paths after excluding each library's own `LC_ID_DYLIB` identity (`otool -D`). Some Qt framework identities still contain Homebrew paths; those are not an external dependency edge by themselves. This check does not prove all runtime plugin loading, code-signature behavior or clean-machine launch. Existing local ad hoc signature validation passed; Developer ID signing and notarization are not provided.

For a lower supported minimum, build or acquire every native dependency for that baseline, set consistent deployment flags, inspect all resulting load commands, and test on that OS. Do not patch minimum-version load commands to conceal incompatible code. The author has been asked whether the intended minimum should be macOS 13, 15 or 26; no answer or support promise is presumed.

## Inventory starting point

Observed frameworks: QtCore, QtGui, QtWidgets, QtDBus. Separate bundled dylibs include libb2, D-Bus, double-conversion, FreeType, GLib/GThread, Graphite2, HarfBuzz, ICU, gettext/libintl, libjpeg-turbo, libpng, md4c, PCRE2 and Zstandard. Qt plugins and embedded third-party code also require attribution; counting separate files is not a complete dependency inventory.

The installed Homebrew Qt 6.11.2 tree supplies both `sbom.spdx.json` and `share/qt/sbom/qtbase-6.11.2.spdx`. These are useful upstream inputs, **not a WebFence SBOM**: they include build tools and components absent from the app, and do not by themselves identify every copied library version. Retain artifact hashes, build provenance, the actual linked Go module list, source archive hashes, patches and notices when generating a package. `go.mod` lists foundations that are not all linked into the current GUI; do not present its entire graph as the binary's contents.

Before distributing a package, assemble the actual component licenses/notices and required corresponding-source/replacement materials, and resolve compatibility through the [legal review](LEGAL-REVIEW.md). Homebrew formula license expressions describe a whole formula, not necessarily each embedded component. No dependency's license is replaced by WebFence's license; no commercial Qt purchase or legal approval has occurred.

## Linux and Windows next work

Debian 12 provides Qt 6.4.2; Debian 13 provides Qt 6.8.2 at inspection time. The current Ubuntu CI is useful but does not prove a Debian package baseline. Build against the oldest selected runtime, derive dependency requirements from actual binaries and test installation in a clean matching system. For a distribution-native `.deb`, prefer OS Qt dependencies with explicit package/version requirements; a portable package needs its own verified dependency bundle. No Linux installer exists yet.

The Windows CI uses MSYS2 UCRT64 on Windows Server. The executable requires Qt plugins/DLLs and compiler/runtime dependencies; the exe alone is insufficient. Stage the complete runtime, test with MSYS2 absent from PATH, and then test actual Windows 10/11 desktop and assistive flows. Qt 6.11 lists Windows 10 1809+ and Windows 11; Qt documents 6.12 as the last branch supporting Windows 10. This is an upstream constraint, not completed WebFence verification or a promise of indefinite maintenance.

Sources: [Qt supported platforms](https://doc.qt.io/qt-6/supported-platforms.html), [Debian 12 Qt](https://packages.debian.org/bookworm/libqt6core6), [Debian 13 Qt](https://packages.debian.org/trixie/libqt6core6t64), local Mach-O metadata and installed Qt SBOMs. Package versions and host metadata were inspected on the stated date.

## Build process update

The macOS script now stages the bundle on the host temporary filesystem, verifies the ad hoc signature, copies completely onto the destination volume and publishes by rename; it retains the old artifact until replacement and restores it if publication fails. This avoids repeated rewrites on an external disk and a partial final bundle during the build. Local build and both codesign checks passed. An intentionally injected compiler error failed as expected, preserved the previous executable hash and cleaned staging. No speed benchmark is claimed.
