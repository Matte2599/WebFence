# Componenti Qt nel bundle macOS / Qt components in the macOS bundle

[Materiali IT](../it/M0-SOURCE-MATERIALS.md) · [Materials EN](../en/M0-SOURCE-MATERIALS.md) · [Dati e hash / Data and hashes](qt-component-review-2026-09-23.json)

Data / Date: 2026-09-23. **Esame tecnico parziale / Partial technical review.**

## Perimetro / Scope

**IT:** Il bundle esaminato dichiara `3598b5f` con modifiche locali, come già registrato nella [prova del pacchetto sorgenti](macos-source-package-2026-09-23.json). Non è una build del commit corrente. Nove binari Qt sono collegati per percorso alle entità della SBOM Qt 6.11.2 conservata nel bundle. Per Cocoa, ricompilato con la patch WebFence, l'entità upstream è soltanto la base di riferimento. Questa verifica non seleziona una licenza, non sostituisce il parere professionale e non chiude M0-02/M0-06.

**EN:** The reviewed bundle declares `3598b5f` with local modifications, as already recorded in the [source-package trial](macos-source-package-2026-09-23.json). It is not a build of the current commit. Nine Qt binaries are associated by path with entities in the Qt 6.11.2 SBOM retained in the bundle. For Cocoa, rebuilt with the WebFence patch, the upstream entity is only a baseline. This review does not select a license, replace professional review or close M0-02/M0-06.

## Componenti distribuiti / Shipped components

| Percorso relativo a Contents / Path relative to Contents | Entità upstream / Upstream entity |
| --- | --- |
| `Frameworks/QtCore.framework/Versions/A/QtCore` | `Core` |
| `Frameworks/QtDBus.framework/Versions/A/QtDBus` | `DBus` |
| `Frameworks/QtGui.framework/Versions/A/QtGui` | `Gui` |
| `Frameworks/QtWidgets.framework/Versions/A/QtWidgets` | `Widgets` |
| `PlugIns/imageformats/libqgif.dylib` | `QGifPlugin` |
| `PlugIns/imageformats/libqico.dylib` | `QICOPlugin` |
| `PlugIns/imageformats/libqjpeg.dylib` | `QJpegPlugin` |
| `PlugIns/platforms/libqcocoa.dylib` | `QCocoaIntegrationPlugin` (base / baseline) |
| `PlugIns/styles/libqmacstyle.dylib` | `QMacStylePlugin` |

**IT:** Tutte e nove le entità riportano nell'upstream `PackageLicenseConcluded` l'espressione `LicenseRef-Qt-Commercial OR LGPL-3.0-only OR GPL-2.0-only OR GPL-3.0-only`. È una dichiarazione di Qt, non una conclusione legale di WebFence né una concessione commerciale acquisita. Le attribuzioni di terzi hanno espressioni proprie, riportate sotto.

**EN:** All nine entities have the upstream `PackageLicenseConcluded` expression `LicenseRef-Qt-Commercial OR LGPL-3.0-only OR GPL-2.0-only OR GPL-3.0-only`. This is a Qt declaration, not a WebFence legal conclusion or an acquired commercial grant. Third-party attributions have their own expressions, listed below.

## Attribuzioni raggiungibili / Reachable attributions

**IT:** Seguendo soltanto `DEPENDS_ON` dalle nove entità si raggiungono 62 pacchetti, inclusi 32 nodi di attribuzione. Questo conteggio **non dimostra 32 componenti incorporati**. I commenti upstream includono, per esempio, moduli CMake di build, header OpenGL descritti per Windows/Linux e un generatore mipmap Direct3D 12. Apache Tika è descritto come fallback se manca shared-mime-info. Non si può dedurre l'esecuzione o l'inclusione macOS dalla sola relazione; gli avvisi raccolti restano un insieme conservativo e non sono rimossi.

**EN:** Following only `DEPENDS_ON` from the nine entities reaches 62 packages, including 32 attribution nodes. This count **does not prove 32 embedded components**. Upstream comments include, for example, build-time CMake modules, OpenGL headers described for Windows/Linux and a Direct3D 12 mipmap generator. Apache Tika is described as a fallback when shared-mime-info is absent. A relationship alone does not establish execution or macOS inclusion; collected notices remain a conservative superset and are not removed.

| Nodo upstream / Upstream node | Espressione dichiarata / Declared expression |
| --- | --- |
| `Core_Attribution_blake2` | `CC0-1.0 OR Apache-2.0` |
| `Core_Attribution_easing` | `BSD-3-Clause` |
| `Core_Attribution_forkfd` | `MIT` |
| `Core_Attribution_md4` | `CC0-1.0` |
| `Core_Attribution_md5` | `CC0-1.0` |
| `Core_Attribution_qeventdispatcher_cf` | `BSD-3-Clause` |
| `Core_Attribution_rfc6234` | `BSD-3-Clause` |
| `Core_Attribution_sha1` | `LicenseRef-SHA1-Public-Domain` |
| `Core_Attribution_sha3_endian` | `BSD-2-Clause` |
| `Core_Attribution_sha3_keccak` | `CC0-1.0` |
| `Core_Attribution_siphash` | `CC0-1.0` |
| `Core_Attribution_tika-mimetypes` | `Apache-2.0` |
| `Core_Attribution_tinycbor` | `MIT` |
| `Core_Attribution_tlexpected` | `CC0-1.0` |
| `Core_Attribution_unicode-character-database` | `Unicode-3.0` |
| `Core_Attribution_unicode-cldr` | `Unicode-3.0` |
| `DBus_Attribution_libdbus-1-headers` | `AFL-2.1 OR GPL-2.0-or-later` |
| `Gui_Attribution_aglfn` | `BSD-3-Clause` |
| `Gui_Attribution_emoji-segmenter` | `Apache-2.0` |
| `Gui_Attribution_grayraster` | `FTL OR GPL-2.0-only` |
| `Gui_Attribution_icc-srgb-color-profile` | `LicenseRef-ICC-License` |
| `Gui_Attribution_opengl-es2-headers` | `MIT` |
| `Gui_Attribution_opengl-headers` | `MIT` |
| `Gui_Attribution_rhi-miniengine-d3d12-mipmap` | `MIT` |
| `Gui_Attribution_smooth-scaling-algorithm` | `BSD-2-Clause AND Imlib2` |
| `Gui_Attribution_vulkan-xml-spec` | `Apache-2.0 OR MIT` |
| `Gui_Attribution_vulkanmemoryallocator` | `MIT` |
| `Gui_Attribution_webgradients` | `MIT` |
| `Gui_Attribution_xserverhelper` | `X11 AND HPND` |
| `Platform_Attribution_extra-cmake-modules` | `BSD-3-Clause` |
| `Platform_Attribution_kwin` | `BSD-3-Clause` |
| `QCocoaIntegrationPlugin_Attribution_cocoa-platform-plugin` | `BSD-3-Clause` |

## Verifica e limiti / Verification and limits

**IT:** Sono stati riletti 27 file `qt_attribution.json` dall'archivio Qt verificato per SHA-256: tutti i 32 `LicenseId` coincidono con le rispettive espressioni nella SBOM. I 23 riferimenti LicenseFile/LicenseFiles portano a 22 testi distinti. I 49 membri complessivi (27 metadati e 22 testi) sono stati confrontati con una seconda lettura dell'archivio; copie nel bundle e nello ZIP identiche. L'assenza di LicenseFile in una voce non significa assenza di obblighi o di avvisi nel sorgente.

**EN:** Twenty-seven `qt_attribution.json` files were reread from the SHA-256-verified Qt archive: all 32 `LicenseId` values match their SBOM expressions. The 23 LicenseFile/LicenseFiles references resolve to 22 distinct texts. All 49 members (27 metadata files and 22 texts) were checked against a second archive read; bundle and ZIP copies are identical. An entry without LicenseFile does not imply absence of obligations or source notices.

**IT:** Ricontrollati l'hash dell'inventario, la copia SBOM nello ZIP e i nove hash binari dopo la firma registrati nel pacchetto sorgenti. Firma del bundle verificata con `codesign --verify --deep --strict`; builder, patch e CMake Cocoa identici alle copie nello ZIP. Per gli otto binari provenienti dal keg, lo SHA-256 installato coincide con quello nell'inventario. Lo SHA-1 upstream coincide con il file installato soltanto per QGifPlugin, QICOPlugin e QMacStylePlugin; differisce per Core, DBus, Gui, Widgets e QJpegPlugin. La causa delle differenze non è dimostrata da questo esame. La mappatura usa percorsi e provenienza registrata: non presenta gli hash upstream come verifica d'integrità del bundle modificato e firmato.

**EN:** Rechecked inventory hash, the ZIP's SBOM copy and all nine post-signing binary hashes recorded in the source package. Bundle signature verified with `codesign --verify --deep --strict`; Cocoa builder, patch and CMake match their ZIP copies. For the eight keg-derived binaries, installed SHA-256 matches the inventory. Upstream SHA-1 matches the installed file only for QGifPlugin, QICOPlugin and QMacStylePlugin; it differs for Core, DBus, Gui, Widgets and QJpegPlugin. This review does not establish the cause of those differences. Mapping uses paths and recorded provenance: it does not present upstream hashes as integrity verification of the modified and signed bundle.

**IT:** Metodo ripetibile: leggere i record PackageName/FileName/SPDXID della SBOM conservando i campi multilinea; associare i source_path dell'inventario tramite CONTAINS; usare il percorso upstream Cocoa solo come baseline; visitare DEPENDS_ON senza duplicati; leggere percorso e indice qt_attribution.json indicati nei commenti; confrontare LicenseId, risolvere i riferimenti relativi entro la radice Qt e confrontare i byte con avvisi e ZIP. Il JSON conserva identificatori, percorsi, espressioni, hash, relazioni con le radici e limiti. Non è un nuovo verificatore SPDX di produzione. Non sono state eseguite ricette o modificate app, Qt installato o preferenze.

**EN:** Reproducible method: read PackageName/FileName/SPDXID records while preserving multiline SBOM fields; associate inventory source_path values through CONTAINS; use the upstream Cocoa path only as a baseline; traverse DEPENDS_ON without duplicates; read the qt_attribution.json path and entry index in comments; compare LicenseId, resolve references within the Qt root and compare bytes with notices and ZIP. The JSON retains identifiers, paths, expressions, hashes, root reachability and limitations. This is not a new production SPDX validator. No recipes were executed and no app, installed Qt or preferences were changed.

**IT:** Restano fuori da questa prova: inventario completo del codice incorporato, selezione/compatibilità delle licenze, altre librerie native, Windows/Linux e ricostruzione di tutti i binari. I relativi gate restano aperti; questa evidenza riduce la mappatura Qt a componenti identificabili dell'app concreta.

**EN:** Outside this trial: complete embedded-code inventory, license selection/compatibility, other native libraries, Windows/Linux and rebuilding all binaries. Those gates remain open; this evidence ties the Qt mapping to identifiable components of the actual app.

## Verifica CI / CI verification

**IT:** [CI `f345a1f`](https://github.com/Matte2599/WebFence/actions/runs/35923685155) conclusa con sei job verdi, tentativo 2: il primo job macOS era fallito per timeout DNS scaricando D-Bus, dopo test GUI/Cocoa e sostituzione plugin riusciti. Ritentato soltanto macOS, senza modifiche al codice; raccolta finale 15 archivi/258 avvisi/quattro supplementi/28 binari associati superata. Gli altri cinque job erano già verdi; 46 regressioni Python sui quattro target nativi e pacchetti Windows/Debian superati. Documentazione della build: 76 Markdown/629 collegamenti locali. Matrice aggiornata per richiamare il collaudo Cocoa già completato di oltre 30 minuti, distinto dalla prova con lettore reale. M0 resta aperta.

**EN:** [CI `f345a1f`](https://github.com/Matte2599/WebFence/actions/runs/35923685155) completed with six passing jobs, attempt 2: the first macOS job failed on a DNS timeout while downloading D-Bus, after passing GUI/Cocoa and plugin replacement trials. Retried macOS only, without code changes; final collection of 15 archives/258 notices/four supplements/28 associated binaries passed. The other five jobs had already passed; 46 Python regressions on four native targets and Windows/Debian packages passed. Built documentation: 76 Markdown files/629 local links. Matrix updated to reference the already completed Cocoa trial exceeding 30 minutes, separately from actual reader testing. M0 remains open.
