# M0 — Matrice di verifica e chiusura

[English](../en/M0-VALIDATION.md) · [Roadmap](../ROADMAP.md) · [Indice](../README.md)

Aggiornamento: 2026-09-23. **M0 aperta.** Questo registro riguarda criteri di uscita, non percentuali derivate dal numero di task. «Parziale» significa che esistono prove positive ma manca almeno una verifica richiesta. Un test offscreen non sostituisce un collaudo assistivo o una macchina pulita.

| ID | Criterio | Stato ed evidenza | Necessario per chiudere |
| --- | --- | --- | --- |
| M0-01 | Desktop nativo, tabella ampia, prove, IT/EN | Parziale: Qt principale, 10.000 fixture, filtri, lingua persistente, testo lungo, menu/focus e self-test sui quattro target. [Stato Qt](QT-DESKTOP.md). | Collaudo assistivo, DPI visivo e stabilità prolungata descritti sotto. |
| M0-02 | Build e packaging sui target richiesti | Parziale: bundle macOS, [pacchetti Debian amd64/arm64 e ZIP Windows](ADR-007-PACKAGING.md) con avvio fuori toolchain verificato. | Inventario completo/notices/sorgenti, sistemi minimi, desktop Windows 10/11 e Debian reali; rischi residui espliciti. |
| M0-03 | GUI, SQLite, JWS, portachiavi | Completato: Qt, SQLite/JWS e [portachiavi nativo](ADR-005-CREDENTIALS.md) selezionati; [CI quattro target verde](https://github.com/Matte2599/WebFence/actions/runs/35870058803), inclusa negazione con token anonimo Windows. | Conservare regressioni; integrazione GUI e gestione chiavi restano successive. |
| M0-04 | Versioni, scheletro, cataloghi, CI | Completato sul codice `8fd55b9`: [CI sei job verdi](https://github.com/Matte2599/WebFence/actions/runs/35879464771), inclusi pacchetti Debian e Windows con runtime separato. | Ripetere sul commit conclusivo di M0 e verificare il remoto. |
| M0-05 | Laboratorio sintetico e primi test scope/rete | Completato: [ADR-003](ADR-003-TRANSPORT.md), CI quattro target, solo loopback. | Mantenere regressioni verdi; trasporto di produzione in M1. |
| M0-06 | Canale privato, revisione legale e contributori | Parziale: Private vulnerability reporting abilitato e verificato nella UI GitHub; [SECURITY](../../SECURITY.md) aggiornato. | [Revisione legale qualificata del testo](LEGAL-REVIEW.md) e accordo contributori prima delle rispettive aperture; non dichiararli approvati senza riscontro. |

## Collaudo desktop riproducibile

Registrare commit, hash del pacchetto, sistema/versione/architettura, Qt, lettore/versione, scala e preferenza di navigazione tastiera. Usare solo fixture integrate; nessun target esterno. Registrare esito e difetto per ciascun passo in entrambe le lingue. Salvare eventuali screenshot/audio solo con dati sintetici.

1. Avviare il pacchetto da un account di prova senza Go o toolchain Qt nel PATH. Finestra e comandi devono essere utilizzabili, senza richieste di rete o privilegi amministrativi inattesi.
2. Attivare **VoiceOver** su macOS, **NVDA** su Windows 10 e 11, **Orca** su un desktop Debian/derivato. Verificare nome/ruolo/stato dei controlli, nomi delle colonne, riga selezionata e lettura delle prove. L'albero AX da solo non basta: registrare cosa viene effettivamente annunciato. Non modificare permanentemente le preferenze personali per far passare la prova.
3. Con sola tastiera caricare fixture (Ctrl/Cmd+O), cercare `DEMO-10000` (Ctrl/Cmd+F), passare ai risultati (F6), leggere la prova (Ctrl/Cmd+Shift+E), selezionare/copiare testo. La stringa con tag/script resta testo inerte. Provare Tab e Shift+Tab, menu e pulsante copia, senza trappole di focus.
4. Cambiare lingua (Ctrl/Cmd+1/2) mantenendo riga e testo selezionato. Applicare filtro senza risultati: nessuna prova precedente rimane presentata come selezionata. Mostrare/nascondere dettagli (Ctrl/Cmd+Shift+D) senza perdere il focus. Svuotare, ricaricare, chiudere e riaprire: lingua ripristinata, fixture non persistite.
5. Provare scala 100%, 150%, 200%, finestra ridotta e testo lungo: etichette leggibili, comandi raggiungibili, scrolling e splitter funzionanti, nessun contenuto essenziale tagliato. Passaggio tra monitor con scale differenti quando disponibile. I self-test offscreen locali con `QT_SCALE_FACTOR=1.5` e `2` sono passati; non verificano la resa visiva né il cambio monitor.
6. Sessione controllata di almeno 30 minuti con cicli di carica/filtro/lingua/svuota e lettura prove. Annotare tempo, memoria iniziale/finale e andamento, blocchi, crash o crescita non stabilizzata. Nessun limite hardware o benchmark dichiarato prima della misura.

Problemi già osservati su macOS: nodo tabella AX intermittente e Tab nativo che salta copia con la preferenza attuale; le scorciatoie e i test Qt non bastano a chiudere tali punti. La disponibilità delle macchine Windows/NVDA e Debian/Orca è stata chiesta all'autore; nessun esito è presunto.

## Chiusura finale

Completare prima i punti aperti con prove ripetibili. Eseguire poi suite Go/race, fuzzing limitato, vet, verifica moduli/advisory, build, self-test e collaudo dei pacchetti applicabili. Controllare traduzioni, collegamenti, licenze/notices e `git diff --check`. Aggiornare questo registro, ADR, README, memoria e roadmap; commit/push e CI sul codice conclusivo. Riportare esattamente i limiti residui: documentarli non equivale automaticamente a soddisfare un gate.

Ricognizione packaging: il [bundle locale esaminato](M0-PACKAGING.md) contiene 19 Mach-O con minimo macOS 26 e 9 con minimo 14. Non prova supporto alle versioni precedenti; scelta del minimo richiesta all’autore. Inventario iniziale e SBOM upstream individuati, pacchetto conforme e prova su macchina pulita ancora aperti.

Prova GUI del 2026-09-23: layout standard e al 150% ispezionati, ma la copia al 150% termina con SIGSEGV durante reset del filtro seguito da cambio lingua tramite accessibilità. Riprodotto con fixture selezionata; traccia nativa in raccolta, causa ancora da isolare. Nessuna chiusura del gate stabilità/DPI.

Aggiornamento: [correzione Cocoa adottata](ADR-006-QT-COCOA.md) nel bundle di sviluppo. La sequenza del crash passa al 150% e 200%; al 200% resta la tabella troppo compressa, oltre a celle AX non sempre esposte e collaudo VoiceOver/stabilità prolungata da completare.

Layout 200% successivamente corretto e provato: due righe complete e ultima colonna raggiungibile da tastiera, prove scorrevoli. Self-test della finestra compatta aggiunto e superato localmente a scala 1/1,5/2. Restano collaudo lettori, più monitor e stabilità misurata.

Procedura strumentata per la sessione prolungata: [M0-STABILITY](M0-STABILITY.md). Prove brevi verificate in CI; sessione Cocoa da 30 minuti avviata, esito ancora aperto.
