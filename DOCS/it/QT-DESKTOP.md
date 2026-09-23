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
- Script bundle principale aggiornato per Qt. Verifica del bundle e interazione nativa da registrare al termine.
- CI sostituita con build, test puri e self-test Qt su macOS ARM64, Ubuntu x86-64/ARM64 e Windows Server x86-64 con MSYS2 UCRT64. Esiti remoti da registrare dopo il push; la presenza del workflow non prova successo.

## Gate aperti

La scelta dell’autore risolve la decisione sul toolkit, non l’accessibilità del prodotto. Restano da isolare il nodo tabella non sempre esposto e le azioni sul menu viste nel [confronto](GUI-COMPARISON.md); servono VoiceOver, NVDA, Orca e prove DPI. Le scorciatoie sono un percorso aggiuntivo, non una certificazione assistiva.

Restano installer Windows/Linux, firme e notarizzazione, esecuzione su Windows 10/11 e Debian desktop, macchina macOS senza toolchain, versioni minime e manutenzione Windows 10 dopo Qt 6.12. Licenza WebFence invariata; inventario e adempimenti Qt/MIQT da completare prima della distribuzione. Nessun acquisto o licenza commerciale sottoscritta.
