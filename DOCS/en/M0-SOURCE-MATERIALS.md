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

The regression was reproduced with the previous collector omitting an explicitly referenced text. New tests cover order, shared references, malformed metadata, traversal, links, duplicates, budgets and ledger tampering. Local bundle rebuilt from `3598b5f` with declared modifications: 258 notices rechecked, current-inventory binding, Homebrew supplements present, ad hoc signature and Cocoa self-test passed; personal language preserved. 41 local Python tests passed; [CI `98478f2`](https://github.com/Matte2599/WebFence/actions/runs/35918855381) completed: six passing jobs and 41 Python regressions on four native targets. macOS bundle/Cocoa/plugin replacement, Windows ZIP with sources and preservation on failure, and Debian amd64/arm64 packages passed. The actual 258 notices are verified in the local bundle; macOS bundle CI still does not acquire the full optional collection. These notices also include unshipped platforms: coverage of source-tree references does not close binary mapping or legal review.

## macOS source package accompanying the bundle

After collecting sources and preparing a bundle with Homebrew supplements, from the project checkout:

```sh
python3 scripts/package-macos-sources.py \
  dist/WebFence.app /tmp/webfence-source-materials \
  dist/WebFence-native-sources.zip
shasum -a 256 dist/WebFence-native-sources.zip
```

The source directory must be the updated 258-notice collection described above. The ZIP destination must be new and outside the app and collection. The script performs no downloads, executes no recipes and does not modify the bundle. It requires macOS to verify signing with `codesign --verify --deep --strict` before and after assembly. Inputs are trusted materials in builder-controlled directories without concurrent writers.

The ZIP contains `WebFence-native-sources/notices/`: the app's notices and IT/EN documents, installed recipes/SBOMs, Cocoa patch and tools, all original upstream archives under `upstream-source/archives/`, 258 regenerated notices and Homebrew supplements regenerated from verified inputs. Archive paths in the copied manifest now resolve inside the ZIP. Loose files from the previous collection do not replace texts inside the verified archive.

`source-package.json` records SHA-256 hashes of included files, the current native inventory, Go provenance and **post-signing** hashes of binaries listed in the app inventory. These associate materials with the specific bundle; ad hoc signing and hashes do not authenticate the publisher. The manifest retains `distribution_ready=false` and `corresponding_sources_complete=false`: it contains available native materials, not all WebFence/Go source code or a complete product SBOM.

Limits: at most 10,000 files/128 MiB of notices copied from the bundle, in addition to the documented source/supplement limits. Non-regular files and links in notices are rejected. ZIP/CRCs and every hash are read back before publication. Output is published with an atomic hard link from staging on the same filesystem; a filesystem without that capability returns an error, without a fallback that overwrites existing files.

Synthetic tests cover content and app binding, input preservation, existing output, paths inside inputs, incomplete collection, altered patch, signature failure, concurrent binary changes, budgets and links. Cross-platform tests simulate `codesign`; the native trial is recorded separately. Remaining build resources/environments, embedded-component mapping, other-platform procedures and legal review remain open.

[Local trial evidence](../evidence/macos-source-package-2026-09-23.json). Separate macOS native-material package: `scripts/package-macos-sources.py` verifies the app signature, regenerates notices/supplements, includes original archives and records post-signing hashes to associate the bundle. No downloads or app modification; ZIP read back and published without overwriting. Actual trial: 173,236,303 bytes, 15 archives, 258 notices, four supplements, 464 material files and 28 associated binaries; macOS unzip and existing-ZIP preservation passed. 46 local Python tests passed. Local output dist/WebFence-native-sources-local.zip; evidence DOCS/evidence/macos-source-package-2026-09-23.json. [CI `f46768d`](https://github.com/Matte2599/WebFence/actions/runs/35920813028) completed: six passing jobs and 46 Python tests on four native targets. The macOS runner actually downloads all 15 archives, regenerates 258 notices and four supplements, creates a 173,243,954-byte ZIP associated with 28 signed binaries and passes unzip; Cocoa and plugin replacement remain green. Windows and Debian amd64/arm64 pass their respective trials. Available materials assembled, not complete corresponding sources or legal approval; Further resources, mapping and external gates remain open.

WebFence code remains in the repository. Its exclusion, together with Go toolchain sources, from the native-material ZIP defines the format: it does not introduce a new M0 exit criterion or an established legal obligation. Manifest completeness items must be read in that context.

[Recipe review ledger](../evidence/homebrew-recipe-review-2026-09-23.md). Technical review of the 15 Homebrew recipes included in the native package: no other explicit patch/resource download identified for the stable macOS build beyond already collected GLib/libb2 supplements; inline D-Bus patch present. Graphite2/HarfBuzz fonts and JPEG are test fixtures; additional libpng PNG is Linux-only. Hashes and copies of all 15 recipes plus Cocoa builder/patch rechecked against the ZIP. Bilingual ledger in DOCS/evidence/homebrew-recipe-review-2026-09-23.md; legal dossier updated to actually available materials. Does not prove whole-environment rebuilding, embedded mapping or legal compatibility. Code unchanged from f46768d, CI run 35920813028 already passed; M0 remains open.

[Verified Qt mapping](../evidence/qt-component-review-2026-09-23.md). Qt mapping for the macOS bundle: nine binaries associated by path with retained SBOM entities; Cocoa treated as upstream baseline plus WebFence patch. 62 packages reachable through DEPENDS_ON, including 32 attributions, not proof of actual inclusion. Verified 27 metadata files and 22 texts against Qt archive, bundle and ZIP; all LicenseId values match. Rechecked nine post-signing binary hashes and bundle signature. Five of eight upstream SHA-1 checksums differ from the keg: no binary identity inferred from the SBOM. Partial technical review, no legal selection/approval; details and limits in the bilingual ledger.

[Review of non-Qt libraries](../evidence/macos-nonqt-component-review-2026-09-23.md). Verified technical provenance for the 19 non-Qt Mach-O files in the local macOS bundle: 18 libraries from 14 Homebrew packages plus the Go executable. Together with nine mapped Qt files, all 28 inventoried files have a technical association for this build. Library/keg hashes, 14 original archives, recipes/SBOMs and 90 source notices were compared with the app and ZIP; ad hoc signature valid. SBOM expressions cover source archives and do not automatically assign licenses to individual binaries. Source completeness, embedded code and legal review remain open.

[Mach-O reference closure](../evidence/macos-linkage-closure-2026-09-24.md). The new check removes external rpaths in staging before inventory: hashes in new bundles refer to already cleaned binaries; earlier collections still refer to their own historical artifact.

## GNU mirror fallback (2026-09-24)

[CI on `928687d`](https://github.com/Matte2599/WebFence/actions/runs/35968717828) stopped only the macOS 26 job during collection: the inventory URL `https://ftpmirror.gnu.org/gnu/gettext/gettext-1.0.tar.gz` returned 404. The same archive at `https://ftp.gnu.org/gnu/gettext/gettext-1.0.tar.gz` returned 200; the local 32,694,085-byte download matched the pinned SHA-256 `85d99b79c981a404874c02e0342176cf75c7698e2b51fe41031cf6526d974f1a`. The collector retries **only** URLs under `ftpmirror.gnu.org/gnu/` at the official `ftp.gnu.org` host, with the same HTTPS, time, size and checksum limits. Its manifest preserves the inventory URL and records the actual `acquisition_url`. [CI `e2cce25`](https://github.com/Matte2599/WebFence/actions/runs/35975333332) verifies full collection on macOS 15 and 26; it does not establish whether that run actually used the fallback; this does not close the material or legal review.
