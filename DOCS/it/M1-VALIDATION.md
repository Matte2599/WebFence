# M1 — Verifica della alpha controllata

[English](../en/M1-VALIDATION.md) · [Roadmap](../ROADMAP.md) · [Piattaforme target](M1-SUPPORT-POLICY.md) · [Analisi legale interna](M1-LEGAL-ASSESSMENT.md) · [Sviluppo](DEVELOPMENT.md)

**M1 — alpha tradizionale controllata: chiusa per decisione dell'autore del 25 settembre 2026.** Il criterio di uscita è il flusso funzionante e verificato con test locali sul codice, fixture proprie e CI multipiattaforma; le prove assistive, multimonitor, su workstation reali e la consulenza legale esterna non sono più gate di M1. WebFence offre una scansione HTTP(S) GET limitata da scope e budget, con un solo controllo osservativo (`HTTP-XCTO-001`), risultati redatti persistenti e una finestra Qt bilingue. La chiusura **non** dichiara equivalenza con scanner commerciali, audit completo, accessibilità certificata o release pronta alla distribuzione.

## Criteri tecnici

| Criterio | Stato | Prova e limite |
| --- | --- | --- |
| Progetti, autorizzazione, scope, trasporto, budget, cancellazione | Implementato e testato localmente | `internal/project`, `internal/scope`, `internal/transport`, run gestite e `RunCrawl`; preflight dei seed, IP esatti fissati, policy GET/prefissi, rate e revoca. Un lock di file esclude un secondo processo sullo stesso archivio; una sola run crawler persistente è ammessa per istanza. Il consenso dichiarato non prova la titolarità. |
| Discovery HTTP(S), controllo a basso impatto e prove redatte | Implementato e testato localmente | Link e form HTML osservati; i link vengono visitati solo con opt-in, i form non vengono inviati. `HTTP-XCTO-001` distingue `nosniff_present`, `nosniff_absent`, non conclusivo e non applicabile. Test sintetici includono casi con e senza header, redirect, esclusioni e limiti. Nessun contenuto grezzo è nel ledger. |
| Persistenza, recupero, limiti, eliminazione base | Implementato e testato localmente | SQLite v4 migra v1/v2/v3, registra run e visite redatte, marca `interrupted` una run lasciata aperta al riavvio, limita il file principale a 64 MiB e le run a 100 per progetto, offre `Backup` tramite `VACUUM INTO` in un file nuovo e privato, elimina progetto e righe figlie. Il test di recupero simula l'interruzione lasciando una run non finalizzata; non dimostra lo spegnimento hardware. WAL e backup esterni non sono cancellati in modo forense. |
| Desktop IT/EN: avanzamento, risultati, copertura incompleta | Implementato e testato con Qt offscreen su macOS | La finestra M1 crea/seleziona/elimina progetti, richiede conferma dell'autorizzazione, configura una origine/un seed, policy e limiti, avvia/annulla, mostra visite e risultati riaperti. Avanzamento = visite completate, **non** percentuale del sito. La copertura incompleta o l'interruzione resta visibile. Il self-test nativo usa un server locale posseduto dal test. |
| Uscita tecnica progetto → scansione lab → risultati → riavvio | Verificata localmente | Test storage/scanner riaprono il DB e leggono visite ed esiti; il self-test Qt percorre creazione → scansione loopback → risultato bilingue → eliminazione. Nessun LLM richiesto. I test non hanno contattato target esterni. |

Il database salva ID progetto, dichiarazione e origini in chiaro, più codici/conteggi redatti delle visite; non salva URL visitati, query, header, body o credenziali. La cancellazione base rimuove le righe logiche del progetto, non eventuali backup separati o tracce forensi. La quota SQLite limita il file principale; WAL e file di backup possono occupare spazio aggiuntivo temporaneo. La modalità pubblica richiede IP pubblici fissati dall'operatore e non è stata provata su una rete pubblica reale. L'interfaccia M1 supporta una sola origine e un seed per scansione; il core supporta più origini/seed entro i limiti documentati.

## Test complessivo di chiusura

La [suite di sviluppo](DEVELOPMENT.md) include `go mod verify`, `go test -race` sui pacchetti core, `go test ./...`, `go vet ./...`, build Qt e `--self-test` offscreen. I risultati puntuali vanno riferiti al commit testato; la CI del branch e del commit integrato su `main` va verificata senza creare commit solo per annotarla. Il test nativo non equivale a un lettore di schermo o a una prova su Windows/Ubuntu reali.

**Evidenza tecnica precedente:** il 25 settembre 2026 su Apple Silicon/macOS 26.6.2 sono passati `go mod verify`, la suite core con `-race`, `go test ./...`, `go vet ./...`, build Qt, `--self-test` offscreen (incluso il flusso M1 con server loopback), `--soak-test=10s` (4 cicli), `git diff --check` e 79 test Python di supporto (4 salti legati all'ambiente). La prova di lock usa anche un secondo processo; il caso header assente/presente viene letto dopo riavvio. La [CI del branch](https://github.com/Matte2599/WebFence/actions/runs/36119542128) e la [CI dello squash `12b3f5e`](https://github.com/Matte2599/WebFence/actions/runs/36120434350) sono risultate verdi (7/7 job ciascuna). Questi esiti non provano i gate osservativi sotto.

## Prove e decisioni rinviate oltre M1

L'autore ha esplicitamente sostituito il precedente requisito di chiusura: M1-H01…H08 **non sono marcati come superati** e non impediscono più la chiusura della milestone. Restano condizioni di qualità, compatibilità, distribuzione o contributi per le attività pertinenti; procedure nel [piano di collaudo](M1-PREREQUISITES.md). Le [versioni minime target](M1-SUPPORT-POLICY.md) sono state fissate, non collaudate tutte su hardware reale.

| Gate | Stato e motivo |
| --- | --- |
| M1-H01 VoiceOver | Aperto: manca ascolto e navigazione IT/EN da parte di una persona sul Mac Apple Silicon. |
| M1-H02 NVDA | Aperto: manca prova assistiva sul Windows 10 reale disponibile; versione/build e NVDA non ancora registrati. |
| M1-H03 Orca | Non eseguito: manca una persona su desktop Ubuntu 24.04 reale. |
| M1-H04 Monitor misti | Aperto: manca prova su due monitor attivi con scale diverse. |
| M1-H05 Workstation supportate | Non eseguito: Mac e Windows 10 disponibili per prove guidate, ma mancano prove complete su workstation reali; CI e container provano i pacchetti, non l'uso indipendente. |
| M1-H06 Versioni minime | Decisione chiusa: macOS 26 ARM64, Windows 10 1809+ x86-64 e Ubuntu 24.04 LTS x86-64/ARM64. Prove sulle versioni minime reali non eseguite. |
| M1-H07 Revisione legale | [Analisi interna M1](M1-LEGAL-ASSESSMENT.md) completata sul testo della licenza; nessun parere professionale o approvazione di distribuzione acquisiti. |
| M1-H08 Accordo contributori | Non attivo: resta necessario prima di integrare contributi sostanziali destinati a rilicenza commerciale. |

Non dedurre esiti assistivi o di distribuzione da CI, emulazione, ispezione del codice o Qt offscreen. La chiusura M1 riguarda l'alpha tecnica; per dichiarazioni di supporto, accessibilità e distribuzione si devono risolvere i rischi pertinenti, senza spacciarli per già superati.
