# M0 — Ricognizione del packaging

[English](../en/M0-PACKAGING.md) · [Verifica](M0-VALIDATION.md) · [Indice](../README.md)

Data: 2026-09-23. Solo artefatti di sviluppo; nessuna release supportata o revisione della distribuzione completata.

## Artefatto macOS esaminato

Il `dist/WebFence.app` locale esistente è un bundle di sviluppo, non il commit corrente del repository. `go version -m` identifica la revisione `c4ce9cab56635db726e590ad0c2c09f20d0f18ee` con `vcs.modified=true`, Go 1.27.1 e MIQT 0.14.0. SHA-256 dell’eseguibile principale: `de663edcda2eceddafb5e020a9b944949371f1bb20c569178d7f03c6a4048436`. Identifica il binario esaminato, non un manifest di release firmato.

L’ispezione in sola lettura di tutti i Mach-O, escludendo duplicati symlink, ha trovato **28 binari/librerie/plugin**. `otool -l` indica macOS minimo **14.0 per 9** e **26.0 per 19**, compreso l’eseguibile principale. QtCore dichiara 14.0; glib e ICU nel bundle dichiarano 26.0. Questo artefatto richiede quindi macOS 26, nonostante il supporto upstream Qt più ampio. Cambiare Info.plist o il deployment target del solo eseguibile Go non abbassa i requisiti incorporati nelle librerie distribuite.

`otool -L` non ha rilevato dipendenze assolute esterne al sistema dopo aver escluso l’identità `LC_ID_DYLIB` della libreria stessa (`otool -D`). Alcune identità dei framework Qt contengono ancora percorsi Homebrew: da sole non sono dipendenze esterne. Questo controllo non prova tutti i caricamenti dinamici dei plugin, il comportamento delle firme o l’avvio su macchina pulita. La verifica locale della firma ad hoc esistente è passata; firma Developer ID e notarizzazione non sono disponibili.

Per un minimo supportato inferiore occorre compilare o acquisire ogni dipendenza nativa per quella base, impostare flag di deployment coerenti, ispezionare tutti i load command risultanti e provare sul relativo OS. Non alterare i load command della versione minima per nascondere codice incompatibile. È stato chiesto all’autore se il minimo desiderato sia macOS 13, 15 o 26; nessuna risposta o promessa di supporto presunta.

## Punto di partenza dell’inventario

Framework osservati: QtCore, QtGui, QtWidgets, QtDBus. Le dylib separate comprendono libb2, D-Bus, double-conversion, FreeType, GLib/GThread, Graphite2, HarfBuzz, ICU, gettext/libintl, libjpeg-turbo, libpng, md4c, PCRE2 e Zstandard. Anche plugin Qt e codice terzo incorporato richiedono attribuzione; contare i file separati non produce un inventario completo.

L’albero Homebrew Qt 6.11.2 installato fornisce `sbom.spdx.json` e `share/qt/sbom/qtbase-6.11.2.spdx`. Sono fonti upstream utili, **non una SBOM di WebFence**: includono strumenti di compilazione e componenti assenti dall’app e non identificano da soli la versione di ogni libreria copiata. Durante la generazione di un pacchetto conservare hash degli artefatti, provenienza della build, moduli Go realmente collegati, hash degli archivi sorgente, patch e avvisi. `go.mod` elenca basi del core non tutte collegate alla GUI corrente; non presentare l’intero grafo come contenuto del binario.

Prima di distribuire un pacchetto raccogliere licenze/avvisi dei componenti effettivi e materiali richiesti per sorgenti corrispondenti/sostituzione, risolvendo la compatibilità tramite la [revisione legale](LEGAL-REVIEW.md). Le espressioni di licenza Homebrew descrivono una formula intera, non necessariamente ogni componente incorporato. La licenza WebFence non sostituisce le licenze delle dipendenze; nessun acquisto Qt commerciale o approvazione legale effettuati.

## Prossimo lavoro Linux e Windows

Debian 12 offre Qt 6.4.2; Debian 13 Qt 6.8.2 al momento della verifica. La CI Ubuntu corrente è utile ma non prova la base di un pacchetto Debian. Compilare contro il runtime minimo scelto, derivare i requisiti dai binari reali e provare l’installazione su un sistema corrispondente pulito. Per un `.deb` nativo preferire dipendenze Qt del sistema con requisiti espliciti di pacchetto/versione; un pacchetto portabile richiede invece un bundle di dipendenze verificato. Nessun installer Linux ancora presente.

La CI Windows usa MSYS2 UCRT64 su Windows Server. L’eseguibile richiede plugin/DLL Qt e dipendenze del compilatore/runtime: l’exe da solo non basta. Preparare l’intero runtime, provare senza MSYS2 nel PATH, poi verificare desktop Windows 10/11 reali e percorsi assistivi. Qt 6.11 elenca Windows 10 1809+ e Windows 11; Qt documenta 6.12 come ultimo ramo con supporto Windows 10. È un vincolo upstream, non una verifica WebFence completata o una promessa di manutenzione indefinita.

Fonti: [piattaforme supportate Qt](https://doc.qt.io/qt-6/supported-platforms.html), [Qt Debian 12](https://packages.debian.org/bookworm/libqt6core6), [Qt Debian 13](https://packages.debian.org/trixie/libqt6core6t64), metadati Mach-O locali e SBOM Qt installate. Versioni dei pacchetti e metadati host esaminati nella data indicata.

## Aggiornamento del processo di build

Lo script macOS ora prepara il bundle nel filesystem temporaneo dell’host, verifica la firma ad hoc, copia completamente sul volume di destinazione e pubblica con rinomina; conserva il vecchio artefatto fino alla sostituzione e lo ripristina se la pubblicazione fallisce. Evita riscritture ripetute sul disco esterno e bundle finale parziale durante la build. Build locale e doppia verifica codesign superate. Prova con errore del compilatore introdotto appositamente: fallimento rilevato, hash del binario precedente invariato e staging ripulito. Nessun benchmark di velocità.

Il bundle macOS ricompila ora il plugin Cocoa Qt 6.11.2 con una correzione temporanea del crash assistivo: [ADR-006](ADR-006-QT-COCOA.md). Occorrono anche CMake, Ninja e gli header MoltenVK/Vulkan (`brew install cmake ninja molten-vk vulkan-headers`); il sorgente Qt viene scaricato e verificato per hash. Versioni Qt diverse vengono rifiutate finché non rivalutate. I binari non confezionati continuano a usare Qt installato.

Bundle locale aggiornato verificato il 2026-09-23: revisione Go `953ed2daf5061b4044f7cd9d8ff72b133153d260`, `vcs.modified=false`. SHA-256 eseguibile `4af00ede347509677b52c55bd8a355a77faef6d9879327868ae325ef124b0abf`; plugin Cocoa corretto `acc31e86a786050c2e7280c5be09df30718262d7b4514dc5d82f65e8d13d5511`. Build e firma ad hoc verificate; [CI quattro target verde](https://github.com/Matte2599/WebFence/actions/runs/35873709410). Questi hash identificano artefatti locali, non garantiscono build bit-per-bit riproducibili. Il minimo OS del nuovo bundle non è stato ridotto.
