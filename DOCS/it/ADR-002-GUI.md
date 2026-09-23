# ADR-002 — Confronto della GUI nativa

[English](../en/ADR-002-GUI.md) · [Indice](../README.md)

Data: 2026-09-23. **Stato: accettato dall’autore — «Andiamo con QT».** Qt Widgets tramite MIQT diventa la GUI principale; Go resta il linguaggio del core. La scelta tecnica non chiude i gate di qualità M0.

## Contesto e decisione

Il prototipo Fyne non ha superato il gate di accessibilità descritto nel [resoconto M0](M0-DESKTOP.md). L’autore richiede un desktop tradizionale e sobrio, con percorso semplice e dettagli per esperti, e ha richiesto un confronto provato direttamente prima della scelta.

Adottare **Qt Widgets tramite MIQT per Go** come base di sviluppo del desktop. Il codice del laboratorio è trasferito in `internal/desktop`, avviato da `cmd/webfence`; fixture, cataloghi e preferenze restano separati. Fyne e il modulo Qt annidato sono rimossi dalla build corrente e conservati nella cronologia Git. La [guida di sviluppo](DEVELOPMENT.md) contiene i comandi aggiornati. Restano aperti accessibilità, packaging di rilascio e manutenzione dei binding. Nessuno scanner, WebView o motore Rust introdotto.

## Alternative e conseguenze

| Alternativa | Vantaggi | Costi e limiti |
| --- | --- | --- |
| Qt Widgets + MIQT | Tabelle con modello dedicato, controlli desktop, editor di testo semplice e infrastruttura assistiva del toolkit; adatto alla direzione UX richiesta | Go più C++/CGO, librerie dinamiche e packaging; MIQT è un binding comunitario più giovane di Qt; obblighi di licenza Qt separati |
| Continuare con Fyne | Codice già presente, sviluppo centrato su Go, licenza permissiva del toolkit, CI delle build già verificata | Accessibilità nativa insufficiente nella prova corrente; lavoro nel toolkit o attesa di miglioramenti, con tempi non stimati |

La licenza WebFence resta invariata e copre il nostro codice. MIQT è MIT. Per Qt Core/Gui/Widgets la proposta è collegamento dinamico con rispetto della LGPLv3: avvisi, sorgenti Qt, possibilità di sostituzione e diritti richiesti; collegare dinamicamente da solo non basta. La distribuzione richiede un inventario dei moduli e delle dipendenze, una verifica delle condizioni e delle eventuali eccezioni necessarie nei termini del pacchetto. Se incompatibili con la distribuzione voluta, valutare una licenza Qt commerciale. Nessun acquisto o revisione legale è stato effettuato. Non estendere le restrizioni WebFence ai componenti LGPL/MIT. Vedi [licenze Qt](https://doc.qt.io/qt-6/licensing.html) e [obblighi LGPL](https://www.qt.io/development/open-source-lgpl-obligations).

Qt 6.11 dichiara Windows 10 da 1809 e Windows 11 x86-64, macOS da 13 e configurazioni Linux x86-64/ARM64. **Qt 6.12 è annunciata come ultima versione con supporto Windows 10**: prima dell’adozione serve un piano per aggiornamenti e durata del supporto richiesto. La matrice upstream non certifica WebFence né MIQT. [Piattaforme Qt](https://doc.qt.io/qt-6/supported-platforms.html).

## Gate M0 ancora aperti

1. Verificare lettura e azioni con VoiceOver, NVDA e Orca, oltre al solo albero accessibile.
2. Compilare e provare packaging/esecuzione sulle quattro architetture richieste e fissare versioni OS minime; specificare il mantenimento di Windows 10.
3. Verificare selezione/copia, focus, DPI, testo lungo, stabilità e consumi in sessioni prolungate.
4. Verificare licenze e dipendenze distribuibili; testare il bundle su macchina priva della toolchain.
5. Mantenere il dominio separato dai widget; rivedere questo ADR se le verifiche richiedono un cambio di architettura.

Le prove e i relativi limiti sono nel [confronto pratico](GUI-COMPARISON.md). La scelta di Qt non conclude M0; vedi [stato dell’integrazione](QT-DESKTOP.md).

Fonti: [MIQT v0.14.0](https://github.com/mappu/miqt/tree/v0.14.0), [Qt accessibility](https://doc.qt.io/qt-6/accessible.html). Consultate il 2026-09-23.
