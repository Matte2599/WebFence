# M1 — Verifica della alpha controllata

[English](../en/M1-VALIDATION.md) · [Roadmap](../ROADMAP.md) · [Prerequisiti umani](M1-PREREQUISITES.md) · [Sviluppo](DEVELOPMENT.md)

La **parte tecnica automatizzabile di M1** è implementata. WebFence offre una scansione HTTP(S) GET limitata da scope e budget, con un solo controllo osservativo (`HTTP-XCTO-001`), risultati redatti persistenti e una finestra Qt bilingue. Questo non è un prodotto equivalente a scanner commerciali, un audit completo o una release pronta alla distribuzione. La **chiusura formale di M1 resta aperta** finché i gate umani sotto non sono documentati e accettati.

## Criteri tecnici

| Criterio | Stato | Prova e limite |
| --- | --- | --- |
| Progetti, autorizzazione, scope, trasporto, budget, cancellazione | Implementato e testato localmente | `internal/project`, `internal/scope`, `internal/transport`, run gestite e `RunCrawl`; preflight dei seed, IP esatti fissati, policy GET/prefissi, rate e revoca. Un lock di file esclude un secondo processo sullo stesso archivio; una sola run crawler persistente è ammessa per istanza. Il consenso dichiarato non prova la titolarità. |
| Discovery HTTP(S), controllo a basso impatto e prove redatte | Implementato e testato localmente | Link e form HTML osservati; i link vengono visitati solo con opt-in, i form non vengono inviati. `HTTP-XCTO-001` distingue `nosniff_present`, `nosniff_absent`, non conclusivo e non applicabile. Test sintetici includono casi con e senza header, redirect, esclusioni e limiti. Nessun contenuto grezzo è nel ledger. |
| Persistenza, recupero, limiti, eliminazione base | Implementato e testato localmente | SQLite v4 migra v1/v2/v3, registra run e visite redatte, marca `interrupted` una run lasciata aperta al riavvio, limita il file principale a 64 MiB e le run a 100 per progetto, offre `Backup` tramite `VACUUM INTO` in un file nuovo e privato, elimina progetto e righe figlie. Il test di recupero simula l'interruzione lasciando una run non finalizzata; non dimostra lo spegnimento hardware. WAL e backup esterni non sono cancellati in modo forense. |
| Desktop IT/EN: avanzamento, risultati, copertura incompleta | Implementato e testato con Qt offscreen su macOS | La finestra M1 crea/seleziona/elimina progetti, richiede conferma dell'autorizzazione, configura una origine/un seed, policy e limiti, avvia/annulla, mostra visite e risultati riaperti. Avanzamento = visite completate, **non** percentuale del sito. La copertura incompleta o l'interruzione resta visibile. Il self-test nativo usa un server locale posseduto dal test. |
| Uscita tecnica progetto → scansione lab → risultati → riavvio | Verificata localmente | Test storage/scanner riaprono il DB e leggono visite ed esiti; il self-test Qt percorre creazione → scansione loopback → risultato bilingue → eliminazione. Nessun LLM richiesto. I test non hanno contattato target esterni. |

Il database salva ID progetto, dichiarazione e origini in chiaro, più codici/conteggi redatti delle visite; non salva URL visitati, query, header, body o credenziali. La cancellazione base rimuove le righe logiche del progetto, non eventuali backup separati o tracce forensi. La quota SQLite limita il file principale; WAL e file di backup possono occupare spazio aggiuntivo temporaneo. La modalità pubblica richiede IP pubblici fissati dall'operatore e non è stata provata su una rete pubblica reale. L'interfaccia M1 supporta una sola origine e un seed per scansione; il core supporta più origini/seed entro i limiti documentati.

## Test complessivo di chiusura tecnica

Eseguire dal commit candidato la [suite di sviluppo](DEVELOPMENT.md): `go mod verify`, `go test -race` sui pacchetti core, `go test ./...`, `go vet ./...`, build Qt e `--self-test` offscreen. Verificare anche la CI del branch e del commit integrato su `main`; i risultati puntuali vanno riferiti al commit effettivamente testato, senza creare commit solo per annotare la CI. Il test nativo non sostituisce un lettore di schermo o una prova su Windows/Debian reali.

**Esecuzione locale del 25 settembre 2026:** su Apple Silicon/macOS 26.6.2 sono passati `go mod verify`, la suite core con `-race`, `go test ./...`, `go vet ./...`, build Qt, `--self-test` offscreen (incluso il flusso M1 con server loopback), `--soak-test=10s` (4 cicli), `git diff --check` e 79 test Python di supporto (4 salti legati all'ambiente). La prova di lock usa anche un secondo processo; il caso header assente/presente viene letto dopo riavvio. La CI multipiattaforma deve essere valutata sul commit di branch e su quello finale di `main`; questo paragrafo non ne anticipa l'esito.

## Gate umani ancora aperti

Tutti i seguenti sono **richiede intervento umano — prerequisito per dichiarare M1 chiusa**; la procedura e il criterio di prova sono nel [piano M1](M1-PREREQUISITES.md).

| Gate | Stato e motivo |
| --- | --- |
| M1-H01 VoiceOver | Aperto: manca ascolto e navigazione IT/EN da parte di una persona sul Mac Apple Silicon. |
| M1-H02 NVDA | Aperto: manca prova assistiva sul Windows 10 reale disponibile; versione/build e NVDA non ancora registrati. |
| M1-H03 Orca | Aperto: mancano desktop Debian reale e persona che ascolti Orca. |
| M1-H04 Monitor misti | Aperto: manca prova su due monitor attivi con scale diverse. |
| M1-H05 Workstation supportate | Aperto: Mac e Windows 10 sono disponibili per prove guidate, ma non risultano prove complete; mancano Windows 11 e Debian x86-64/ARM64 reali. CI e container non le sostituiscono. |
| M1-H06 Versioni minime | Aperto: macOS 26 scelto ma da provare sulla versione minima; minimi Windows/Debian da definire e provare. |
| M1-H07 Revisione legale | Aperto: consulente non ancora incaricato; licenza, notices e dossier non approvati professionalmente. |
| M1-H08 Accordo contributori | Aperto: testo e processo di accettazione da revisionare con il consulente e approvare dall'autore. |

Non segnare questi gate come superati sulla base di CI, emulazione, ispezione di codice o test Qt offscreen. Sono necessari prima di dichiarazioni di supporto/distribuzione pertinenti; lo sviluppo locale può continuare.
