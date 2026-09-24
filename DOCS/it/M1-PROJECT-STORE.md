# M1 — Store locale dei progetti

[English](../en/M1-PROJECT-STORE.md) · [Progetto e autorizzazione](M1-PROJECT-AUTHORIZATION.md) · [ADR SQLite](ADR-004-STORAGE-SIGNATURE.md) · [Roadmap](../ROADMAP.md)

`internal/storage` realizza la **prima persistenza di metadati M1** con SQLite. È un componente Go del core: il desktop non lo invoca ancora e non salva i progetti dell'operatore. Non contiene credenziali, risultati di scansione, prove, report o copie del sito.

## Contratto implementato

`storage.Open(ctx, path)` richiede un percorso assoluto il cui genitore esiste già e crea un file nuovo con permessi `0600` sui sistemi Unix. Rifiuta file non regolari e, su Unix, file esistenti accessibili a gruppo/altri. Chi chiama sceglie e protegge la directory su un filesystem locale affidabile; l'API non definisce ancora la cartella dati dell'app né risolve ogni gara su symlink o ACL Windows. Gli errori pubblici sono codici senza percorso o dati di progetto.

Lo schema **v1** contiene `projects` e `project_origins`, con chiave esterna e cancellazione a cascata. Il riferimento all'autorizzazione è testo descrittivo, non il documento originale; rimane comunque metadato potenzialmente sensibile **in chiaro** nel file SQLite. Le origini sono canoniche e ordinate. `CreateProject` inserisce progetto e origini in una transazione; un ID già presente è rifiutato. `LoadProject` e `ListProjects` ricostruiscono progetti immutabili con una lettura coerente. Una dichiarazione scaduta resta leggibile per gestione e futura modifica, ma `BeginRun` continua a rifiutare nuove run. `DeleteProject` elimina soltanto le righe di questo schema; non è cancellazione fisica sicura di pagine, WAL, backup o futuri artefatti.

All'apertura vengono controllati versione, tabelle richieste, `quick_check` e chiavi esterne; un database sconosciuto o con versione più nuova è rifiutato. Le connessioni usano WAL, `synchronous=FULL`, foreign key attive, `trusted_schema=OFF` e timeout di lock finito. La v1 nasce atomicamente da un file vuoto; non esiste ancora una migrazione v1→v2. I test sintetici coprono riapertura dopo commit, elenco, cancellazione a cascata, scadenza dopo ripristino, duplicati, rollback a metà inserimento, schemi incompatibili, record alterati e cancellazione del contesto. La CI esegue la suite con race detector sulle piattaforme native configurate.

## Limiti e prossimi passi

Questa è **persistenza di configurazione**, non il flusso completo di M1. Mancano gestione GUI IT/EN, rinnovo e versioni dell'autorizzazione, quota di disco, backup/ripristino verificati, dati di run e risultati, arresto dei job prima della cancellazione, file di evidenza e rimozione di tutte le copie gestite dall'app. Il filesystem locale e i permessi/ACL su ogni sistema reale richiedono prove dedicate. Non copiare un database aperto ignorando WAL/SHM; la strategia di backup sarà un blocco separato. Nessuna scansione esterna è stata eseguita per queste prove.
