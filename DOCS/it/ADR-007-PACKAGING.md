# ADR-007 — Pacchetti di prova Debian e Windows

[English](../en/ADR-007-PACKAGING.md) · [Indice](../README.md)

Data: 2026-09-23. Stato: implementato; verifiche CI per piattaforma riportate sotto. Pacchetti di sviluppo M0, non release supportate. I gate di distribuzione, licenze complete e collaudo assistivo restano nella [matrice](M0-VALIDATION.md).

## Decisione

**Debian:** `.deb` nativo amd64/arm64, compilato in un’immagine ufficiale Go 1.27.1 basata su Debian 12. Immagini multiarch fissate per digest; pacchetti APT aggiornati dai repository Debian, versioni effettive registrate. Debian 12 è una base tecnica candidata, non una nuova promessa di supporto approvata dall’autore. Qt resta una dipendenza del sistema: nessuna copia privata di glibc o Qt. `dpkg-shlibdeps` ricava i vincoli dai simboli ELF reali. QtGui Debian include XCB/offscreen; `qt6-qpa-plugins` contiene altri backend e non viene aggiunto inutilmente. Wayland è suggerito tramite `qt6-wayland`, ancora da collaudare.

**Windows:** ZIP portabile UCRT64 x86-64, senza installer, privilegi amministrativi o firma Authenticode. `windeployqt6` raccoglie i plugin Qt; un controllo ricorsivo degli import PE aggiunge le DLL del medesimo prefix UCRT64. Sono accettati soltanto PE x86-64. Import di sistema/API set registrati, DLL mancanti o ambigue e pacchetti senza attribuzioni fanno fallire il confezionamento. Il controllo degli import non copre tutti i possibili caricamenti dinamici: servono anche esecuzione del pacchetto e desktop reali.

## Provenienza e licenze

`package-go-notices.py` legge i moduli **effettivamente collegati** nel binario, controlla versione/hash contro il grafo Go e copia gli avvisi presenti alla radice dei moduli e la licenza Go. Rifiuta sostituzioni locali non esaminate. Non considera tutto `go.mod` come contenuto della GUI. Sono metadati e avvisi iniziali, non una SBOM completa né revisione legale.

Il `.deb` contiene LICENSE, README IT/EN, metadata Go, versioni native usate per compilare, revisione dichiarata e hash degli input sorgente. Nel container il binario usa `-buildvcs=false`: la provenienza esterna e gli hash distinguono questo caso da una revisione Git incorporata. Il flag dirty deve descrivere il contesto realmente passato; la CI usa il checkout corrente, le prove locali con modifiche indicano dirty=true.

Lo ZIP registra ogni DLL con hash del file copiato e dell’originale, pacchetto MSYS2 proprietario, metadati `pacman -Qi` e copie dei relativi avvisi. I notices selezionati sono confrontati per hash con i membri degli archivi binari MSYS2 e ricontrollati dopo l'estrazione dello ZIP: [43 DLL, 22 pacchetti e 74 notices nel primo collaudo CI](../evidence/windows-notice-linkage-2026-09-24.md); la [selezione corrente comprende 136 file](../evidence/windows-attribution-sidecars-2026-09-24.md). I file Qt riscritti da windeployqt possono avere hash differente dall’originale. Le licenze delle dipendenze non vengono sostituite dalla licenza WebFence. Inventario completo dei componenti incorporati, sorgenti corrispondenti e materiali richiesti per la distribuzione restano da completare. La CI costruisce e prova i pacchetti senza pubblicarli come release o caricare automaticamente artefatti binari.

Gli SHA-256 dei 22 archivi binari originali sono ora [vincolati ai checksum pubblicati sulle pagine MSYS2](../evidence/windows-binary-checksum-lock-2026-09-24.md). Il registro revisionato è incluso nello ZIP e la prova sull'archivio estratto ne ricontrolla il legame con `native-build.json`. Le versioni nuove richiedono riesame esplicito; il controllo non sostituisce firme PGP, inventario del codice incorporato o revisione legale. Il packaging verifica inoltre che i riferimenti ai testi di licenza nei JSON di attribuzione Qt selezionati rimangano confinati ai protocolli e puntino a file inclusi; il test dello ZIP richiede 27 riferimenti per la versione bloccata.

## Comandi presenti

Su un host con Docker, build e collaudo Debian dell’architettura nativa:

```sh
docker build --progress=plain --file packaging/debian/Dockerfile \
  --build-arg SOURCE_REVISION="$(git rev-parse HEAD)" \
  --build-arg SOURCE_DIRTY=true \
  --output type=local,dest=dist/debian-test .
```

`SOURCE_DIRTY=true` segnala esplicitamente una prova di sviluppo; usare false soltanto con un contesto pulito verificato. Gli output comprendono `.deb`, checksum e log delle prove. In un ambiente Debian già predisposto: `sh scripts/package-debian.sh`; richiede Go, CGO/g++, Qt 6 development, pkg-config, dpkg-dev, desktop-file-utils e Python 3. Non è una cross-compilazione.

Nel terminale MSYS2 UCRT64, dopo `go build -ldflags "-H=windowsgui -s -w" -o bin/webfence.exe ./cmd/webfence`, con Python UCRT64 installato:

```sh
python3 scripts/package-windows.py "$(cygpath -w /ucrt64)" "$(cygpath -w "$PWD/bin/webfence.exe")"
```

Da PowerShell 7: `./scripts/test-windows-package.ps1 -Archive ./dist/webfence-windows-amd64.zip`. Il test usa una directory temporanea con spazi e Unicode e processi figli con PATH limitato a Windows; non cambia il PATH personale. Prova offscreen e plugin Windows nativo. Nessun dato di cliente o target viene incluso.

## Prove e limiti

Il container runtime Debian parte dall’immagine slim senza Go, gcc o qmake. Installa il `.deb` tramite APT, esegue i self-test come utente non privilegiato, prima offscreen e poi XCB in Xvfb con Openbox. Una prova separata avvia la GUI normale in una sessione privata Xvfb/D-Bus, legge l'albero AT-SPI in italiano e inglese e controlla la sequenza tabella 10.000 → 1 → 0 → 10.000 con la prima cella valida dopo il reset; [procedura ed evidenza](../evidence/debian-atspi-2026-09-24.md). Python, AT-SPI e D-Bus sono dipendenze del **solo container di test**, non del `.deb`. Esegue purge e verifica rimozione di eseguibile/launcher e conservazione di una preferenza utente sintetica. I log e il checksum sono esportati solo se tutte le fasi passano. Questo prova dipendenze dichiarate e comportamento del pacchetto nel container, non un desktop Debian reale, Wayland, annunci di Orca, GPU o installazione fisica.

Prima prova XCB: i test di focus fallivano in un server X senza window manager. Il collaudo ora attende un window manager dedicato; il self-test dell’app attende fino a tre secondi l’attivazione asincrona della finestra prima delle verifiche tastiera e fallisce esplicitamente se non avviene. La normale esecuzione interattiva non viene modificata.

Windows CI usa Windows Server, non Windows 10/11. PATH ripulito e avvio da ZIP estratto sono evidenze più forti della sola build, ma non equivalgono a una macchina completamente pulita o al collaudo NVDA. Esiti effettivi riportati sotto.

Fonti: [dpkg-shlibdeps](https://manpages.debian.org/bookworm/dpkg-dev/dpkg-shlibdeps.1.en.html), [file QtGui Debian](https://packages.debian.org/bookworm/arm64/libqt6gui6/filelist), [distribuzione Windows Qt](https://doc.qt.io/qt-6/windows-deployment.html), [Qt UCRT64 MSYS2](https://packages.msys2.org/packages/mingw-w64-ucrt-x86_64-qt6-base).

Prova locale ARM64 del 2026-09-23: build `.deb`, installazione nel runtime senza toolchain, self-test offscreen e XCB/Openbox da utente non privilegiato e purge superati. Verificati log e checksum esportati. Questa prova locale non stabilisce equivalenza con desktop reali. La build Windows usa il sottosistema GUI per evitare una console aggiuntiva.

Verifica CI `ea2896a`, [run 35876057107](https://github.com/Matte2599/WebFence/actions/runs/35876057107): build/test macOS e Linux e pacchetti Debian amd64/arm64 superati. Windows supera build e self-test, ma il confezionamento si ferma perché ICU conserva LICENSE in `share/icu/<versione>/LICENSE`. Raccolta corretta per includere avvisi nominati esplicitamente sotto `share`, sempre appartenenti al pacchetto installato; avvisi mancanti continuano a bloccare il pacchetto. [Inventario MSYS2 ICU](https://packages.msys2.org/packages/mingw-w64-ucrt-x86_64-icu). Verifica successiva riportata sotto. Aggiunto Installed-Size al `.deb`; preferenza sintetica per il test purge creata dallo stesso utente non privilegiato.

Windows verificato sul commit `32655b0` ([CI 35877356234](https://github.com/Matte2599/WebFence/actions/runs/35877356234)): ZIP completo, raccolta notices ICU corretta, self-test offscreen e backend Windows nativo superati con PATH di solo sistema e percorso con spazi/Unicode. SHA-256 dello ZIP generato sul runner: `9c5226fbb452fe8f0f2329496f0bc80cfff01532215df7d4dedb943735f5c6f1`; artefatto effimero non pubblicato. Non dimostra Windows 10/11 o NVDA.

Esito conclusivo del task: **tutti e sei i job verdi** sul codice `32655b0` ([run](https://github.com/Matte2599/WebFence/actions/runs/35877356234)). Debian amd64 e arm64 superano build, installazione, self-test offscreen/XCB e purge. Hash dei `.deb` effimeri della CI:

| Architecture | SHA-256 |
| --- | --- |
| amd64 | `50c539bc43ab9810e7c7a87723d8b2acbf731462ca3ec711e3d6ec8504d1cbd1` |
| arm64 | `a0ae5104727feca727d75c214021253fffa464f56a152a39f8909a5cad924405` |

Gli hash identificano questa esecuzione; timestamp e pacchetti APT/MSYS2 possono cambiare tra build. Non si dichiara riproducibilità bit-per-bit. M0-02 resta parziale per i gate descritti nella matrice.

## Documentazione offline nei pacchetti

Il raccoglitore comune `scripts/package-project-docs.py` conserva README IT/EN, LICENSE, DOCS, memoria, istruzioni per contributori/sicurezza e materiali locali richiamati dai documenti. È usato da macOS, Windows e Debian prima della pubblicazione dell’artefatto. Su Windows la documentazione è nella cartella `WebFence` dello ZIP; su Debian in `/usr/share/doc/webfence`, con LICENSE identico al file convenzionale `copyright`; su macOS in `Contents/Resources/notices`. La presenza della patch Cocoa nella documentazione Linux/Windows è materiale di riferimento, non indica che quei binari la utilizzino.

Il packaging verifica che i collegamenti Markdown locali puntino a file inclusi, senza accedere alla rete. Non verifica URL remoti o ancore di intestazione, né trasforma i comandi descritti in funzionalità del prodotto. Documenti esistenti non vengono sovrascritti; gli errori fanno fallire lo staging. Per ricontrollare un albero di documentazione confezionato: `python3 scripts/package-project-docs.py PERCORSO --check`.

Corregge un difetto dei precedenti pacchetti Windows/Debian: i README rimandavano a DOCS non inclusa, e Debian conservava LICENSE solo come `copyright`. Il contesto Docker ora include anche i documenti; gli hash degli input Debian li comprendono. Le prove dello ZIP estratto e del pacchetto Debian installato controllano gli accessi alla documentazione IT/EN e alla licenza.

Verifiche locali del raccoglitore: 68 documenti Markdown e 519 collegamenti locali; rimozione intenzionale di una guida rilevata, documenti già presenti preservati. Bundle macOS con firma/self-test Cocoa superati; `.deb` ARM64 compilato, installato e rimosso con test offscreen/XCB superati. Documentazione estratta dal `.deb`: 68 Markdown/521 link validi, LICENSE identico a copyright. SHA-256 del `.deb` locale: `e5c175563b2f6d5479ec873644fd20b7601fd12bcf635fbb6ddceae71c8f094a`. Build da `a4c621c` con modifiche dichiarate. [CI `f99b3e7`, run 35900777988](https://github.com/Matte2599/WebFence/actions/runs/35900777988): tutti e sei i job superati, documentazione inclusa verificata nei log Windows e Debian.

Il primo collaudo installato ha rilevato che `bookworm-slim` esclude i documenti tramite dpkg. Il solo container di test ora conserva esplicitamente `/usr/share/doc/webfence`; il `.deb` non modifica la policy dpkg dell’utente. Su sistemi configurati per eliminare documentazione, l’amministratore può continuare a farlo.

## Corrispondenza con i pacchetti binari MSYS2

Il packaging Windows ora richiede nella cache pacman un unico archivio binario della versione installata di ciascun pacchetto proprietario delle DLL incluse. Conservare `/var/cache/pacman/pkg` dopo l’installazione delle dipendenze. Archivi assenti/ambigui o versioni non corrispondenti fanno fallire il packaging prima della sostituzione dello ZIP; non viene effettuato un aggiornamento automatico delle dipendenze.

`scripts/msys2_binary_metadata.py` legge gli archivi progressivamente, senza estrarre o eseguire codice, confronta ogni DLL originale con il membro dell’archivio e conserva `.PKGINFO`/`.BUILDINFO`. Confronta nome, versione, pacchetto sorgente e architettura; registra hash dell’archivio e della ricetta. [BUILDINFO v2](https://man.archlinux.org/man/BUILDINFO.5.en) definisce `pkgbuild_sha256sum`; per pacchetti suddivisi il nome della DLL proprietaria deve comparire nell’elenco dei pacchetti prodotti. Il confronto è con la DLL originale: `windeployqt` può modificare la copia confezionata.

Il manifest aggiunge `binary_package_member` per DLL e `binary_package` per proprietario, con metadati e hash; mantiene `distribution_ready=false`. `zstd.exe` della toolchain è usato solo per decomprimere gli archivi in fase di build. Limiti per archivio: 512 MiB compressi, 2 GiB dichiarati espansi, 100.000 membri e 1 MiB per file di metadati; percorsi non validi, metadati duplicati e link al posto delle prove richieste sono rifiutati. La cache è un input di build fidato, non input di scansione/AI.

Questo controllo prova corrispondenza fra archivio in cache e DLL installate; non verifica firme pacman, disponibilità dei sorgenti o completezza delle licenze. La ricetta identificata dall’hash può essere acquisita e confrontata separatamente. Prova locale del lettore sul pacchetto Qt reale e 16 regressioni Python superate; la verifica completa è riportata sotto.

Prima CI `ceef622` ([run 35903124589](https://github.com/Matte2599/WebFence/actions/runs/35903124589)): Windows raggiunge il packaging ma fallisce per archivio Qt assente. I log mostrano che l’azione MSYS2 fissata esegue `pacman -Scc` dopo il salvataggio della cache. Impostato `cache: false` per quella sola azione: conserva i pacchetti scaricati nel job per la verifica, al costo di riscaricarli nei job successivi. Cache Go invariata; controlli sugli archivi non allentati. [CI correttiva `054bb2d`](https://github.com/Matte2599/WebFence/actions/runs/35904059667) completata: tutti e sei i job superati. Windows verifica 43 DLL da 22 pacchetti, con 21 pacchetti sorgente distinti; ZIP e test GUI superati. [Tabella e hash delle ricette](../evidence/windows-native-provenance-2026-09-23.md).

Il packaging Windows supporta anche la [raccolta e inclusione verificata dei sorgenti MSYS2](M0-WINDOWS-SOURCES.md#includere-i-sorgenti-nello-zip), con opzioni esplicite e prova dello ZIP estratto. Gli obblighi di distribuzione restano da revisionare.
