# Qt Widgets / MIQT laboratory

**IT:** Esperimento M0, non una migrazione approvata. Usa gli stessi record sintetici e cataloghi di WebFence. Non invia richieste, non esegue scansioni, non salva progetti o preferenze. La lingua iniziale segue il sistema (fallback inglese); il selettore vale per la sessione. La vista avanzata mostra testo inerte e la copia è esplicita. I dati copiati restano negli appunti anche dopo lo svuotamento della GUI.

**EN:** M0 experiment, not an approved migration. Uses WebFence's same synthetic records and catalogs. No requests, scans, saved projects or preferences. Initial language follows the system (English fallback); selection is session-only. Advanced view displays inert text and copying is explicit. Copied data remains in the clipboard after clearing the GUI.

## Build / Compilazione

**IT:** Richiede Go 1.27.1, MIQT 0.14.0, compilatore C++ e librerie di sviluppo Qt 6 Widgets. L’applicazione principale resta Fyne. Il modulo annidato non viene verificato da `go test ./...` nella radice: eseguire separatamente i comandi seguenti.

**EN:** Go 1.27.1, MIQT 0.14.0 (`go.mod`), a C++ compiler and Qt 6 Widgets development libraries. The root application remains Fyne. This nested module is excluded from root `go test ./...`; run its checks explicitly.

macOS Apple Silicon, Homebrew:

```sh
brew install qtbase pkgconf
cd experiments/qt
export CGO_CXXFLAGS='-O2 -g -std=c++17'
go build -o ../../bin/webfence-qt .
../../bin/webfence-qt
```

**IT:** Dipendenze Debian/Ubuntu: `g++ pkg-config qt6-base-dev`. Le versioni effettivamente provate sono nel resoconto del confronto; la sola installazione non dimostra compatibilità. Per l’uso nativo serve una sessione grafica.

**EN:** Debian/Ubuntu development dependencies: `g++ pkg-config qt6-base-dev`. The exact tested Qt/platform versions and results are in the comparison report, not inferred from successful installation. Qt requires a working graphical session for native use.

```sh
export CGO_CXXFLAGS='-O2 -g -std=c++17'
go mod verify
go vet ./...
QT_QPA_PLATFORM=offscreen go run . --self-test
```

**IT:** Il self-test usa widget e modello Qt reali sul thread principale: 10.000 righe, filtri, cambio lingua, selezione, svuotamento e prove lunghe. Non scrive negli appunti e non dimostra usabilità con screen reader. L’ispezione nativa è separata. La prima compilazione di MIQT include i binding C++ e può richiedere minuti; le successive usano la cache Go.

**EN:** The self-test exercises actual Qt widgets/model on the main OS thread, including 10,000 rows, filtering, language changes, selection, clearing and long evidence. It does not write to the clipboard or prove screen-reader usability. Native UI inspection is separate. First MIQT compilation includes C++ bindings and can take several minutes; later builds use Go's cache.

## macOS development bundle / Bundle macOS

From the repository root / Dalla radice:

```sh
sh experiments/qt/package-macos.sh
open dist/WebFence-Qt-Lab.app
```

**IT:** `macdeployqt` copia librerie e plugin nel bundle locale. Non è un installer né una release firmata/notarizzata per la distribuzione. Servono verifica su macchina pulita e inventario completo di dipendenze/licenze prima di distribuirlo. I binari sono esclusi da Git.

**EN:** `macdeployqt` copies runtime libraries/plugins into the local bundle. This is not an installer or a signed/notarized release. Validation on a clean machine and a complete dependency/license inventory are still required before distribution. Binaries are ignored by Git.

## Licensing / Licenze

**IT:** Il codice dell'esperimento resta sotto la licenza WebFence. [MIQT](https://github.com/mappu/miqt/tree/v0.14.0) usa MIT; Qt ha termini separati. L'ipotesi da valutare è Qt Core/Gui/Widgets con collegamento dinamico sotto LGPLv3, rispettando obblighi di avvisi, sorgenti Qt, sostituibilità delle librerie e diritti previsti dalla licenza. Il solo collegamento dinamico non esaurisce gli obblighi. Se non rispettabili, valutare una licenza commerciale Qt prima di distribuirlo. Nessun acquisto o revisione legale eseguito; non usare moduli GPL-only senza decisione esplicita.

**EN:** Experiment code remains under WebFence's license. [MIQT](https://github.com/mappu/miqt/tree/v0.14.0) uses MIT; Qt has separate terms. The candidate approach is dynamically linked Qt Core/Gui/Widgets under LGPLv3, satisfying notices, Qt source, library replacement and other license rights. Dynamic linking alone does not satisfy every obligation. If compliance is not possible, evaluate commercial Qt licensing before distribution. No purchase or legal review has occurred; do not use GPL-only modules without an explicit decision.

Sources / Fonti: [Qt licensing](https://doc.qt.io/qt-6/licensing.html), [LGPL obligations](https://www.qt.io/development/open-source-lgpl-obligations).
