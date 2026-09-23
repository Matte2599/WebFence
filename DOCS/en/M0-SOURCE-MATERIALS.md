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

Whole-source-tree notices can cover unshipped components; map them to actual binaries and choose/verify applicable terms. Recipes, SBOMs and the Cocoa patch remain in the original bundle identified by the inventory. Collected files are not automatically incorporated into the package or published as a release. Legal review, minimum OS versions, clean desktop trials and Windows materials remain part of M0-02/M0-06; collecting files does not approve them.

## Identified GLib supplement

The [Homebrew recipe at revision 550d1a4](https://github.com/Homebrew/homebrew-core/blob/550d1a4e5c2da2dd3c8c630e43b1ef9a84afd700/Formula/g/glib.rb) matches the installed recipe byte for byte after removing only its `bottle` stanza. Acquired `Patches/glib/hardcoded-paths.diff` from the same Git tree and the gobject-introspection 1.86.0 archive, with SHA-256 verified against the recipe. Patch dry-run on verified GLib 2.90.0 source passes without fuzz; no installation, code execution or keg modification.

[Provenance and hashes](../evidence/macos-sources-2026-09-23/glib-supplement.json) retained; materials in `/tmp/webfence-glib-materials-550d1a4/`, separate from automated output. Matching identifies recipe materials without proving a bit-identical bottle rebuild. Integration into the complete package and verification of other recipes remain open.

The libb2 0.98.1 Big Sur configure patch was also acquired from the recipe’s pinned URL, SHA-256 checked and tested with a no-fuzz dry-run: [ledger](../evidence/macos-sources-2026-09-23/libb2-supplement.json), local file `/tmp/webfence-libb2-materials/configure-big_sur.diff`. The D-Bus patch is already embedded in its retained recipe. Other recipes contain textual substitutions and test resources (fonts/images): do not automatically equate them with shipped content; retain recipes and verify each material’s role during assembly.
