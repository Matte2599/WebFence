# Modello delle minacce di WebFence

[English](../en/THREAT-MODEL.md) · [Indice](../README.md)

Stato: requisiti di sicurezza del prodotto, da provare prima dei rilasci. Target, pagine, feed, progetti importati e risposte AI sono non fidati. Un target autorizzato può comunque essere malevolo verso lo scanner.

## Beni e confini

Proteggere credenziali, progetti e dati dei clienti, chiavi di firma, integrità dei report, macchina dell'operatore e disponibilità dei target. Confini principali: GUI/core, core/target, core/browser, core/LLM, import/storage, updater/eseguibili e export/destinatario. Il primo prodotto non implementa isolamento tra tenant: un account locale condiviso non è una separazione tra organizzazioni.

| Minaccia | Controllo previsto | Prova di accettazione |
| --- | --- | --- |
| SSRF e DNS rebinding | Policy centralizzata, IP effettivo verificato, redirect e IPv6 controllati | Destinazione privata dopo redirect o cambio DNS rifiutata |
| Browser fuori scope | Profilo isolato, broker/egress, controllo di subresource e WebSocket | Fixture con canali alternativi non contatta host esclusi |
| Sovraccarico o azione distruttiva | Budget, esclusioni, limiti decompressione, stop globale, nessun form automatico | Slow response e compressione ostile rispettano limiti |
| Furto credenziali | Portachiavi, scope per origine, nessun inoltro cross-origin | Redirect a host estraneo non riceve token o cookie |
| Contaminazione tra progetti | Accesso per ID progetto, cache e contesti AI separati | Stesso URL in due progetti non condivide prove o credenziali |
| Prompt injection | Modello senza autorità, schema e strumenti limitati dal codice | Testo del target non allarga scope né accede a file |
| Report XSS / injection | Testo escapato, rendering senza script o fetch remoti | Prova con markup ostile resta inerte in UI e HTML |
| Import malevolo | Dimensioni limitate, schema, niente traversal/symlink/archivi esplosivi | Import non scrive fuori dalla cartella autorizzata |
| Alterazione report | Manifest e firma standard, fiducia esterna nella chiave | Modifica di file o sostituzione chiave non passa come attendibile |
| Supply chain | Dipendenze e release fissate, SBOM, firma updater, diritti verificati | Update alterato o versione inattesa respinti |
| Esfiltrazione verso provider | Modalità remota esplicita e minimizzazione | AI disabilitata non genera traffico provider |
| Abuso delle chiavi | Firma con accesso ristretto e chiavi separate | Browser/LLM non leggono la chiave privata |

## Identità e credenziali

La GUI opera con i privilegi dell'utente, non richiede amministratore per scansioni HTTP normali. Autenticazioni ai target usano account di test. Login e sessioni sono separati per identità e progetto; una sessione scaduta interrompe i controlli dipendenti. Non registrare password, bearer token, cookie o payload personali integrali.

Se in futuro esiste una API di controllo, aggiungere autenticazione, autorizzazione per risorsa, controlli origin/CSRF dove applicabili e protezione da DNS rebinding verso loopback. Una porta locale non è automaticamente sicura.

## Cosa non garantisce il modello

Una macchina già compromessa può leggere la memoria dell'operatore e alterare il programma. Firma e cifratura non risolvono quel problema. L'isolamento browser va provato su ciascun sistema operativo; i meccanismi differiscono. L'applicazione non può garantire che un target arbitrario non attribuisca effetti a richieste apparentemente innocue.

Prima di introdurre browser, AI, plugin, updater o multiutente, aggiornare questa matrice e i test negativi. Le vulnerabilità del programma seguono [SECURITY.md](../../SECURITY.md), quelle dei target restano riservate ai soggetti autorizzati.

## Evidenze M0 del confine di rete

[ADR-003](ADR-003-TRANSPORT.md) registra le prove sintetiche di DNS variabile/misto, peer effettivo, TLS, redirect, budget e arresto del broker loopback. Sono prove circoscritte, senza scansioni esterne; la matrice del prodotto non è integralmente verificata.

## Evidenze M1 del trasporto pubblico fissato

[ADR-008](ADR-008-PINNED-PUBLIC-TRANSPORT.md) e le [visite controllate](M1-CONTROLLED-CRAWL.md) aggiungono policy di route, rifiuto di IP speciali, grant esatti, rate per broker e verifica a ogni hop. I test simulano il peer pubblico tramite un server loopback; non provano egress di sistema, coordinamento tra processi o target pubblici reali. Restano aperti i controlli della matrice che richiedono browser, credenziali o UI.
