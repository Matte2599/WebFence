# Sviluppo, desktop e localizzazione

[English](../en/DEVELOPMENT.md) · [Indice](../README.md)

## Stato attuale e avvio

Primo prototipo M0 offline: finestra nativa, dataset sintetico a richiesta, filtri, tabella virtualizzata, dettaglio prove, copia esplicita e lingua persistente. Nessun crawler, motore di rete, database, CVE, firma o AI. Leggere [resoconto M0](M0-DESKTOP.md) e [direzione UX](UX.md).

Prerequisiti: Go **1.27.1**, Fyne **2.8.1** fissato in `go.mod`, compilatore C e librerie grafiche. macOS richiede Xcode/Command Line Tools; Debian/derivati richiedono `gcc libgl1-mesa-dev xorg-dev libxkbcommon-dev libwayland-dev`; Windows richiede GCC/MinGW-w64 a 64 bit nel PATH. Le dipendenze Go vengono scaricate al primo avvio della toolchain; l'applicazione non effettua scansioni o download.

```sh
git clone https://github.com/Matte2599/WebFence.git
cd WebFence
go mod download
go run ./cmd/webfence
```

Per compilare su macOS/Linux e per i controlli automatici:

```sh
go build -trimpath -o bin/ ./cmd/webfence
go mod verify
go vet -tags ci ./...
go test -tags ci -race ./...
git diff --check
```

Su Windows lo stesso comando di build produce `bin/webfence.exe`; eseguirlo da PowerShell o Explorer. Il tag `ci` usa il driver grafico software nei test: **non prova la finestra nativa né uno screen reader**. Le prove locali eseguite e la CI sono distinte nel resoconto.

## Struttura implementata

```text
cmd/webfence/           punto d'ingresso desktop
internal/demo/         fixture sintetiche pure, senza I/O
internal/i18n/         cataloghi JSON incorporati IT/EN
internal/ui/           workspace Fyne e test delle interazioni
scripts/package-macos.sh
.github/workflows/ci.yml
DOCS/                  documentazione bilingue
```

Il pacchetto `demo` non è un motore di analisi e i suoi record non sono finding. I futuri pacchetti di dominio/rete/storage nasceranno con casi d'uso verificabili; non creare contenitori vuoti. Il core rimarrà indipendente da Fyne.

## Sistemi e packaging

Requisito confermato: **macOS solo Apple Silicon; Windows 10 e 11 x86-64; Linux Debian e derivati x86-64 e ARM64**. Niente architetture a 32 bit. Versioni minime di macOS/Debian ancora da definire sulla base di prove. La CI compila su macOS ARM64, Windows Server x64 e Ubuntu x64/ARM64; non dimostra compatibilità interattiva con Windows 10/11 o tutte le distribuzioni Debian.

Su macOS Apple Silicon:

```sh
sh scripts/package-macos.sh
open dist/WebFence.app
```

Il bundle locale `0.0.1` serve esclusivamente alla valutazione M0; non è una release. Lo script valida il plist, non installa certificati, non firma con un'identità di distribuzione e non notarizza. La firma eventualmente inserita dalla toolchain non equivale a Developer ID. `dist/` e `bin/` sono esclusi da Git. Nessuna semplice cross-compilation con CGO disabilitato è promessa.

Per riprodurre l'esperimento di accessibilità Fyne su macOS, chiudere prima il prototipo e ricompilare:

```sh
FYNE_BUILD_TAGS=accessibility sh scripts/package-macos.sh
```

Il bridge sperimentale **non ha superato il gate M0**. La build normale non lo abilita; la selezione definitiva del toolkit resta aperta. Per tornare alla build normale, richiudere e rieseguire lo script senza quella variabile. Vedi [limiti osservati](M0-DESKTOP.md).

La sola preferenza applicativa persistente è `ui.language`, gestita da Fyne nella directory utente dell'app `io.github.Matte2599.WebFence`, non nel repository. Esempi, filtri e selezione non sono salvati. «Svuota esempi» rimuove i record in memoria, non ciò che l'utente ha già copiato negli appunti. Keychain, database e loro cancellazione sono ancora da implementare.

## IT/EN

Cataloghi di stringhe con chiavi stabili, italiano e inglese completi. Lingua iniziale dal sistema se supportata, altrimenti inglese; selettore persistente nel desktop. Lingua del report configurabile separatamente. No concatenazione di frasi tradotte; plurali e numeri/date formattati in presentazione. Persistenza sempre UTC, codici di stato e ID in forma canonica.

I testi delle regole includono titolo, impatto, spiegazione e rimedio in entrambe le lingue. Le prove originali non sono tradotte né alterate; eventuali traduzioni sono annotazioni separate. Il fallback inglese evita blocchi, ma una traduzione mancante nelle funzioni rilasciate è un difetto da correggere.

Ogni PR che cambia requisiti o funzioni aggiorna i documenti omologhi IT/EN. La lingua non deve cambiare fingerprint, severità, conteggi, scope o decisioni. Una traduzione di un report firmato è un nuovo artefatto da firmare.

## Rilascio e operatività

Prima di un rilascio: test dei rischi pertinenti, scansione dipendenze/segreti, SBOM, inventario licenze, checksum, firma dei pacchetti, note IT/EN e procedura di rollback. Prima di migrare dati creare un backup verificato; il rollback del binario non inverte automaticamente lo schema. Aggiornamenti di regole e modelli sono versionati separatamente e non cambiano una run già avviata.

Log locali redatti con ID di run, durata, limiti ed errori; niente telemetria predefinita. In caso di spazio insufficiente, feed scaduto, chiave bloccata, runtime assente o OOM, mostrare il componente interessato e ciò che resta utilizzabile. Non convertire una degradazione in successo silenzioso.
