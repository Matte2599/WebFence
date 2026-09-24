# M1 — Primo blocco: progetto e dichiarazione di autorizzazione

[English](../en/M1-PROJECT-AUTHORIZATION.md) · [Roadmap](../ROADMAP.md) · [Architettura](ARCHITECTURE.md)

`internal/project` introduce il modello di progetto **in memoria e senza rete**. Questo blocco prepara lo snapshot del perimetro per una run, ma non realizza ancora scansione, persistenza SQLite o una schermata di gestione progetti.

## Contratto implementato

`project.New(Draft)` richiede ID e nome progetto, nome dichiarato del proprietario del target, riferimento descrittivo all'autorizzazione, conferma esplicita dell'operatore, scadenza futura e da una a 32 origini HTTP(S) esatte. Il riferimento è un'etichetta, non il documento né una credenziale; nessuno di questi campi viene inviato in rete. La conferma registra un'affermazione dell'operatore: **WebFence non verifica autonomamente proprietà o validità legale dell'autorizzazione**.

Un input parziale o ambiguo viene rifiutato. Le origini vengono validate da `internal/scope`, rese canoniche e copiate; duplicati equivalenti sono rifiutati. Il progetto e la lista delle origini non espongono strutture mutabili. `BeginRun()` crea uno snapshot immutabile solo se il permesso non è scaduto. `RunScope.CheckOrigin(url)` ricontrolla la scadenza a ogni chiamata e applica schema, host e porta esatti. Il valore zero nega l'uso. Gli errori hanno codici stabili senza URL, proprietario o riferimento.

Questo controllo è **solo il livello di origine**. Non limita ancora metodi, percorsi, IP/CIDR, DNS, budget o frequenza, e non apre socket. Un suo esito positivo non basta per effettuare una richiesta. Il broker M0 resta confinato al loopback; non è stato collegato a questo modello o alla GUI. Nessun sito esterno è stato contattato per il blocco.

## Verifiche e prossimi blocchi

I test sintetici coprono richiesta ammessa, schema/porta/sottodominio esclusi, scadenza al confine esatto e dopo l'avvio, valore zero, configurazioni invalide, duplicati canonici e impossibilità di allargare uno snapshot modificando le slice del chiamante. La suite race di `internal/project` è passata localmente; la CI multipiattaforma include il nuovo pacchetto nella suite race.

Restano per M1: persistenza e migrazione del progetto, rinnovo/versione dell'autorizzazione, policy di metodi/percorsi/esclusioni e di destinazione IP, integrazione obbligatoria con il trasporto per ogni hop, budget/rate limit condivisi, UI IT/EN, discovery e prove redatte. Nessuno di questi punti è dichiarato implementato qui. I [collaudi umani ereditati da M0](M0-VALIDATION.md) restano prerequisiti di validazione M1.
