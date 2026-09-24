# M0 — Bundle macOS corrente e materiali associati / Current macOS bundle and associated materials

[Matrice IT](../it/M0-VALIDATION.md) · [Matrix EN](../en/M0-VALIDATION.md) · [Packaging IT](../it/M0-PACKAGING.md) · [Packaging EN](../en/M0-PACKAGING.md)

## Italiano

Il 2026-09-24 è stato ricostruito `dist/WebFence.app` su macOS 26.6.2 Apple Silicon dal commit pulito `d92c20e10193a838e793483671c8762d3cef182e` (`go version -m`: `vcs.modified=false`, Go 1.27.1, Qt 6.11.2). La build ha usato l'archivio Qt già raccolto, verificato per SHA-256, gli stessi 15 archivi upstream verificati e il piano dei supplementi Homebrew revisionato. Nessun file sorgente del repository è stato modificato durante la build. Il bundle storico `3598b5f` è stato sostituito soltanto dopo i controlli dello staging previsti dallo script.

| Controllo | Esito |
| --- | --- |
| `sh scripts/package-macos.sh`, con `WEBFENCE_QT_SOURCE_ARCHIVE`, `WEBFENCE_NATIVE_SOURCE_MATERIALS` e `WEBFENCE_NATIVE_SUPPLEMENT_PLAN` | Passato; 15 archivi upstream verificati nella raccolta separata, 258 avvisi e quattro supplementi allegati al bundle |
| Firma ad hoc e chiusura Mach-O | `codesign --verify --deep --strict` passato; 28 binari, 194 riferimenti, 148 Apple e 46 interni all'app |
| Minimo incorporato e dichiarato | 26.0.0 per questo bundle; non è il minimo ufficiale scelto per il prodotto |
| Pacchetto `dist/WebFence-native-sources-d92c20e.zip` | 173.395.432 byte; `unzip -tq` passato; manifest con 15 archivi, 258 avvisi, quattro supplementi e hash dei 28 binari firmati ricontrollati contro l'app corrente |
| Soak Cocoa finale, PATH di solo sistema | Quattro cicli completi in 12,003 s, uscita zero, fixture sintetiche |
| Self-test Cocoa locale | **Non superato**: controlli dei dati, righe, filtri e layout passano; le asserzioni di attivazione/focus falliscono mentre il Mac risulta bloccato. La CI macOS 15 dello stesso codice aveva superato il self-test. Non attribuire la causa alla sola schermata bloccata senza ricontrollo. |
| AX/VoiceOver sul bundle corrente | Non eseguiti: il Mac è bloccato; M0-01 resta aperto. |

SHA-256: eseguibile `246fce7a0081059b29e31891749603bd97df3908e128577c9b2892e2f3b1c526`; plugin Cocoa `c76b858274c405f2e7b6024aac9678024c77402798bae6768214e86c86adfc3a`; ZIP sorgenti `859afac531d95fba10d57c28896a06c8e3fe5f8156bf31bdf66adad86d100617`. Il vecchio `dist/WebFence-native-sources-local.zip` è un artefatto storico **non associato** al nuovo bundle; usare solo il file `-d92c20e.zip` per questo confronto. Nessuno dei due è una release.

La compilazione del plugin ha emesso l'avviso Qt che l'SDK Apple 27.0 in uso non è fra quelli provati da Qt 6.11.2 (fino a 26); la compilazione, firma e le prove sopra sono passate, ma l'avviso è un limite da riesaminare nella matrice di supporto. Sorgenti corrispondenti completi, contenuto incorporato, collaudo su Mac pulito, minimo ufficiale e revisione legale restano aperti.

## English

On 2026-09-24, `dist/WebFence.app` was rebuilt on Apple Silicon macOS 26.6.2 from clean commit `d92c20e10193a838e793483671c8762d3cef182e` (`go version -m`: `vcs.modified=false`, Go 1.27.1, Qt 6.11.2). The build reused the SHA-256-verified Qt archive, the same 15 verified upstream archives and the reviewed Homebrew supplement plan. No repository source file changed during the build. The historical `3598b5f` bundle was replaced only after the script's staging checks.

| Check | Result |
| --- | --- |
| `sh scripts/package-macos.sh` with `WEBFENCE_QT_SOURCE_ARCHIVE`, `WEBFENCE_NATIVE_SOURCE_MATERIALS` and `WEBFENCE_NATIVE_SUPPLEMENT_PLAN` | Passed; 15 upstream archives verified in the separate collection, 258 notices and four supplements attached to the bundle |
| Ad hoc signature and Mach-O closure | `codesign --verify --deep --strict` passed; 28 binaries, 194 references, 148 Apple and 46 inside the app |
| Embedded and declared minimum | 26.0.0 for this bundle; not the product's chosen official minimum |
| `dist/WebFence-native-sources-d92c20e.zip` package | 173,395,432 bytes; `unzip -tq` passed; manifest with 15 archives, 258 notices, four supplements and hashes of 28 signed binaries rechecked against the current app |
| Final Cocoa soak with system-only PATH | Four full cycles in 12.003 s, exit zero, synthetic fixtures |
| Local Cocoa self-test | **Did not pass**: data, row, filter and layout checks pass; activation/focus assertions fail while the Mac is locked. macOS 15 CI for the same code passed the self-test. Do not attribute the cause solely to the lock without a recheck. |
| AX/VoiceOver on the current bundle | Not run: the Mac is locked; M0-01 remains open. |

SHA-256: executable `246fce7a0081059b29e31891749603bd97df3908e128577c9b2892e2f3b1c526`; Cocoa plugin `c76b858274c405f2e7b6024aac9678024c77402798bae6768214e86c86adfc3a`; source ZIP `859afac531d95fba10d57c28896a06c8e3fe5f8156bf31bdf66adad86d100617`. The older `dist/WebFence-native-sources-local.zip` is a historical artifact **not associated** with this new bundle; use only the `-d92c20e.zip` file for this comparison. Neither is a release.

Building the plugin emitted Qt's warning that the Apple SDK 27.0 in use is beyond those tested with Qt 6.11.2 (through 26); compilation, signing and the checks above passed, but this warning remains a support-matrix consideration. Complete corresponding sources, embedded contents, clean-Mac trial, official minimum and legal review remain open.
