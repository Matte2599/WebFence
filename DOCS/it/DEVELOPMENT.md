# Sviluppo del desktop Qt

[English](../en/DEVELOPMENT.md) · [Indice](../README.md)

## Avvio e controlli

Qt Widgets/MIQT è la GUI principale scelta dall’autore. Go **1.27.1** e MIQT **0.14.0** sono fissati nel modulo. È ancora un prototipo offline con esempi: nessuno scanner, progetto persistente **nella GUI**, CVE o AI. Il core dispone di uno [store progetti M1 separato](M1-PROJECT-STORE.md). [Stato e verifiche desktop](QT-DESKTOP.md).

La prima compilazione dei binding può richiedere diversi minuti; le successive beneficiano della cache Go. La CI macOS compila i wrapper C++ con `-O0 -g0` per contenere il costo della prima compilazione; build e bundle usano gli stessi flag e la cache. Le librerie Qt installate restano quelle del pacchetto Homebrew. Lo script locale usa per default `-O2 -g`; la CI verifica funzionalità, non prestazioni di release.

Servono compilatore C++17, CGO, pkg-config e Qt 6 Core/Gui/Widgets. Su macOS Apple Silicon installare Xcode/Command Line Tools e `brew install qtbase pkgconf`; su Debian/Ubuntu `sudo apt-get install g++ pkg-config qt6-base-dev`.

```sh
go mod download
export CGO_CXXFLAGS='-O2 -g -std=c++17'
go run ./cmd/webfence
```

```sh
go mod verify
go test -race ./internal/demo ./internal/i18n ./internal/preferences ./internal/scope ./internal/project ./internal/storage ./internal/transport ./internal/foundation ./internal/signature ./internal/credentials
go vet ./...
go build -o bin/webfence ./cmd/webfence
QT_QPA_PLATFORM=offscreen ./bin/webfence --self-test
git diff --check
```

Il self-test usa Qt reale senza display e una directory temporanea per la lingua; la copia è intercettata per non modificare gli appunti. Verifica anche errori di salvataggio, testo lungo e cambio lingua tramite azioni. Non è una prova con screen reader. Il tag Fyne `ci` non serve più. `go test ./...` richiede ora le dipendenze Qt per compilare il desktop; i pacchetti puri si verificano separatamente come sopra.

Su Windows x86-64 usare MSYS2 **UCRT64** con Go nel PATH e toolchain coerente: `mingw-w64-ucrt-x86_64-gcc`, `mingw-w64-ucrt-x86_64-pkgconf`, `mingw-w64-ucrt-x86_64-qt6-base`. Dalla shell UCRT64, usare le stesse variabili e compilare con `go build -ldflags "-H=windowsgui -s -w" -o bin/webfence.exe ./cmd/webfence`; eseguire `./bin/webfence.exe --self-test` con `QT_QPA_PLATFORM=offscreen`. Qt DLL e plugin devono essere disponibili; il file exe da solo non è un pacchetto distribuibile. La CI usa [setup-msys2](https://github.com/msys2/setup-msys2); MSVC non è il compilatore CGO di questa configurazione.

## Struttura e dati

- `cmd/webfence`: avvio desktop e opzione `--self-test`.
- `internal/desktop`: workspace Qt, ownership dei valori del modello e self-test.
- `internal/demo`: fixture pure senza rete.
- `internal/i18n`: cataloghi JSON IT/EN incorporati.
- `internal/preferences`: sola lingua, indipendente dal toolkit.
- `internal/project` e `internal/storage`: modello di autorizzazione e store SQLite dei soli metadati, ancora scollegati dalla GUI.

I futuri pacchetti del motore restano indipendenti da Qt. Non usare la cache delle fixture come modello di conservazione per dati reali. Fyne e il vecchio laboratorio Qt sono nella cronologia Git, non nella build corrente.

Nel desktop la sola preferenza salvata è il file `WebFence/ui-language` sotto `os.UserConfigDir()` (macOS: `~/Library/Application Support`; Linux: `$XDG_CONFIG_HOME` o `~/.config`; Windows: `%AppData%`). La scrittura usa un temporaneo e rinomina nella stessa directory. Errori sono visibili nella GUI; la lingua della sessione resta utilizzabile. Le vecchie preferenze Fyne non sono importate né cancellate. Esempi, filtri e selezione non sono salvati. Gli appunti copiati esplicitamente non vengono cancellati da «Svuota esempi».

Menu Lingua e selettore cambiano IT/EN; scorciatoie **Ctrl+1/Ctrl+2** (**Cmd+1/Cmd+2** su macOS). **Ctrl+O/Cmd+O** carica gli esempi. Tastiera e screen reader completi restano da convalidare.

Il menu Visualizza aggiunge ricerca (Ctrl/Cmd+F), risultati (F6), evidenze (Ctrl/Cmd+Shift+E) e dettagli avanzati (Ctrl/Cmd+Shift+D). Vedi [tastiera, focus e limiti assistivi](QT-DESKTOP.md).

## Piattaforme e bundle

Requisiti: macOS Apple Silicon, Windows 10/11 x86-64, Debian/derivati x86-64 e ARM64; nessun 32 bit. Versioni minime e mantenimento Windows 10 restano da verificare secondo [ADR-002](ADR-002-GUI.md). Build CI non equivalgono a uso assistivo sui sistemi richiesti.

```sh
sh scripts/package-macos.sh
open dist/WebFence.app
```

Lo script genera il bundle locale `0.0.1`, valida il plist e usa `macdeployqt` per le librerie/plugin. Richiede Homebrew Qt e Apple Silicon. Nessuna firma Developer ID o notarizzazione; resta da provare su macchina pulita. `bin/` e `dist/` sono esclusi da Git. La licenza WebFence resta invariata; Qt/MIQT hanno diritti separati e obblighi da verificare prima della distribuzione.

Nel bundle preparato lo script elimina gli `LC_RPATH` esterni, poi controlla che le dipendenze Mach-O esplicite si risolvano nel bundle o nelle librerie di sistema Apple prima della firma. Si può ricontrollare l'artefatto già pubblicato con `python3 scripts/check-macos-linkage.py dist/WebFence.app`; [perimetro ed evidenza](../evidence/macos-linkage-closure-2026-09-24.md).

## IT/EN

Cataloghi di stringhe con chiavi stabili, italiano e inglese completi. Lingua iniziale dal sistema se supportata, altrimenti inglese; selettore persistente nel desktop. Lingua del report configurabile separatamente. No concatenazione di frasi tradotte; plurali e numeri/date formattati in presentazione. Persistenza sempre UTC, codici di stato e ID in forma canonica.

I testi delle regole includono titolo, impatto, spiegazione e rimedio in entrambe le lingue. Le prove originali non sono tradotte né alterate; eventuali traduzioni sono annotazioni separate. Il fallback inglese evita blocchi, ma una traduzione mancante nelle funzioni rilasciate è un difetto da correggere.

Ogni PR che cambia requisiti o funzioni aggiorna i documenti omologhi IT/EN. La lingua non deve cambiare fingerprint, severità, conteggi, scope o decisioni. Una traduzione di un report firmato è un nuovo artefatto da firmare.

## Rilascio e operatività

Prima di un rilascio: test dei rischi pertinenti, scansione dipendenze/segreti, SBOM, inventario licenze, checksum, firma dei pacchetti, note IT/EN e procedura di rollback. Prima di migrare dati creare un backup verificato; il rollback del binario non inverte automaticamente lo schema. Aggiornamenti di regole e modelli sono versionati separatamente e non cambiano una run già avviata.

Log locali redatti con ID di run, durata, limiti ed errori; niente telemetria predefinita. In caso di spazio insufficiente, feed scaduto, chiave bloccata, runtime assente o OOM, mostrare il componente interessato e ciò che resta utilizzabile. Non convertire una degradazione in successo silenzioso.

## Primo livello scope (M0)

`internal/scope` contiene la policy immutabile delle origini; il laboratorio HTTP è soltanto in `lab_test.go`. [Contratto e limiti](M0-SCOPE.md). Il desktop non effettua richieste.

```sh
go test -race -cover ./internal/scope
go test ./internal/scope -run '^$' -fuzz '^FuzzCheck$' -fuzztime=20s -parallel=4
```

## Trasporto di laboratorio M0

`internal/transport.NewLab` usa origini/IP loopback espliciti, resolver controllato e limiti obbligatori. Non è collegato al desktop. [Decisione, contratto e limiti](ADR-003-TRANSPORT.md).

```sh
go test -race -cover ./internal/transport
```

## SQLite e JWS: fondazioni M0

Driver e libreria scelti in [ADR-004](ADR-004-STORAGE-SIGNATURE.md). `internal/foundation` contiene solo test SQLite su file temporanei; `internal/signature` espone firma/verifica di byte, senza report, JCS o portachiavi. Nessuno dei due è collegato alla GUI.

```sh
go test -race ./internal/foundation ./internal/signature
go test ./internal/signature -run '^$' -fuzz '^FuzzVerify$' -fuzztime=20s -parallel=4
```

## Portachiavi: prove M0

[ADR-005](ADR-005-CREDENTIALS.md) descrive backend, limiti e prove. Gli unit test ordinari non scrivono credenziali personali. Per le prove native macOS (portachiavi temporaneo separato) e Windows (voce sintetica con namespace casuale):

```sh
go test -race -tags=keychainintegration ./internal/credentials -count=1 -v
```

Su Linux usare esclusivamente il launcher isolato, con `dbus-run-session`, `gnome-keyring-daemon`, `gdbus` e `rg` installati. Non impostare manualmente il flag di isolamento sul bus personale: il test blocca la raccolta del servizio di prova.

```sh
sh scripts/test-keychain-linux.sh
```

La raccolta facoltativa dei sorgenti Windows richiede anche GnuPG nel PATH: verifica offline le otto firme delle versioni vincolate usando solo le chiavi pubbliche nel repository. [Procedura e limiti](M0-WINDOWS-SOURCES.md#verifica-offline-delle-otto-firme-distaccate). La normale build GUI non richiede GnuPG.

Il bundle macOS ricompila ora il plugin Cocoa Qt 6.11.2 con una correzione temporanea del crash assistivo: [ADR-006](ADR-006-QT-COCOA.md). Occorrono anche CMake, Ninja e gli header MoltenVK/Vulkan (`brew install cmake ninja molten-vk vulkan-headers`); il sorgente Qt viene scaricato e verificato per hash. Versioni Qt diverse vengono rifiutate finché non rivalutate. I binari non confezionati continuano a usare Qt installato.

Aggiunti confezionamento `.deb`/ZIP Windows e collaudi del runtime separato: [ADR-007 e comandi](ADR-007-PACKAGING.md). [CI `32655b0` superata, sei job](https://github.com/Matte2599/WebFence/actions/runs/35877356234); pacchetti di sviluppo, non release supportate.

Il packaging macOS richiede anche Python 3.9+ per raccogliere provenienza e avvisi prima della firma: [materiali e lacune note](M0-PACKAGING.md). Il controllo rifiuta librerie native non abbinate o ambigue.

Raccolta facoltativa degli archivi sorgente e avvisi da un inventario macOS: [procedura, requisiti e test](M0-SOURCE-MATERIALS.md). Non viene eseguita automaticamente dalla build del prodotto.

Impostare `WEBFENCE_NATIVE_SOURCE_MATERIALS` alla directory di una raccolta completata per includere gli avvisi sorgente ricontrollati prima della firma macOS; vedere la stessa procedura. Gli archivi restano separati.

Documentazione offline inclusa su tutti i pacchetti tramite un raccoglitore comune, con verifica dei file collegati: [procedura e limiti](ADR-007-PACKAGING.md#documentazione-offline-nei-pacchetti).
