# ADR-004 — SQLite e firma JWS

[English](../en/ADR-004-STORAGE-SIGNATURE.md) · [Indice](../README.md)

Data: 2026-09-23. Stato: selezione accettata per le fondazioni M0; integrazione applicativa M1/M2 non realizzata. Il portachiavi rimane una decisione separata da verificare.

## Decisione e alternative

| Componente | Scelta | Motivazione e costo |
| --- | --- | --- |
| SQLite | `github.com/mattn/go-sqlite3` **v1.14.52**, SQLite incorporato **3.53.4** | API `database/sql`, motore C upstream incorporato nel modulo; la toolchain C è già richiesta da Qt. Aggiunge compilazione C e richiede verifiche native su ogni target. |
| JWS | `github.com/lestrrat-go/jwx/v4` **v4.5.0** | Supporta l'identificatore `Ed25519` di RFC 9864. Compatibile con Go 1.27.1 senza `GOEXPERIMENT`; uso limitato a JWS con chiave esplicita. |

`modernc.org/sqlite` v1.59.0 è l'alternativa senza CGO: utile per una futura CLI pura, ma non elimina CGO dal desktop Qt e aggiunge il runtime del motore tradotto. Non è stato eseguito un benchmark comparativo; non si rivendicano vantaggi di velocità o memoria. Non usiamo il plugin SQL Qt: lo storage del core deve restare indipendente dal toolkit.

`go-jose/v4` v4.1.5 è stato esaminato, ma espone `EdDSA`, non l'identificatore richiesto dal profilo. jwx v3.3.0 supporta il profilo; scegliamo v4 sul Go già fissato per evitare una successiva migrazione. v4.5.0 include la correzione upstream per l'escaping dei nomi dei membri JSON. Le dipendenze dirette/transitive sono fissate in `go.mod`/`go.sum`; `option/v3` ha versione **alpha1**, rischio di evoluzione API da riesaminare agli aggiornamenti. Nessun aggiornamento automatico a runtime.

## Esperimento SQLite

`internal/foundation/sqlite_test.go` contiene solo prove, senza store o schema del prodotto. Ogni test usa file temporanei sintetici e URI costruiti con `net/url`. Configurazione: un collegamento per pool, WAL, `synchronous=FULL`, foreign key attive, busy timeout 50 ms e `trusted_schema=OFF` applicato con hook a ogni nuova connessione.

Prove: commit/rollback, riapertura e integrità, testo Unicode e SQL trattato come parametro, foreign key, reinizializzazione dopo ricambio connessioni, lettura senza vedere scritture non committate, secondo writer respinto con `SQLITE_BUSY`, ripresa dopo rilascio lock, cancellazione di query lunga e attesa del pool. La scadenza del context non sostituisce il timeout nativo del busy handler.

WAL richiede filesystem locale compatibile; niente cartelle di rete o sincronizzazione del DB attivo. Le prove non dimostrano recupero da perdita di alimentazione, backup, migrazioni, cifratura, quote o cancellazione sicura. Questi flussi e permessi/ACL dello storage reale sono lavoro M1. Non copiare un DB aperto ignorando WAL/SHM.

## Contratto JWS implementato

`internal/signature.Sign` e `Verify` operano su byte esatti. Profilo di fattibilità: serializzazione compact, payload incorporato da 1 byte a 1 MiB, header protetto fino a 1 KiB con **solo** `alg=Ed25519`, `kid` e `typ=webfence-manifest+jws`. `kid` ha 1–64 caratteri ASCII alfanumerici, `_` o `-`; tipo interno non registrato. Header duplicati, sconosciuti, UTF-8 invalido, algoritmi diversi, `crit`, `b64`, `jku`, `jwk`, serializzazione JSON e base64url non canonico vengono respinti.

Il chiamante fornisce chiave pubblica fidata e ID atteso separatamente dal documento. Nessun download di chiavi, persistenza, log del payload o fallback di algoritmo. Errori stabili senza dati sensibili. Chiavi private sintetiche generate nei test, mai versionate. Prove di verifica incrociata con `crypto/ed25519`, chiave estranea, alterazione dei tre segmenti, firme valide su profili vietati, limiti, concorrenza e fuzzing.

Questo non è ancora il verificatore dei report: mancano JCS, schema manifest, hash e percorsi dei file, trust store, revoca, rotazione e integrazione portachiavi. Una verifica positiva autentica i byte rispetto alla chiave fornita; non prova identità, attendibilità delle osservazioni o sicurezza di un sito. La GUI non usa ancora SQLite o JWS.

## Verifiche e licenze

Test con race detector, test Go complessivi, vet e verifica moduli superati localmente su macOS ARM64. Il test stampa SQLite 3.53.4. Il fuzzing JWS locale di 20 secondi ha completato 2.507.150 esecuzioni senza errore. `govulncheck` v1.8.0 con `-test ./...` non ha trovato vulnerabilità note nel codice Go analizzato al momento della verifica; non esamina tutte le librerie native Qt/SQLite né costituisce una certificazione. CI del commit `f1d2a84` superata sui quattro target, macOS ARM64, Windows Server x86-64 e Ubuntu x86-64/ARM64 ([run 35863431196](https://github.com/Matte2599/WebFence/actions/runs/35863431196)): suite race, build, vet, self-test Qt, fuzz JWS su Linux x86-64 e bundle macOS. Nessuna prova Windows 10/11 o Debian interattiva implicita.

Le licenze dei moduli aggiunti sono MIT (mattn, jwx, dsig, option, fastjson), verificate nei sorgenti fissati. SQLite mantiene i propri termini public domain. Conservare i testi/notices nei futuri pacchetti che li includono; questa ricognizione non chiude l'inventario Qt né la revisione legale della distribuzione.

Fonti primarie: [driver e DSN](https://github.com/mattn/go-sqlite3/tree/v1.14.52), [WAL](https://www.sqlite.org/wal.html), [trusted_schema](https://www.sqlite.org/pragma.html#pragma_trusted_schema), [jwx v4.5.0](https://github.com/lestrrat-go/jwx/releases/tag/v4.5.0), [requisiti jwx](https://github.com/lestrrat-go/jwx/tree/v4.5.0), [go-jose v4.1.5](https://github.com/go-jose/go-jose/blob/v4.1.5/shared.go), [RFC 9864](https://www.rfc-editor.org/rfc/rfc9864.html). Le versioni sono state risolte con Go e confrontate con i sorgenti; un indice di ricerca può mostrare release precedenti.
