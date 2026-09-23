# ADR-001 — Go per il core e il desktop

[English](../en/ADR-001-LANGUAGE.md) · [Indice](../README.md)

Data: 2026-09-23. Stato: scelta iniziale raccomandata; toolkit GUI subordinato al prototipo M0. Requisito confermato dall'autore: applicazione desktop nativa.

## Contesto

Il carico principale previsto è orchestrare richieste HTTP, crawling, regole, persistenza, browser e provider AI. Non esistono misure che dimostrino un collo di bottiglia CPU. Il progetto parte senza codice e deve rimanere gestibile da un autore con contributori occasionali. I pesi LLM e il browser possono consumare più risorse del coordinatore in entrambi i linguaggi.

## Confronto

| Criterio | Go | Rust |
| --- | --- | --- |
| Concorrenza di rete | Goroutine e libreria standard adatte al coordinamento | Runtime asincrono e librerie da comporre esplicitamente |
| Memoria | Gestione automatica con garbage collector; servono limiti e profiling | Ownership e controllo senza GC; `unsafe` e FFI richiedono revisione |
| Manutenzione | Scelta favorevole alla semplicità del core e all'inserimento di contributori | Maggiore investimento iniziale su ownership, async e integrazioni |
| Desktop | Fyne offre una GUI compilata Go, da verificare sul prodotto | Iced è una possibile GUI compilata Rust, da verificare allo stesso modo |
| Distribuzione | Il core può restare semplice; GUI e browser aggiungono dipendenze native | Anche GUI e librerie di sistema richiedono packaging per piattaforma |
| Inferenza locale | Processo esterno, senza portare i pesi nel binario | Stessa separazione possibile; non impone di riscrivere il runtime LLM |

Le valutazioni di manutenzione sono giudizi progettuali, non benchmark. Entrambi i linguaggi possono implementare uno scanner sicuro o insicuro: controlli di scope, autenticazione e limiti non discendono dal linguaggio.

## Decisione

Usare **Go** per dominio, scansione, persistenza e desktop. Prototipare **Fyne** prima di fissare la GUI: finestra nativa e widget renderizzati dal toolkit, senza richiedere una UI in un browser. Questo non implica widget identici a quelli di ogni sistema operativo. Verificare accessibilità con tecnologie assistive, tastiera, tabelle estese, testo delle prove, DPI, selezione/copertura del testo e integrazione del portachiavi.

Evitare un core Go più un secondo core Rust fin dall'inizio. Isolare browser e runtime LLM con contratti di processo, timeout e autorizzazioni. Non promettere un singolo eseguibile statico per l'intero prodotto desktop.

## Criteri per cambiare decisione

Riaprire l'ADR se il prototipo GUI non soddisfa requisiti essenziali, se il gruppo di sviluppo ha un vantaggio operativo dimostrabile in Rust, oppure se profili ripetibili evidenziano costi di GC/CPU non risolvibili con limiti, algoritmi e allocazioni migliori. In quel caso confrontare Go/Fyne e Rust/Iced sullo stesso scenario; una preferenza estetica non giustifica da sola una riscrittura del motore.

Prima implementazione M0: Go 1.27.1 e Fyne 2.8.1 fissati in `go.mod`/`go.sum`; CI legge la versione Go dal modulo. Build e bundle di sviluppo verificati su macOS 26.6.2 ARM64 con Xcode 27. La prova di accessibilità non supera il gate: albero incompleto e interazione assistita non verificata. **Fyne rimane candidato; la valutazione deve essere riaperta su questo limite prima di confermarlo per il prodotto.** Dettagli e alternative da confrontare nel [resoconto M0](M0-DESKTOP.md).

Fonti: [Go FAQ](https://go.dev/doc/faq), [Rust ownership](https://doc.rust-lang.org/nomicon/ownership.html), [Fyne](https://github.com/fyne-io/fyne), [Iced](https://github.com/iced-rs/iced). Verifiche del 2026-09-23.

Aggiornamento M0: [confronto pratico Qt/Fyne](GUI-COMPARISON.md) e [ADR-002 proposto](ADR-002-GUI.md). L’esperimento Qt è separato; scelta dell’autore e gate di adozione ancora aperti.
