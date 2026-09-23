# M0 — Qt come desktop principale

[English](../en/QT-DESKTOP.md) · [Indice](../README.md) · [ADR-002](ADR-002-GUI.md)

Data: 2026-09-23. **Qt scelto dall’autore; integrazione nel percorso principale. M0 resta aperta.**

## Cambiamento

`cmd/webfence` avvia Qt Widgets tramite MIQT 0.14.0. Il workspace è in `internal/desktop`; il modulo principale non dipende più da Fyne. Il laboratorio annidato è stato consolidato, evitando due copie della GUI. Le prove Fyne e il confronto precedente restano documentazione storica e codice nella cronologia Git (`2000c9a`).

Il desktop conserva 10.000 fixture, filtri, selezione, evidenze inerti, copia esplicita e IT/EN. La vista semplice mostra un riepilogo; «Dettagli avanzati» rivela il testo originale. Menu Lingua e scorciatoie Cmd/Ctrl+1 e Cmd/Ctrl+2 cambiano lingua senza alterare ID o prove. Il testo di interfaccia è nei cataloghi condivisi.

La preferenza viene salvata dal pacchetto Go `preferences`, senza dipendere dai binding Qt: file utente `WebFence/ui-language`, contenente solo `it` o `en`. Il salvataggio fallito mostra un avviso e lascia funzionare la sessione. Nessun import o cancellazione delle vecchie preferenze Fyne. Comandi e percorsi sono nella [guida](DEVELOPMENT.md).

Nessuno scanner, networking verso target, CVE, AI, database o report firmato introdotto. La cache dei valori Qt resta limitata alle fixture e alle due lingue; non è un meccanismo di conservazione per dati reali.

## Verifiche di questo task

- Test Go con race detector su fixture, cataloghi e preferenze superati: cambio ripetuto lingua, fallback, dati corrotti e errori I/O.
- Build principale, `go vet ./...` con C++17 e verifica moduli superati su macOS ARM64.
- Self-test Qt offscreen superato: 10.000 righe, identità della selezione, filtri, stato vuoto, azioni lingua persistenti, testo lungo, copia intercettata e avviso di errore recuperabile. Usa una directory temporanea, non la preferenza dell’utente; non modifica gli appunti.
- Bundle principale Qt generato, plist e firma locale ad hoc verificati. GUI aperta: 10.000 righe, filtro ultima riga, selezione ed evidenze avanzate; Cmd+2 cambia in inglese preservando le prove, chiusura/riapertura ripristina inglese, Cmd+1 torna in italiano e Cmd+O carica gli esempi. Nessuna firma Developer ID/notarizzazione.
- **CI finale superata:** [run 35853356437](https://github.com/Matte2599/WebFence/actions/runs/35853356437), codice `fa8bd32`. Tutti e quattro i job completati: build, test Go con race detector, vet e self-test Qt; anche bundle macOS. macOS 15 ARM64 usa Qt 6.11.2, Windows Server 2022 x86-64 usa MSYS2 UCRT64/Qt 6.11.2-2, Ubuntu 24.04 x86-64 e ARM64 usano Qt 6.4.2. Questi esiti non equivalgono a GUI assistiva provata su Windows 10/11 o Debian desktop.

Il packaging riusa gli stessi flag C++ della build, evitando una seconda compilazione dei binding. In CI macOS si usa `-O0 -g0`; la prima compilazione resta lunga (circa 11 minuti nella run finale), mentre il confezionamento successivo è terminato in circa 40 secondi. Sono osservazioni di questa run, non benchmark del prodotto. Build locale predefinita `-O2 -g` verificata separatamente.

## Tastiera e verifica accessibilità — 2026-09-23

Il menu **Visualizza** offre accesso diretto a ricerca, risultati ed evidenze; File precede Visualizza e Lingua. Ricerca e gravità hanno etichette visibili associate ai controlli. Tab esce dalla tabella (le frecce navigano le righe) e dal testo delle evidenze; Shift+Tab torna indietro. L’ordine Tab segue barra comandi → filtri → tabella → riepilogo → evidenze/copia quando visibili. Nascondere i dettagli mentre il focus è nelle evidenze o sul pulsante copia riporta il focus a «Dettagli avanzati».

| Azione | Windows/Linux | macOS |
| --- | --- | --- |
| Ricerca, selezionando il filtro corrente | Ctrl+F | Cmd+F |
| Risultati; prima riga se manca una selezione | F6 | F6 (eventualmente Fn+F6) |
| Apri e leggi evidenze | Ctrl+Shift+E | Cmd+Shift+E |
| Mostra/nascondi dettagli avanzati | Ctrl+Shift+D | Cmd+Shift+D |

Il cambio lingua invia notifiche di dati/intestazioni modificate, senza reset del modello. Conserva la riga corrente e la selezione del testo nelle evidenze. I filtri che cambiano le righe continuano a usare il reset previsto da Qt.

Il self-test verifica focus delle azioni, sincronizzazione menu/checkbox, assenza di dati obsoleti, selezione del testo e interfaccia `QAccessibleTableInterface` (dimensioni e ID dell’ultima cella dopo caricamento, traduzione, filtri, ripristino e svuotamento). In modalità offscreen richiede l’attivazione dell’accessibilità Qt solo per il test: senza un lettore attivo, Qt può non invalidare la cache delle celle sui reset. Su Qt 6.4 offscreen `SetActive` notifica gli osservatori ma non attiva il bridge: il test registra `qt_accessibility_active=false` e invia un reset esplicito alla cache dell’interfaccia prima delle letture. In quel caso verifica dati e dimensioni, non la consegna automatica degli eventi. Su Qt 6.11.2 locale l’attivazione è risultata vera. La GUI normale conserva l’attivazione gestita dal sistema. Il test invia inoltre eventi Tab/Shift+Tab ai widget Qt per controllare uscita dalla tabella, riepilogo, evidenze, copia e salto dei controlli nascosti. Queste prove non misurano il bridge macOS né ciò che viene annunciato da VoiceOver.

Il nodo tabella assente dopo un filtro è stato riprodotto nell’ispezione AX macOS prima della modifica. Il nuovo percorso da tastiera e le prove interne Qt non bastano a dichiarare risolto quel limite. Per chiudere il gate occorre ripetere con VoiceOver, NVDA e Orca: carica → filtra → leggi intestazioni/riga → apri evidenze → cambia lingua → elimina il filtro → svuota; verificare annunci, focus, testo originale e assenza di celle obsolete. Registrare versioni OS/Qt/lettore, DPI e risultati per piattaforma.

Riferimenti implementativi: [focus Qt](https://doc.qt.io/qt-6/focus.html), [notifiche del modello](https://doc.qt.io/qt-6/qabstractitemmodel.html#dataChanged), [reset nella sorgente Qt 6.11.2](https://github.com/qt/qtbase/blob/v6.11.2/src/widgets/itemviews/qabstractitemview.cpp). I valori restituiti da `Index()` e `TextCursor()` in MIQT hanno finalizer automatici: non liberare manualmente lo stesso valore senza prima rimuovere il finalizer.

## Gate aperti

La scelta dell’autore risolve la decisione sul toolkit, non l’accessibilità del prodotto. Restano da isolare il nodo tabella non sempre esposto e le azioni sul menu viste nel [confronto](GUI-COMPARISON.md); servono VoiceOver, NVDA, Orca e prove DPI. Le scorciatoie sono un percorso aggiuntivo, non una certificazione assistiva.

Restano installer Windows/Linux, firme e notarizzazione, esecuzione su Windows 10/11 e Debian desktop, macchina macOS senza toolchain, versioni minime e manutenzione Windows 10 dopo Qt 6.12. Licenza WebFence invariata; inventario e adempimenti Qt/MIQT da completare prima della distribuzione. Nessun acquisto o licenza commerciale sottoscritta.
