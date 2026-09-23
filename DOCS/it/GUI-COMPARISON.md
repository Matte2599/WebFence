# M0 — Confronto pratico Fyne e Qt Widgets

[English](../en/GUI-COMPARISON.md) · [Indice](../README.md) · [Proposta ADR-002](ADR-002-GUI.md)

Data: 2026-09-23. **Confronto sperimentale; toolkit definitivo non scelto.** Il programma principale resta Go/Fyne. Il laboratorio [Qt/MIQT](../../experiments/qt/README.md) è un modulo separato e non esegue richieste di rete.

## Scenario e ambiente

Entrambi usano gli stessi 10.000 record di `internal/demo` e i cataloghi IT/EN del progetto. Il laboratorio Qt aggiunge una vista semplice con riepilogo e una casella per mostrare prove avanzate. Non è il design definitivo. La lingua Qt vale per la sessione; il prototipo Fyne salva invece la preferenza.

Host: macOS 26.6.2 ARM64, Xcode 27, Go 1.27.1. Fyne 2.8.1; candidato MIQT 0.14.0 con Qt 6.11.2 da Homebrew (`qtbase`, `pkgconf`). Qt richiede C++17 esplicito tramite `CGO_CXXFLAGS`. Non sono state eseguite scansioni di siti.

## Risultati verificati

| Prova | Fyne | Qt Widgets / MIQT |
| --- | --- | --- |
| 10.000 righe, filtri, selezione ultima riga | Test e GUI verificati nel primo task | Self-test e caricamento/filtro nella GUI nativa superati |
| Cambio IT/EN e identità della selezione | Verificato | Self-test superato; percorso completo del selettore nativo non convalidato |
| Stato vuoto e nessuna prova obsoleta | Test superati | Self-test superato |
| Evidenze come testo inerte | Verificato | Test superato; 624.640 byte restituiti integri dall’editor |
| Tastiera e testo selezionabile | Gate assistivo non superato | Navigazione alla riga con Tab/frecce e selezione del testo con Cmd+A osservate; tentativo di digitazione non modifica le prove |
| Albero accessibile macOS | Finestra/menu nella build normale; bridge opzionale incompleto | Pulsanti, ricerca, selettori, tabella, celle del risultato filtrato e testo delle prove visibili |
| Azioni tramite strumento assistivo | Attivazione problematica | Caricamento, filtro e dettagli avanzati funzionano; click su cella/menu lingua non sufficiente nella prova |
| Bundle locale | Avvio verificato | `macdeployqt`, avvio e verifica firma ad hoc locale superati; non firma Developer ID/notarizzazione |

L’ispezione Qt non dimostra accessibilità completa. Durante ulteriori tentativi sul menu lingua lo strumento ha restituito `noWindowsAvailable`; la successiva acquisizione ha mostrato un’istanza vuota. Causa non isolata: non attribuire automaticamente un crash a Qt né considerare superato il percorso. Servono riproduzione controllata e VoiceOver reale. Non sono stati provati NVDA o Orca. Con tutte le 10.000 righe caricate lo strumento ha mostrato il nodo tabella senza enumerare le celle; le celle sono state osservate dopo il filtro a una riga.

## Test e misure locali

- Applicazione principale: `go test -tags ci -race ./...` superato; non include il modulo annidato Qt.
- Laboratorio: `go mod verify`, `go vet ./...` con C++17 e `QT_QPA_PLATFORM=offscreen ... --self-test` superati. Il self-test utilizza widget e modello reali sul thread OS proprietario; non simula uno screen reader.
- Gestione dei valori del modello: MIQT restituisce puntatori a `QVariant` che Qt copia. L’esperimento conserva valori riutilizzabili e li libera dopo la distruzione del modello. Due passaggi di 10.000 letture non fanno crescere la cache al secondo passaggio. Questa è una cache del corpus finito, non un modello di conservazione per futuri dati di clienti; non è un soak test o una misura RSS.
- Ultima esecuzione locale offscreen: caricamento con elaborazione eventi **3,746 ms**, filtro dell’ultima riga **4,377 ms**. Una singola osservazione sul prototipo, non percentile, latenza di presentazione o confronto prestazionale con Fyne.
- Prima build Qt riuscita dopo la configurazione C++17: **97,48 s**. Include la compilazione dei binding; non è una misura ripetibile a cache comparabili.
- Spazio su disco rilevato con `du -sh`: bundle Fyne **31 MiB**, Qt **88 MiB**. Artefatti locali diversi, arrotondati, senza ottimizzazione comune; non sono RAM, download compressi o dimensioni finali di release.
- Il backend offscreen ha emesso avvisi sui font e `propagateSizeHints`; il self-test è terminato con codice 0.

## Matrice e limiti

macOS ARM64 è l’unico desktop Qt provato direttamente. È stato aggiunto un workflow dedicato per compilazione e self-test offscreen su Ubuntu 24.04 x86-64/ARM64; **l’esito remoto deve essere registrato dopo l’esecuzione**. Il workflow principale Fyne non verifica Qt. Nessuna esecuzione Qt su Windows 10/11 o Debian desktop è stata effettuata; compatibilità dichiarata dai fornitori e build CI non sostituiscono queste prove.

Restano DPI multipli, accessibilità con lettori reali, stabilità prolungata, packaging Windows/Linux, bundle macOS su macchina pulita, inventario delle licenze e versioni OS minime. Non distribuire il bundle come prodotto pronto.

## Valutazione

**Raccomandazione: proseguire la validazione di Qt Widgets mantenendo Go**, perché il prototipo offre già una base assistiva e controlli adatti alla UX richiesta. L’adozione resta condizionata ai gate dell’[ADR-002](ADR-002-GUI.md). Il costo è maggiore complessità di compilazione, distribuzione e gestione dei binding; la maturità di Qt non va attribuita automaticamente a MIQT. Fyne resta più semplice per il codice Go, ma il suo limite assistivo corrente richiederebbe lavoro aggiuntivo senza una soluzione già verificata.

Licenze e vincolo futuro Windows 10 sono nell’ADR. Le osservazioni sopra derivano dalle prove locali, non da promesse dei toolkit.
