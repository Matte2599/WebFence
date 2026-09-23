# Homebrew recipe review / Revisione ricette Homebrew — 2026-09-23

[Materiali IT](../it/M0-SOURCE-MATERIALS.md) · [Materials EN](../en/M0-SOURCE-MATERIALS.md) · [Source-package identity / Identità pacchetto](macos-source-package-2026-09-23.json)

**IT:** Esame manuale delle 15 ricette conservate nel bundle locale e nello ZIP nativo, non esecuzione Ruby. Ogni ricetta è stata confrontata byte per byte con la copia nello ZIP; confrontati anche builder e patch Cocoa. Il perimetro è la build **stabile macOS ARM64** descritta da queste ricette. Non sono state ricompilate tutte le dipendenze o certificata la corrispondenza binaria delle bottle.

**EN:** Manual review of the 15 recipes retained in the local bundle and native ZIP, without executing Ruby. Each recipe was compared byte for byte with its ZIP copy; Cocoa builder and patch were also compared. Scope is the **stable macOS ARM64** build described by these recipes. Not all dependencies were rebuilt and bottle binary reproducibility was not certified.

Le ricette sono sotto `WebFence-native-sources/notices/homebrew/PACKAGE/.brew/FORMULA.rb` nello ZIP. / Recipes are under that path inside the ZIP.

| Package | Recipe SHA-256 | Esito / Finding |
| --- | --- | --- |
| `dbus@1.16.2_1` | `0ccf3ea6dd594140eaeaecc7dbd5810af6a6ca94885311501cd4bdabd5f843df` | Patch `:DATA` nel file; sostituzione plist inline. / Patch `:DATA` in the file; inline plist replacement. |
| `double-conversion@3.4.0` | `e5fa484ea27ae32a1dd2ecab92a2dcde38f00ffa47b073310ff1768ceba501c5` | Nessuna patch/risorsa aggiuntiva dichiarata. / No additional patch/resource declared. |
| `freetype@2.14.3` | `cc500226718264185ecf4b0560e0d9daee933c3e40622dd936b955536d025613` | Sostituzioni inline per configurazione/pkg-config; nessuna risorsa aggiuntiva. / Inline config/pkg-config substitutions; no additional resource. |
| `gettext@1.0` | `170f5a29d2eb95b9dd1f1f754c0f9b90692c3d5dcf6db02c199c93e2c445177d` | Opzioni configure e ambiente inline; nessuna risorsa aggiuntiva dichiarata. / Inline configure/environment options; no additional resource declared. |
| `glib@2.90.0` | `eb964bd9d934f92a1e271e84085c58541770d41f5698e7e5f17094375bb46d2a` | Patch percorsi + gobject-introspection già inclusi nei supplementi; sostituzioni inline. / Path patch + gobject-introspection already included in supplements; inline substitutions. |
| `graphite2@1.3.15` | `dc60f1784330fbf36c320a92dbf31b0f4f5ed83bab6b252963cdade7b87f6d49` | Font `testfont` soltanto in `test do`; non richiesto da `install`. / `testfont` only in `test do`; not required by `install`. |
| `harfbuzz@14.5.0` | `4a01ef42e17002fbe8c4ae20690b972bbf2a346512ba91e076411b2e7ddf65ec` | Font `homebrew-test-ttf` soltanto in `test do`; build con test disabilitati. / `homebrew-test-ttf` only in `test do`; build disables tests. |
| `icu4c@78@78.3` | `db6d475bc93d9138f6ec7272dcc2718614e3df68dc0edeb2ab63054493eec04b` | Sostituzioni pkg-config inline; nessuna risorsa aggiuntiva. / Inline pkg-config substitutions; no additional resource. |
| `jpeg-turbo@3.2.0` | `b1937813c7c67fa8a44a7a958f1496e9e237d942052b52d421c41e61df4b526b` | Fixture JPEG di Homebrew nel test della formula; nessun download risorsa nella build ARM64. / Homebrew JPEG fixture in formula test; no resource download in ARM64 build. |
| `libb2@0.98.1` | `35179ede9f32d4cb2ab8ea26eb08735e3cdbbbfa1537177bddc66a7b84b517cd` | Patch configure Big Sur già inclusa e vincolata alla ricetta. / Big Sur configure patch already included and recipe-bound. |
| `libpng@1.6.58` | `c125a77d8263e0fb215f8f5d693c53324caabb63801df7093d9f3c8e42f4dd22` | `pngtest.png` aggiuntivo dichiarato e usato solo per Linux. / Additional `pngtest.png` declared and used only for Linux. |
| `md4c@0.6.0` | `74346a540eeacdd467de96bb48067d5d5ce00b635b82ad26d85c03d6ad019a0a` | Nessuna patch/risorsa aggiuntiva dichiarata. / No additional patch/resource declared. |
| `pcre2@10.48` | `493c6e81fa41064d8cdbb554f0884a9afe856f11c62d250ed747ce5f565a3f0d` | Nessuna patch/risorsa aggiuntiva stabile; autogen solo nel ramo HEAD. / No additional stable patch/resource; autogen only in HEAD branch. |
| `qtbase@6.11.2` | `4495dddcc98a75181a12b80e40a830bc6b5e6d9d8d3d578115d8dffee2418c18` | Rimozioni third-party e sostituzioni CMake inline; nessun download aggiuntivo di patch/risorse. Correzione Cocoa WebFence separata. / Inline third-party removals and CMake substitutions; no extra patch/resource download. WebFence Cocoa correction is separate. |
| `zstd@1.5.7_1` | `9a9b0183896886e82b915af4db576f27d34d1cee7757d6c5beb0e1eb74c68e0d` | Opzioni CMake e sostituzione pkg-config inline; nessuna risorsa aggiuntiva. / Inline CMake options and pkg-config substitution; no additional resource. |

## Esito / Outcome

**IT:** Nessun altro download esplicito di patch/risorse applicabile alla build stabile macOS è stato individuato nelle ricette esaminate oltre a GLib/libb2 già raccolti. La patch D-Bus è inclusa nel testo della ricetta. I font Graphite2/HarfBuzz e la fixture JPEG servono ai test delle formule; il PNG aggiuntivo libpng è limitato al ramo Linux. Non sono automaticamente materiali incorporati nell’app. Non sono stati scaricati o eseguiti tali test, né attivati servizi D-Bus.

**EN:** No other explicit patch/resource download applicable to the stable macOS build was identified in the reviewed recipes beyond already collected GLib/libb2 inputs. The D-Bus patch is embedded in its recipe. Graphite2/HarfBuzz fonts and the JPEG fixture serve formula tests; libpng's additional PNG is limited to the Linux branch. They are not automatically materials embedded in the app. Those tests were not downloaded or executed, and no D-Bus services were activated.

**IT:** `depends_on`, `uses_from_macos`, macro Homebrew (`std_cmake_args`, `std_meson_args`, ecc.), SDK e compilatori descrivono anche l’ambiente necessario a ricostruire una formula intera. La loro ricostruzione non è dimostrata da questo esame. Analogamente, `rm_r` e opzioni system-* nella ricetta Qt spiegano perché l’albero sorgente e gli avvisi raccolti sono un insieme più ampio del codice effettivamente incorporato. Mappatura dei componenti incorporati, verifica delle licenze applicabili e chiusura dei materiali corrispondenti restano da completare.

**EN:** `depends_on`, `uses_from_macos`, Homebrew macros (`std_cmake_args`, `std_meson_args`, etc.), SDKs and compilers also describe the environment needed to rebuild a whole formula. This review does not demonstrate rebuilding that environment. Similarly, `rm_r` and system-* options in the Qt recipe explain why the collected source tree and notices exceed actually embedded code. Embedded-component mapping, applicable-license review and corresponding-material closure remain incomplete.

**IT:** Il riesame vale per gli hash sopra, non per qualunque versione futura o per tutte le bottle macOS. La variante libb2 Sequoia è già confrontata nella [verifica separata](macos-sources-2026-09-23/sequoia-recipe-check.json). Questo registro riduce l’incertezza sulle risorse dichiarate; non è approvazione legale, SBOM completa o chiusura M0-02/M0-06.

**EN:** Review applies to the hashes above, not arbitrary future versions or all macOS bottles. The Sequoia libb2 variant was already compared in a [separate check](macos-sources-2026-09-23/sequoia-recipe-check.json). This ledger narrows uncertainty about declared resources; it is not legal approval, a complete SBOM or M0-02/M0-06 closure.
