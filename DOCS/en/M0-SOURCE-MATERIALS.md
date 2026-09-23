# M0 — Source material collection

[Italiano](../it/M0-SOURCE-MATERIALS.md) · [Packaging](M0-PACKAGING.md) · [Index](../README.md)

## Available capability

`scripts/collect-native-sources.py` collects upstream archives and notices into a directory separate from the macOS bundle. Input is the `native-build.json` produced by our packaging, **a trusted build input**, never scanned-site or AI text. It does not execute archive scripts or modify the signed app, installed Qt or personal preferences.

Requires Python 3.9+ and curl 8.4+ for downloads: [from this version the size limit also applies without Content-Length](https://curl.se/docs/manpage.html#--max-filesize). URLs and SHA-256 hashes come from Homebrew SBOMs retained in the inventory. Matching establishes correspondence with those metadata, not independent verification of their publisher’s authenticity.

```sh
python3 scripts/collect-native-sources.py \
  dist/WebFence.app/Contents/Resources/notices/native-build.json \
  /tmp/webfence-source-materials

# Optional: reuse a local archive, verified again by hash.
# Add --reuse-archive PATH to the command; this option is repeatable.
python3 -m unittest discover -s scripts/tests -p 'test_*.py' -v
```

The destination directory must be new. Output contains original archives, copied notices with paths and hashes, links recorded but never followed, a copy of the original inventory and `source-materials.json`. `INCOMPLETE` remains until success; an error does not produce the final manifest. Downloaded materials are retained: they can be inspected and reused, with fresh verification, in a new destination. Do not distribute incomplete output.

HTTPS only, including redirects; at most five redirects, 180 s per transfer and stall interruption. Personal `.curlrc` is not loaded. Limits: 32 archives, 128 MiB each, 512 MiB combined; per archive 100,000 members and 2 GiB declared uncompressed, 4 MiB per notice and 32 MiB of notices. Tar reading is streamed: no general extraction, symlink creation or code execution. Absolute/traversal paths, duplicate notices, wrong hashes and exceeded budgets prevent success.

Copies common license/notice names, `LICENSES` directories, `qt_attribution.json` metadata and files explicitly referenced by FreeType’s `LICENSE.TXT`. A future FreeType version missing an expected reference requires review. Some notices occur inside source files, which are retained whole. This selection assists review; it does not prove that all attribution duties are covered.

## Verification on 2026-09-23

Clean bundle `8d3053f` inventory: **15 archives, 170,920,531 bytes (about 163 MiB), 247 notice/metadata files**. Actual downloads verified; Qt and GLib reused from hash-verified local archives. Second collection entirely from verified archives, to include FreeType references initially missed by generic names. GLib now supplies the `LICENSES` texts absent from the installed keg; `COPYING` remains an archive link, recorded without following it.

[Collection manifest and hashes](../evidence/macos-sources-2026-09-23/source-materials.json) and [input inventory](../evidence/macos-sources-2026-09-23/native-build.input.json) are retained in the repository; archives and collected texts remain in `/tmp/webfence-native-sources-8d3053f-reviewed/`, outside Git. Seven synthetic tests passed: existing-file preservation, wrong hashes, paths and duplicates, bounds, missing references, invalid metadata and curl requirements. [CI `39c48ef` passed, six jobs](https://github.com/Matte2599/WebFence/actions/runs/35888497914): seven collector tests on four targets, plus product regressions and packaging.

Independently checked every archive and notice hash. A fresh GLib transfer with the final curl configuration succeeded; a 1 KiB limit rejects the download without storing bytes. Local curl reports this rejection with code 56 and a maximum-file-size message: the collector treats any nonzero exit as failure, without depending on one specific code.

## Remaining boundaries

The manifest retains **`distribution_ready: false` and `corresponding_sources_complete: false`**. These are upstream sources of identified libraries, not yet all corresponding sources of the distribution: package-manager patches/resources, build environment and rebuild/replacement instructions are needed. The GLib supplement below covers part of these materials; the rest of the closure still requires verification and assembly.

Whole-source-tree notices can cover unshipped components; map them to actual binaries and choose/verify applicable terms. Recipes, SBOMs and the Cocoa patch remain in the original bundle identified by the inventory. Notices can now be attached explicitly before signing, using the procedure below; archives remain separate and no release is published. Legal review, minimum OS versions, clean desktop trials and Windows materials remain part of M0-02/M0-06; collecting files does not approve them.

## Identified GLib supplement

The [Homebrew recipe at revision 550d1a4](https://github.com/Homebrew/homebrew-core/blob/550d1a4e5c2da2dd3c8c630e43b1ef9a84afd700/Formula/g/glib.rb) matches the installed recipe byte for byte after removing only its `bottle` stanza. Acquired `Patches/glib/hardcoded-paths.diff` from the same Git tree and the gobject-introspection 1.86.0 archive, with SHA-256 verified against the recipe. Patch dry-run on verified GLib 2.90.0 source passes without fuzz; no installation, code execution or keg modification.

[Provenance and hashes](../evidence/macos-sources-2026-09-23/glib-supplement.json) retained; materials in `/tmp/webfence-glib-materials-550d1a4/`, separate from automated output. Matching identifies recipe materials without proving a bit-identical bottle rebuild. Integration into the complete package and verification of other recipes remain open.

The libb2 0.98.1 Big Sur configure patch was also acquired from the recipe’s pinned URL, SHA-256 checked and tested with a no-fuzz dry-run: [ledger](../evidence/macos-sources-2026-09-23/libb2-supplement.json), local file `/tmp/webfence-libb2-materials/configure-big_sur.diff`. The D-Bus patch is already embedded in its retained recipe. Other recipes contain textual substitutions and test resources (fonts/images): do not automatically equate them with shipped content; retain recipes and verify each material’s role during assembly.

## Attach verified notices to a development bundle

After collecting sources, rebuild with the optional input directory:

```sh
WEBFENCE_NATIVE_SOURCE_MATERIALS=/tmp/webfence-source-materials \
  sh scripts/package-macos.sh
```

`scripts/attach-native-sources.py` compares package identities, source URLs and SHA-256 hashes against the new bundle inventory. It rechecks every source archive and regenerates notices directly from the archive, comparing them with the collection manifest. Modified loose notice files are ignored. Incomplete collections, different dependency versions, corrupted archives, mismatched notice metadata, symlinks in input materials and exceeded collection budgets fail before attachment. Existing attachment directories are preserved; temporary output is removed on failure. Inputs are trusted build artifacts in a directory owned by the builder; concurrent modification is unsupported.

Output is `Contents/Resources/notices/upstream-source`, included before final ad hoc signing. `attachment.json` binds the current inventory hash separately from acquisition metadata: the collection's original WebFence commit is not presented as the new executable's commit. All 258 source-tree notices can be included for the current local dependencies. Full archives remain in the separate collection directory; archive paths in the copied manifest refer to that directory, not the app. Keep both materials when preparing a future distribution. Supplementary Homebrew patches/resources still require assembly.

This option performs no downloads. Without the variable, development packaging retains its existing installed notices and inventory. Both manifests retain `distribution_ready=false`; copied source-tree notices can include unshipped components and require mapping/review. The helper alone must not be used to alter an already signed bundle: use the packaging entry point so resources are signed and failed staging cannot replace the previous app.

Local verification: 12 Python regressions passed (seven collection, five attachment), actual extraction from 15 archives and independent recheck of all 247 hashes. Bundle from `0fd78f0` with declared modifications: ad hoc signature, current-inventory binding and Cocoa self-test passed. Previous CI `0fd78f0`, run 35897426940, completed all six jobs; [CI `56840c5`](https://github.com/Matte2599/WebFence/actions/runs/35898848286) for the new changes completed: all six jobs passed, including 12 Python regressions on four targets. Actual attachment of the 247 notices was verified locally; CI packaging still runs without the optional variable.

Full-packaging failure injection: a synthetic collection with `INCOMPLETE` was rejected; previous executable and attachment hashes unchanged, previous ad hoc signature still valid.

## First Windows acquisition

Acquired the MSYS2 Qt 6.11.2-2 archive with upstream source, recipe and ten patches; internal hashes checked without executing code. [Provenance, hashes and limits](../evidence/windows-qt-source-2026-09-23.md). The subsequent [Windows collection](M0-WINDOWS-SOURCES.md) covers 21 source packages with recipes bound to binaries; complete assembly and review remain open.

The Windows manifest now binds DLLs to cached binary packages and the source recipe hash: [procedure and limits](ADR-007-PACKAGING.md#matching-msys2-binary-packages). Locally verified the source PKGBUILD ↔ Qt binary BUILDINFO binding against a published-checksum-verified package too; detached signature still not verified.

## Homebrew supplements in the macOS bundle

The reviewed `packaging/macos/homebrew-supplements.json` plan identifies four files: the GLib recipe at the previously verified Homebrew revision, GLib path patch, gobject-introspection 1.86.0 and libb2 configure patch. Each group is bound to the installed recipe hash in the bundle as well as the inventory name/version. The plan is a trusted build input to review when dependencies change: the program does not interpret Ruby or automatically infer every required resource.

```sh
# Explicit attachment before signing, with verified HTTPS downloads.
WEBFENCE_NATIVE_SUPPLEMENT_PLAN=packaging/macos/homebrew-supplements.json \
  sh scripts/package-macos.sh

# Separate collection without modifying a signed application.
python3 scripts/native_supplements.py \
  dist/WebFence.app/Contents/Resources/notices/native-build.json \
  packaging/macos/homebrew-supplements.json \
  /tmp/webfence-homebrew-supplements
```

The variable is independent of `WEBFENCE_NATIVE_SOURCE_MATERIALS`: the 258 upstream notices and supplements can be included together. Without an explicit plan packaging acquires no supplements. The standalone collector accepts `--reuse-directory DIRECTORY` to reuse files in `package@version/filename` layout, verifying without network access. The destination must be new; output is published only after all checks pass. Failure removes temporary staging and preserves existing destinations.

Limits: 16 packages, 64 files, 16 MiB per file/recipe and 64 MiB combined supplements; 16 MiB per manifest. Credential-free HTTPS with the curl limits described above; size and SHA-256 checked after copying too. Mismatched versions, recipes or paths, duplicates, links and incorrect hashes are rejected. No patch, recipe or archive content is executed. Builder-controlled inputs and staging, without concurrent writers.

Bundle output is `Contents/Resources/notices/homebrew-supplements`, included before ad hoc signing. It contains the four materials, two installed recipes, plan, current inventory copy and `attachment.json` with hashes and provenance. Retains `distribution_ready=false` and `corresponding_sources_complete=false`: this plan covers GLib/libb2, not every Homebrew resource, environment or distribution obligation.

Local verification on 2026-09-23: actual download of four files/1,093,639 bytes and matching two recipes succeeded; 30 Python regressions passed. Local bundle with 247 notices and supplements: hashes rechecked, ad hoc signature and Cocoa self-test passed. Invalid plan rejected by full packaging: previous executable, attachment and language preference unchanged, previous signature valid. Final CI is recorded below.

CI `62a8b54`, run 35910855230: five jobs passed; macOS rejects the plan because a requested version is absent from the runner inventory; logs show GLib was not upgraded as a transitive Qt dependency. Preparation corrected by explicitly requesting GLib from Homebrew and recording before/after versions; plan hashes and versions remain mandatory, no automatic acceptance of different recipes. Corrective CI is recorded below.

Attempt `b6467cb` shows GLib 2.88.3 before/after: the runner Homebrew catalog still considers it current. Added explicit `brew update` before installation; the 2.90.0 plan is unchanged. Also verified arm64_sequoia packages against published checksums: GLib uses the same reviewed recipe, libb2 a recipe with identical source/patch/build but different no_autobump directive and test delimiter. The plan enumerates both verified libb2 hashes, without normalization or accepting other hashes. 31 local regressions passed; final CI is recorded below. [Comparison evidence](../evidence/macos-sources-2026-09-23/sequoia-recipe-check.json).

Final outcome: [CI `5521c34`](https://github.com/Matte2599/WebFence/actions/runs/35912249024), all six jobs passed. 31 Python regressions passed on four targets; the macOS log confirms GLib 2.88.3 → 2.90.0, the Sequoia libb2 hash and checks of four supplements/two recipes in the signed bundle, followed by Cocoa self-test and short soak. Windows again passes collection/attachment of 21 archives, extracted-ZIP checks and ZIP preservation on failure; Debian amd64/arm64 passes installation, GUI and removal. Intermediate run 35911763472 was superseded by the new push after the macOS failure: both native Linux jobs passed, the other three were cancelled; it is not a successful complete CI run.

Rebuilding and replacing the Cocoa plugin with a diagnostic modification: [procedure and evidence](M0-QT-REPLACEMENT.md). Shipped bundle materials reused, same Go build ID, self-test and four Cocoa cycles passed locally; originals preserved. This proves replacement of the plugin only, not the entire distribution.

## Qt license references completed in the collector

Comparing `LicenseFile`/`LicenseFiles` in Qt 6.11.2 metadata found **11 notices omitted** by generic filename matching: three FreeType texts, IJG, PSL, two SHA3, DejaVu and three Wayland texts. The collector now reads metadata first and files second, so it also works when a text precedes its reference in the archive. It records metadata/path pairs in `qt_license_references`. Literal newlines used by Qt in JSON strings are accepted; paths remain separately validated.

Parent-directory references are allowed only within the same archive root; absolute paths, drives, backslashes, control characters, missing files, links and duplicates are rejected. The same member/size limits apply to both passes, plus 4,096 reference pairs and 32 MiB of total metadata. No links are followed or code executed. Attachment also regenerates the reference ledger and rejects manifest tampering.

[Verification against real sources](../evidence/qt-notice-references-2026-09-23.json): same 15 archives/170,920,531 bytes, now **258 notices**, with all previous 247 unchanged. All archive and notice hashes were independently rechecked; the 11 additional texts compared directly against the Qt archive. Collection and separate attachment succeeded in `/tmp/webfence-native-sources-3598b5f-qt-references/`. Updated attachment rejects older Qt collections lacking these files: recollect into a new directory using `--reuse-archive`, without manually changing manifests.

The regression was reproduced with the previous collector omitting an explicitly referenced text. New tests cover order, shared references, malformed metadata, traversal, links, duplicates, budgets and ledger tampering. Local bundle rebuilt from `3598b5f` with declared modifications: 258 notices rechecked, current-inventory binding, Homebrew supplements present, ad hoc signature and Cocoa self-test passed; personal language preserved. 41 local Python tests passed; CI on new code remains to be verified. These notices also include unshipped platforms: coverage of source-tree references does not close binary mapping or legal review.
