# M1 — Primo blocco: progetto e dichiarazione di autorizzazione

[English](../en/M1-PROJECT-AUTHORIZATION.md) · [Roadmap](../ROADMAP.md) · [Architettura](ARCHITECTURE.md)

`internal/project` introduce il modello di progetto **in memoria e senza rete**. Questo primo blocco prepara lo snapshot del perimetro per una run. La [persistenza SQLite](M1-PROJECT-STORE.md) è un componente separato; scansione e schermata di gestione progetti non sono ancora realizzate.

## Contratto implementato

`project.New(Draft)` richiede ID e nome progetto, nome dichiarato del proprietario del target, riferimento descrittivo all'autorizzazione, conferma esplicita dell'operatore, scadenza futura e da una a 32 origini HTTP(S) esatte. Il riferimento è un'etichetta, non il documento né una credenziale; nessuno di questi campi viene inviato in rete. La conferma registra un'affermazione dell'operatore: **WebFence non verifica autonomamente proprietà o validità legale dell'autorizzazione**.

Un input parziale o ambiguo viene rifiutato. Le origini vengono validate da `internal/scope`, rese canoniche e copiate; duplicati equivalenti sono rifiutati. Il progetto e la lista delle origini non espongono strutture mutabili. `BeginRun()` crea uno snapshot immutabile solo se il permesso non è scaduto. `RunScope.CheckOrigin(url)` ricontrolla la scadenza a ogni chiamata e applica schema, host e porta esatti. Il valore zero nega l'uso. Gli errori hanno codici stabili senza URL, proprietario o riferimento.

Una nuova dichiarazione completa può essere validata con `Project.ReviseAuthorization(AuthorizationDraft)`: conserva ID e nome, incrementa la revisione e lascia intatti progetto e `RunScope` precedenti. `Project.Revision()` e `RunScope.Revision()` identificano la versione. Lo [store v2](M1-PROJECT-STORE.md) salva le revisioni, rileva modifiche concorrenti e restituisce solo quella corrente per nuove run. Le vecchie run mantengono il loro limite originale; il loro arresto dopo una revoca non è ancora automatizzato.

Questo controllo è **solo il livello di origine**. Non limita ancora metodi, percorsi, IP/CIDR, DNS, budget o frequenza, e non apre socket. Un suo esito positivo non basta per effettuare una richiesta. Il [blocco M1 successivo](M1-AUTHORIZED-LAB.md) lo collega al broker solo loopback; la GUI resta scollegata. Nessun sito esterno è stato contattato.

## Verifiche e prossimi blocchi

I test sintetici coprono richiesta ammessa, schema/porta/sottodominio esclusi, scadenza al confine esatto e dopo l'avvio, valore zero, configurazioni invalide, duplicati canonici, revisione e impossibilità di allargare uno snapshot modificando le slice del chiamante o rinnovando il progetto. La CI multipiattaforma include il pacchetto nella suite race.

Restano per M1: integrazione desktop dello store, revoca e arresto delle run attive, migrazioni future, policy di metodi/percorsi/esclusioni e di destinazione IP, broker di produzione con budget/rate limit condivisi, UI IT/EN, discovery e prove redatte. L'integrazione per hop nel solo laboratorio è descritta separatamente. I [collaudi umani ereditati da M0](M0-VALIDATION.md) restano prerequisiti di validazione M1.
