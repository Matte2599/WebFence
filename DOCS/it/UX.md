# Interfaccia e usabilità

[English](../en/UX.md) · [Indice](../README.md)

## Direzione confermata

Richiesta dell'autore del 2026-09-23: GUI semplice, innovativa ma tradizionale, familiare agli utenti degli strumenti Windows del periodo 2015–2020. Comprensibile a chi conosce poco la sicurezza e sufficientemente approfondita per un professionista. È una direzione di interazione, non una copia del marchio o dell'interfaccia di un altro prodotto.

L'app rimane desktop nativa anche su macOS e Linux. Menu, scorciatoie e dialoghi devono rispettare le convenzioni del sistema. L'aspetto grafico non può compensare la mancanza di accessibilità del toolkit.

## Struttura prevista

- Menu tradizionali: File, Progetto, Analisi, Visualizza, Aiuto. Azioni con testo e icona; evitare comandi essenziali riconoscibili soltanto dal simbolo.
- Barra strumenti compatta per il lavoro corrente; al massimo un'azione primaria evidente.
- Navigazione stabile: Progetti, Configurazione analisi, Risultati, Convalida fix, Report. Le sezioni saranno introdotte quando funzionanti.
- Area centrale a tabelle e pannello dettagli ridimensionabile. Ricerca, filtri, ordinamento e selezione multipla solo quando hanno un comportamento definito.
- Barra di stato con avanzamento reale, richieste/budget e stato della copertura. Mai avanzamento fittizio o «sicuro» ricavato da zero risultati.

Usare tipografia leggibile, spaziatura regolare, bordi discreti, colori sobri e gerarchia visiva contenuta. Preferire controlli tradizionali a enormi card, effetti decorativi, gradienti o animazioni pervasive. Tema chiaro/scuro coerente con il sistema; la gravità ha sempre un'etichetta, non solo un colore. Densità regolabile e layout adattabile sono obiettivi successivi, non capacità già disponibili.

## Complessità progressiva

| Percorso guidato | Approfondimento esperto |
| --- | --- |
| Identifica il progetto e chiarisce l'autorizzazione | Scope esatto, esclusioni, origini e identità |
| Propone un profilo con limiti spiegati | Budget, concorrenza, timeout e controlli individuali |
| Spiega cosa è stato osservato e cosa fare | Richieste/risposte redatte, ID regola, confidenza e provenienza |
| Verifica una correzione con esito comprensibile | Prerequisiti del retest, comparabilità, copertura e regressioni |
| Esporta un report con limiti visibili | Dettagli della firma, manifest e verifica della chiave |

Le sezioni «Avanzate» espongono più dettagli senza cambiare automaticamente scope, aggressività, AI o autorizzazioni. Il passaggio deve conservare dati, risultati e stato. La modalità guidata non nasconde errori, controlli saltati o incertezza e non promette l'assenza di vulnerabilità. Distinguere sempre «nessun problema osservato» da «verifica non eseguita».

Errori: spiegare il problema, la conseguenza e l'azione disponibile; dettagli tecnici espandibili. Termini tecnici con breve spiegazione contestuale. Conferme solo per azioni che lo richiedono, come la cancellazione definitiva dei dati del progetto; nessuna sequenza di popup per azioni ordinarie.

## Verifiche e stato

Testare percorsi senza mouse, ordine del focus, nomi e stato dei controlli nello screen reader, copia/selezione del testo delle prove, ridimensionamento e DPI, contrasto e testi IT/EN. Valutare sia utenti inesperti sia professionisti su compiti concreti; non sono ancora stati condotti test di usabilità con partecipanti.

Il prototipo M0 implementa barra comandi, filtri, tabella, pannello prove e lingua. È un laboratorio di fattibilità: menu completi, wizard, sezioni avanzate, workspace di scansione e gestione dei progetti sono **pianificati**. L'aspetto attuale usa il tema Fyne del sistema; non è il design definitivo. Il [resoconto M0](M0-DESKTOP.md) registra un gate di accessibilità ancora aperto.

Aggiornamento M0: [confronto pratico Qt/Fyne](GUI-COMPARISON.md) e [ADR-002 proposto](ADR-002-GUI.md). L’esperimento Qt è separato; scelta dell’autore e gate di adozione ancora aperti.
