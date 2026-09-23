# ADR-007 — Pacchetti di prova Debian e Windows

[English](../en/ADR-007-PACKAGING.md) · [Indice](../README.md)

Data: 2026-09-23. Stato: implementato; verifiche CI per piattaforma riportate sotto. Pacchetti di sviluppo M0, non release supportate. I gate di distribuzione, licenze complete e collaudo assistivo restano nella [matrice](M0-VALIDATION.md).

## Decisione

**Debian:** `.deb` nativo amd64/arm64, compilato in un’immagine ufficiale Go 1.27.1 basata su Debian 12. Immagini multiarch fissate per digest; pacchetti APT aggiornati dai repository Debian, versioni effettive registrate. Debian 12 è una base tecnica candidata, non una nuova promessa di supporto approvata dall’autore. Qt resta una dipendenza del sistema: nessuna copia privata di glibc o Qt. `dpkg-shlibdeps` ricava i vincoli dai simboli ELF reali. QtGui Debian include XCB/offscreen; `qt6-qpa-plugins` contiene altri backend e non viene aggiunto inutilmente. Wayland è suggerito tramite `qt6-wayland`, ancora da collaudare.

**Windows:** ZIP portabile UCRT64 x86-64, senza installer, privilegi amministrativi o firma Authenticode. `windeployqt6` raccoglie i plugin Qt; un controllo ricorsivo degli import PE aggiunge le DLL del medesimo prefix UCRT64. Sono accettati soltanto PE x86-64. Import di sistema/API set registrati, DLL mancanti o ambigue e pacchetti senza attribuzioni fanno fallire il confezionamento. Il controllo degli import non copre tutti i possibili caricamenti dinamici: servono anche esecuzione del pacchetto e desktop reali.

## Provenienza e licenze

`package-go-notices.py` legge i moduli **effettivamente collegati** nel binario, controlla versione/hash contro il grafo Go e copia gli avvisi presenti alla radice dei moduli e la licenza Go. Rifiuta sostituzioni locali non esaminate. Non considera tutto `go.mod` come contenuto della GUI. Sono metadati e avvisi iniziali, non una SBOM completa né revisione legale.

Il `.deb` contiene LICENSE, README IT/EN, metadata Go, versioni native usate per compilare, revisione dichiarata e hash degli input sorgente. Nel container il binario usa `-buildvcs=false`: la provenienza esterna e gli hash distinguono questo caso da una revisione Git incorporata. Il flag dirty deve descrivere il contesto realmente passato; la CI usa il checkout corrente, le prove locali con modifiche indicano dirty=true.

Lo ZIP registra ogni DLL con hash del file copiato e dell’originale, pacchetto MSYS2 proprietario, metadati `pacman -Qi` e copie dei relativi avvisi. I file Qt riscritti da windeployqt possono avere hash differente dall’originale. Le licenze delle dipendenze non vengono sostituite dalla licenza WebFence. Inventario completo dei componenti incorporati, sorgenti corrispondenti e materiali richiesti per la distribuzione restano da completare. La CI costruisce e prova i pacchetti senza pubblicarli come release o caricare automaticamente artefatti binari.

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

Il container runtime Debian parte dall’immagine slim senza Go, gcc o qmake. Installa il `.deb` tramite APT, esegue i self-test come utente non privilegiato, prima offscreen e poi XCB in Xvfb con Openbox. Esegue purge e verifica rimozione di eseguibile/launcher e conservazione di una preferenza utente sintetica. I log e il checksum sono esportati solo se tutte le fasi passano. Questo prova dipendenze dichiarate e comportamento del pacchetto nel container, non un desktop Debian reale, Wayland, Orca, GPU o installazione fisica.

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
