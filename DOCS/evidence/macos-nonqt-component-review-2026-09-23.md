# Librerie native non Qt nel bundle macOS / Non-Qt native libraries in the macOS bundle

[Materiali IT](../it/M0-SOURCE-MATERIALS.md) · [Materials EN](../en/M0-SOURCE-MATERIALS.md) · [Dati verificati / Verified data](macos-nonqt-component-review-2026-09-23.json) · [Mappatura Qt / Qt mapping](qt-component-review-2026-09-23.md)

Data / Date: 2026-09-23. **Esame tecnico parziale / Partial technical review.**

**IT:** Il bundle locale esaminato dichiara la revisione `3598b5f` con modifiche locali; non è una release del commit corrente. Su 28 file Mach-O inventariati, nove componenti Qt sono nel registro separato. Qui sono verificati i 19 rimanenti: 18 librerie Homebrew da 14 pacchetti e l'eseguibile Go. Le associazioni sono specifiche per gli hash di questo bundle e del pacchetto sorgenti nativi, non per build future.

**EN:** The reviewed local bundle declares revision `3598b5f` with local changes; it is not a release of the current commit. Of 28 inventoried Mach-O files, nine Qt components are covered by the separate ledger. This review verifies the remaining 19: 18 Homebrew libraries from 14 packages and the Go executable. Associations are specific to this bundle's and native source package's hashes, not future builds.

## File distribuiti / Shipped files

| File nel bundle / Bundle file | Provenienza registrata / Recorded origin |
| --- | --- |
| `Frameworks/libb2.1.dylib` | `libb2@0.98.1` |
| `Frameworks/libdbus-1.3.dylib` | `dbus@1.16.2_1` |
| `Frameworks/libdouble-conversion.3.dylib` | `double-conversion@3.4.0` |
| `Frameworks/libfreetype.6.dylib` | `freetype@2.14.3` |
| `Frameworks/libglib-2.0.0.dylib` | `glib@2.90.0` |
| `Frameworks/libgraphite2.3.dylib` | `graphite2@1.3.15` |
| `Frameworks/libgthread-2.0.0.dylib` | `glib@2.90.0` |
| `Frameworks/libharfbuzz.0.dylib` | `harfbuzz@14.5.0` |
| `Frameworks/libicudata.78.dylib` | `icu4c@78@78.3` |
| `Frameworks/libicui18n.78.dylib` | `icu4c@78@78.3` |
| `Frameworks/libicuuc.78.dylib` | `icu4c@78@78.3` |
| `Frameworks/libintl.8.dylib` | `gettext@1.0` |
| `Frameworks/libjpeg.8.dylib` | `jpeg-turbo@3.2.0` |
| `Frameworks/libmd4c.0.dylib` | `md4c@0.6.0` |
| `Frameworks/libpcre2-16.0.dylib` | `pcre2@10.48` |
| `Frameworks/libpcre2-8.0.dylib` | `pcre2@10.48` |
| `Frameworks/libpng16.16.dylib` | `libpng@1.6.58` |
| `Frameworks/libzstd.1.dylib` | `zstd@1.5.7_1` |
| `MacOS/webfence` | `WebFence Go` |

## Dichiarazioni SBOM dei sorgenti / Source SBOM declarations

**IT:** La colonna segue `SPDXRef-Archive-*` della SBOM Homebrew conservata. `licenseConcluded` è un'espressione aggregata per l'archivio sorgente e **non determina** la licenza dei soli oggetti compilati in ciascuna libreria né seleziona un'alternativa per WebFence. In particolare l'espressione aggregata di gettext unisce GPL e LGPL: non attribuirla automaticamente a `libintl.8.dylib`. La revisione delle condizioni applicabili è ancora aperta.

**EN:** The column follows `SPDXRef-Archive-*` in the retained Homebrew SBOM. `licenseConcluded` is an aggregate expression for the source archive and **does not determine** the license of the objects compiled into each library or select an alternative for WebFence. In particular gettext's aggregate expression combines GPL and LGPL: do not automatically assign it to `libintl.8.dylib`. Review of applicable terms remains open.

| Pacchetto / Package | `licenseConcluded` dell'archivio / Archive `licenseConcluded` | Avvisi sorgente / Source notices |
| --- | --- | ---: |
| `dbus@1.16.2_1` | `AFL-2.1 OR GPL-2.0-or-later` | 12 |
| `double-conversion@3.4.0` | `BSD-3-Clause` | 2 |
| `freetype@2.14.3` | `FTL` | 12 |
| `gettext@1.0` | `GPL-3.0-or-later AND LGPL-2.1-or-later` | 17 |
| `glib@2.90.0` | `LGPL-2.1-or-later` | 14 |
| `graphite2@1.3.15` | `GPL-2.0-or-later OR LGPL-2.1-or-later OR MPL-1.1+` | 4 |
| `harfbuzz@14.5.0` | `MIT` | 6 |
| `icu4c@78@78.3` | `ICU` | 2 |
| `jpeg-turbo@3.2.0` | `IJG AND Zlib AND BSD-3-Clause` | 5 |
| `libb2@0.98.1` | `CC0-1.0` | 1 |
| `libpng@1.6.58` | `libpng-2.0` | 6 |
| `md4c@0.6.0` | `MIT` | 2 |
| `pcre2@10.48` | `BSD-3-Clause` | 4 |
| `zstd@1.5.7_1` | `(BSD-3-Clause OR GPL-2.0-only) AND BSD-2-Clause AND MIT` | 3 |

## Verifiche e limiti / Checks and limits

**IT:** Per ogni libreria, lo SHA-256 del file del keg installato coincide con `source_sha256` dell'inventario. L'hash del file nel bundle dopo la firma coincide con `source-package.json`; la firma ad hoc del bundle è stata verificata con `codesign --verify --deep --strict`. Per ciascuno dei 14 pacchetti, la dichiarazione `SPDXRef-Archive-*` coincide con quella salvata nell'inventario e lega URL/hash SHA-256 all'archivio originale nella raccolta e nello ZIP. Copie SBOM, ricette, metadati e avvisi installati presenti nell'app coincidono con quelle nello ZIP. Tutti i 90 avvisi raccolti dai 14 archivi hanno hash uguali ai manifest e copie uguali in app/ZIP. L'eseguibile Go è legato al solo hash dopo firma e ai metadati di build registrati; il bundle dichiara `vcs.modified=true`.

**EN:** For each library, the installed keg file's SHA-256 matches inventory `source_sha256`. The post-signing bundle file hash matches `source-package.json`; the bundle's ad hoc signature passed `codesign --verify --deep --strict`. For each of the 14 packages, its `SPDXRef-Archive-*` declaration matches the one saved in the inventory and binds URL/SHA-256 to the original archive in the collection and ZIP. SBOM, recipe, metadata and installed-notice copies in the app match their ZIP copies. All 90 notices collected from the 14 archives match their manifest hashes and app/ZIP copies. The Go executable is associated only with its post-signing hash and recorded build metadata; the bundle declares `vcs.modified=true`.

**IT:** Insieme al [registro Qt](qt-component-review-2026-09-23.md), tutti i 28 binari inventariati hanno così una provenienza tecnica registrata. Restano da accertare il codice effettivamente incorporato in ogni libreria, la completezza dei sorgenti e dei diritti per la distribuzione, le librerie di sistema/CGO/Go, le piattaforme Windows e Linux e la compatibilità delle licenze. Un archivio originale e gli avvisi non provano che una bottle sia ricostruibile bit per bit. Nessuna ricetta è stata eseguita, nessun file installato o preferenza è stata modificata. `distribution_ready=false` resta invariato.

**EN:** Together with the [Qt ledger](qt-component-review-2026-09-23.md), all 28 inventoried binaries now have recorded technical provenance. Actual code incorporated in each library, source and distribution-right completeness, system/CGO/Go libraries, Windows and Linux, and license compatibility remain to be established. An original archive and notices do not prove bit-for-bit bottle reproducibility. No recipes were executed and no installed files or preferences were changed. `distribution_ready=false` remains unchanged.

## Approfondimento gettext / Gettext follow-up — 2026-09-24

**IT:** La ricetta Homebrew conservata (SHA-256 `170f5a29d2eb95b9dd1f1f754c0f9b90692c3d5dcf6db02c199c93e2c445177d`) indica `LGPL-2.1-or-later` per `libintl`/`libasprintf` e configura `--with-included-gettext` su macOS. Nell'archivio sorgente verificato, `gettext-runtime/intl/README` identifica quella directory come sorgente GNU libintl (SHA-256 `515acabc5f0d6d5cb0e6316064c2fc7185ebb97a0b6366401eac8014044f4bbf`); `gettext-runtime/intl/COPYING.LIB` contiene LGPL 2.1 (SHA-256 `20e50fe7aae3e56378ebf0417d9de904f55a0e61e4df315333e632a4d3555d95`), e `dcigettext.c` dichiara LGPL 2.1 o successiva (SHA-256 `45207e8ae3149af68a73de267e46ec05a996d6b0ca0fee0ed8f9e56f7e5de6b4`). È evidenza più specifica dell'espressione SBOM aggregata; non prova l'elenco completo degli oggetti nel binario o assolve gli obblighi di distribuzione. Il testo COPYING.LIB è già nei 90 avvisi verificati.

**EN:** The retained Homebrew recipe (SHA-256 `170f5a29d2eb95b9dd1f1f754c0f9b90692c3d5dcf6db02c199c93e2c445177d`) identifies `LGPL-2.1-or-later` for `libintl`/`libasprintf` and configures `--with-included-gettext` on macOS. In the verified source archive, `gettext-runtime/intl/README` identifies that directory as GNU libintl sources (SHA-256 `515acabc5f0d6d5cb0e6316064c2fc7185ebb97a0b6366401eac8014044f4bbf`); `gettext-runtime/intl/COPYING.LIB` contains LGPL 2.1 (SHA-256 `20e50fe7aae3e56378ebf0417d9de904f55a0e61e4df315333e632a4d3555d95`), and `dcigettext.c` declares LGPL 2.1 or later (SHA-256 `45207e8ae3149af68a73de267e46ec05a996d6b0ca0fee0ed8f9e56f7e5de6b4`). This is more specific than the aggregate SBOM expression; it does not establish every object in the binary or discharge distribution obligations. COPYING.LIB is already among the 90 verified notices.
