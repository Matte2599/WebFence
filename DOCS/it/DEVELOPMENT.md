# Sviluppo del desktop Qt

[English](../en/DEVELOPMENT.md) · [Indice](../README.md)

## Avvio e controlli

Qt Widgets/MIQT è la GUI principale scelta dall’autore. Go **1.27.1** e MIQT **0.14.0** sono fissati nel modulo. È ancora un prototipo offline con esempi: nessuno scanner, progetto persistente, CVE o AI. [Stato e verifiche](QT-DESKTOP.md).

La prima compilazione dei binding può richiedere diversi minuti; le successive beneficiano della cache Go.

Servono compilatore C++17, CGO, pkg-config e Qt 6 Core/Gui/Widgets. Su macOS Apple Silicon installare Xcode/Command Line Tools e `brew install qtbase pkgconf`; su Debian/Ubuntu `sudo apt-get install g++ pkg-config qt6-base-dev`.

```sh
go mod download
export CGO_CXXFLAGS='-O2 -g -std=c++17'
go run ./cmd/webfence
```

```sh
go mod verify
go test -race ./internal/demo ./internal/i18n ./internal/preferences
go vet ./...
go build -o bin/webfence ./cmd/webfence
QT_QPA_PLATFORM=offscreen ./bin/webfence --self-test
git diff --check
```

Il self-test usa Qt reale senza display e una directory temporanea per la lingua; la copia è intercettata per non modificare gli appunti. Verifica anche errori di salvataggio, testo lungo e cambio lingua tramite azioni. Non è una prova con screen reader. Il tag Fyne `ci` non serve più. `go test ./...` richiede ora le dipendenze Qt per compilare il desktop; i pacchetti puri si verificano separatamente come sopra.

Su Windows x86-64 usare MSYS2 **UCRT64** con Go nel PATH e toolchain coerente: `mingw-w64-ucrt-x86_64-gcc`, `mingw-w64-ucrt-x86_64-pkgconf`, `mingw-w64-ucrt-x86_64-qt6-base`. Dalla shell UCRT64, usare le stesse variabili e compilare con `go build -o bin/webfence.exe ./cmd/webfence`; eseguire `./bin/webfence.exe --self-test` con `QT_QPA_PLATFORM=offscreen`. Qt DLL e plugin devono essere disponibili; il file exe da solo non è un pacchetto distribuibile. La CI usa [setup-msys2](https://github.com/msys2/setup-msys2); MSVC non è il compilatore CGO di questa configurazione.

## Struttura e dati

- `cmd/webfence`: avvio desktop e opzione `--self-test`.
- `internal/desktop`: workspace Qt, ownership dei valori del modello e self-test.
- `internal/demo`: fixture pure senza rete.
- `internal/i18n`: cataloghi JSON IT/EN incorporati.
- `internal/preferences`: sola lingua, indipendente dal toolkit.

I futuri pacchetti del motore restano indipendenti da Qt. Non usare la cache delle fixture come modello di conservazione per dati reali. Fyne e il vecchio laboratorio Qt sono nella cronologia Git, non nella build corrente.

La sola preferenza salvata è il file `WebFence/ui-language` sotto `os.UserConfigDir()` (macOS: `~/Library/Application Support`; Linux: `$XDG_CONFIG_HOME` o `~/.config`; Windows: `%AppData%`). La scrittura usa un temporaneo e rinomina nella stessa directory. Errori sono visibili nella GUI; la lingua della sessione resta utilizzabile. Le vecchie preferenze Fyne non sono importate né cancellate. Esempi, filtri e selezione non sono salvati. Gli appunti copiati esplicitamente non vengono cancellati da «Svuota esempi».

Menu Lingua e selettore cambiano IT/EN; scorciatoie **Ctrl+1/Ctrl+2** (**Cmd+1/Cmd+2** su macOS). **Ctrl+O/Cmd+O** carica gli esempi. Tastiera e screen reader completi restano da convalidare.

## Piattaforme e bundle

Requisiti: macOS Apple Silicon, Windows 10/11 x86-64, Debian/derivati x86-64 e ARM64; nessun 32 bit. Versioni minime e mantenimento Windows 10 restano da verificare secondo [ADR-002](ADR-002-GUI.md). Build CI non equivalgono a uso assistivo sui sistemi richiesti.

```sh
sh scripts/package-macos.sh
open dist/WebFence.app
```

Lo script genera il bundle locale `0.0.1`, valida il plist e usa `macdeployqt` per le librerie/plugin. Richiede Homebrew Qt e Apple Silicon. Nessuna firma Developer ID o notarizzazione; resta da provare su macchina pulita. `bin/` e `dist/` sono esclusi da Git. La licenza WebFence resta invariata; Qt/MIQT hanno diritti separati e obblighi da verificare prima della distribuzione.

## IT/EN

Cataloghi di stringhe con chiavi stabili, italiano e inglese completi. Lingua iniziale dal sistema se supportata, altrimenti inglese; selettore persistente nel desktop. Lingua del report configurabile separatamente. No concatenazione di frasi tradotte; plurali e numeri/date formattati in presentazione. Persistenza sempre UTC, codici di stato e ID in forma canonica.

I testi delle regole includono titolo, impatto, spiegazione e rimedio in entrambe le lingue. Le prove originali non sono tradotte né alterate; eventuali traduzioni sono annotazioni separate. Il fallback inglese evita blocchi, ma una traduzione mancante nelle funzioni rilasciate è un difetto da correggere.

Ogni PR che cambia requisiti o funzioni aggiorna i documenti omologhi IT/EN. La lingua non deve cambiare fingerprint, severità, conteggi, scope o decisioni. Una traduzione di un report firmato è un nuovo artefatto da firmare.

## Rilascio e operatività

Prima di un rilascio: test dei rischi pertinenti, scansione dipendenze/segreti, SBOM, inventario licenze, checksum, firma dei pacchetti, note IT/EN e procedura di rollback. Prima di migrare dati creare un backup verificato; il rollback del binario non inverte automaticamente lo schema. Aggiornamenti di regole e modelli sono versionati separatamente e non cambiano una run già avviata.

Log locali redatti con ID di run, durata, limiti ed errori; niente telemetria predefinita. In caso di spazio insufficiente, feed scaduto, chiave bloccata, runtime assente o OOM, mostrare il componente interessato e ciò che resta utilizzabile. Non convertire una degradazione in successo silenzioso.
