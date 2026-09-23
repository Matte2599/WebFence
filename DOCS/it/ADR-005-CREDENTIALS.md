# ADR-005 — Portachiavi nativo

[English](../en/ADR-005-CREDENTIALS.md) · [Indice](../README.md)

Data: 2026-09-23. Stato: implementazione M0 selezionata, verifica runtime multipiattaforma in attesa. È una base del core, non una GUI di gestione credenziali o il ciclo di vita delle chiavi dei report.

## Decisione

`internal/credentials` offre `New`, `Get`, `Set` e `Delete`. Espone soltanto il namespace fisso WebFence. Gli ID contengono 1–64 lettere ASCII minuscole, cifre, `_` o `-`; i segreti contengono 1–2048 byte arbitrari. Le minuscole evitano alias dei nomi Windows, che non distinguono maiuscole e minuscole. Nessun elenco, cancellazione massiva, fallback su file in chiaro o segreti d’ambiente, log o sblocco automatico. Input non validi e context cancellati sono respinti prima dell’accesso nativo. Le diagnostiche native diventano errori stabili senza ID o segreti; assente, bloccato/interazione necessaria, ambiguo e indisponibile restano distinti quando il backend lo consente.

| Sistema | Adattatore | Limiti |
| --- | --- | --- |
| macOS | Piccolo adattatore CGO alle API classiche del portachiavi Security.framework; portachiavi predefinito, servizio/account esatti | Consente binari di sviluppo senza entitlement Data Protection. Le API Apple sono deprecate: scelta di fattibilità M0 da rivalutare per la distribuzione firmata. |
| Windows | `github.com/danieljoos/wincred` v1.2.3, credenziale generica di Credential Manager | Utente corrente, `PersistLocalMachine`; niente roaming enterprise. Non costituisce isolamento per applicazione rispetto ad altro codice eseguito dallo stesso utente. |
| Linux | `github.com/godbus/dbus/v5` v5.2.2 e adattatore ristretto Secret Service | Richiede servizio di sessione esistente e raccolta predefinita sbloccata. Nessuna attivazione del servizio, creazione raccolta, sblocco o esecuzione di prompt. |

Gli adattatori non dipendono dalla GUI. SQLite, firme dei report e credenziali dei provider non sono ancora collegati. `Set` sostituisce una voce esatta; con chiamanti concorrenti prevale l’ultima scrittura, senza compare-and-swap. Una scrittura nativa fallita/cancellata può avere esito incerto: il chiamante deve verificare tramite lettura, senza presumere un rollback.

## Comportamento nativo e confini

Il portachiavi classico macOS usa un interruttore di interazione valido per l’intero processo. Ogni operazione del pacchetto viene serializzata, disabilita la UI e ripristina il valore precedente prima del ritorno. Un portachiavi bloccato fallisce prima della ricerca; l’autenticazione necessaria fallisce senza aprire finestre. Non introdurre chiamanti Security.framework indipendenti nello stesso processo finché esiste questo adattatore: il lock non coordina librerie esterne. La cancellazione funziona prima dell’ingresso e durante l’attesa del lock; le chiamate sincrone Security.framework non sono interrompibili dopo l’ingresso. Nessuna goroutine separata prosegue una modifica dopo il ritorno. Non chiamare dal thread eventi Qt. La futura app firmata deve rivalutare Data Protection/LAContext e migrazione delle credenziali. Il solo attributo moderno «no UI» delle query non garantisce la soppressione per le voci del portachiavi legacy, come documentato dall’SDK.

Anche le chiamate Windows sono sincrone e non interrompibili dopo l’ingresso. Accesso negato dal sistema diventa bloccato/interazione necessaria; sessioni di accesso indisponibili restano indisponibili. Il blocco schermo non è un confine di diniego provato o promesso. Le credenziali generiche non sono chiavi hardware o identità di firma non esportabili.

Linux ammette un solo endpoint D-Bus Unix-domain (`path` o `abstract`, GUID facoltativo). TCP, autolaunch, endpoint multipli, chiavi ambigue e NUL sono respinti. Una connessione privata usa autenticazione EXTERNAL, fissa il proprietario univoco corrente di Secret Service e applica un context di tre secondi a connessione, autenticazione e chiamate. Chiude la propria connessione, mai un bus condiviso dell’app. Raccolta predefinita assente/bloccata causa errore; più voci corrispondenti sono ambigue. Il trasferimento usa l’algoritmo `plain` della specifica sul bus locale: non aggiunge cifratura di trasporto. La protezione su disco spetta al servizio OS. Bus e servizio locali sono fidati; non difende da account, daemon del bus o amministratore compromessi.

I buffer posseduti dal wrapper vengono azzerati dopo le scritture e le letture respinte; le letture riuscite appartengono al chiamante. Go, librerie native e sistema possono conservare copie. Nessuna garanzia di cancellazione sicura. Lo store non offre ancora recupero, rotazione, backup, editor ACL o traduzioni degli errori nella GUI: appartengono all’integrazione.

## Alternative esaminate

`zalando/go-keyring` v0.2.8 usa il comando `security` su macOS e attese di prompt Linux senza context del chiamante. Il segreto macOS passa su stdin, non argv; la scelta riguarda identità del comando e controllo del ciclo di vita. `99designs/keyring` v1.2.2 include più backend del necessario e nella lettura macOS controlla risultati vuoti prima dell’errore, rischiando di rappresentare una lettura indisponibile/bloccata come assente. `keybase/go-keychain` v0.0.1 espone wrapper SecItem ma non il controllo di interazione legacy necessario per operazione. Nessun benchmark o graduatoria generale di sicurezza di questi progetti. I moduli Windows e D-Bus scelti sono MIT; `x/sys` transitivo è BSD-3-Clause, fissato nel grafo moduli. I servizi Apple/Windows mantengono i propri termini di piattaforma.

## Verifiche

MacOS ARM64 locale: superati unit test con race detector e prove native reali di scrittura/lettura; valori binari, dimensione massima, aggiornamento, riapertura, isolamento ID esatto, cancellazione e assenza. Un portachiavi temporaneo separato viene bloccato per verificare il diniego di Get/Set/Delete, poi cancellato; i portachiavi personali non vengono bloccati. Un percorso di portachiavi esplicito assente causa errore. Unit test su validazione senza I/O, cancellazione durante l’attesa, store zero, redazione errori e buffer. Test Go completi, vet e verifica moduli superati. Binari di test Linux/Windows compilati localmente in cross-compilazione: non prova di esecuzione nativa. Integrazione CI sui quattro target aggiunta, in attesa dell’esecuzione effettiva.

Test Linux su indirizzi bus respinti, socket assente e cancellazione durante autenticazione senza risposta. `scripts/test-keychain-linux.sh` avvia sessione D-Bus privata e GNOME Keyring con directory XDG temporanee e password sintetica. Verifica servizio assente prima dell’avvio, poi lettura/scrittura/cancellazione e diniego con raccolta bloccata. Non esegue la prova di blocco sul bus personale del desktop. I test Windows usano un prefisso casuale e cancellano soltanto la propria voce sintetica; non bloccano la sessione dell’utente. Comportamento nativo Windows con blocco/sessione indisponibile ancora da provare in un collaudo interattivo controllato, senza inventare equivalenza con Linux/macOS.

## Fonti

[Controllo interazione classico Apple](https://developer.apple.com/documentation/security/seckeychainsetuserinteractionallowed(_:)), header Security dell’SDK macOS locale; [tipi e persistenza credenziali Windows](https://learn.microsoft.com/en-us/windows/win32/api/wincred/ns-wincred-credentialw); [wincred v1.2.3](https://github.com/danieljoos/wincred/tree/v1.2.3); [godbus v5.2.2](https://github.com/godbus/dbus/tree/v5.2.2); [specifica Secret Service](https://specifications.freedesktop.org/secret-service/latest/), [trasferimento plain](https://specifications.freedesktop.org/secret-service/latest/ch07s02.html).

Prima CI `aa722e3`: test nativi macOS superati; entrambi i runner Linux hanno evidenziato un rifiuto errato della lettura binaria. GNOME Keyring restituisce sempre `text/plain`, anche dopo scrittura `application/octet-stream` ([sorgente upstream](https://github.com/GNOME/gnome-keyring/blob/main/daemon/dbus/gkd-secret-secret.c)). L’adattatore conserva ora byte opachi senza interpretare MIME; controlla ancora sessione, parametri e dimensione. Il launcher attende il proprietario del servizio senza attivare un secondo daemon. Correzione da verificare nella CI successiva.

Aggiunta prova Windows in sottoprocesso dedicato: un thread fissato usa il token anonimo di Windows per verificare il diniego nativo di lettura/scrittura/cancellazione; il token viene ripristinato e la credenziale sintetica originale viene verificata nel processo padre. Nessun blocco della sessione personale. Compilazione e vet Windows in cross-compilazione superati; esecuzione nativa ancora da verificare. [API Microsoft](https://learn.microsoft.com/en-us/windows/win32/api/securitybaseapi/nf-securitybaseapi-impersonateanonymoustoken).
