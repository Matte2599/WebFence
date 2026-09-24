# Dati persistenti, cancellazione e convalida fix

[English](../en/DATA-AND-RETEST.md) · [Indice](../README.md)

Stato: modello logico M1–M4; il solo [schema v1 dei metadati di progetto](M1-PROJECT-STORE.md) è implementato. Le altre entità, la conservazione completa e il retest sono pianificati. Driver SQLite e prove temporanee M0 in [ADR-004](ADR-004-STORAGE-SIGNATURE.md).

## Entità

| Entità | Dati e invarianti |
| --- | --- |
| Project | ID, proprietario locale, ambiente, lingua, conservazione; non contiene segreti |
| ScopeRevision / Authorization | Regole ammesse/escluse, prova o riferimento di autorizzazione, scadenza |
| Asset / Endpoint | Origine, metodo, route e posizioni parametri; dati dinamici separati |
| CredentialRef | Riferimento al portachiavi, identità logica e scadenza; niente token nel record |
| ScanRun / CheckExecution | Versioni, scope immutabile, stato, budget, checkpoint, esito per controllo |
| Finding / Observation | Identità stabile del problema e osservazioni immutabili delle singole run |
| Evidence | Oggetto redatto, hash, fonte, limiti di cattura, data ed eventuale scadenza |
| IntelligenceSnapshot | Fonte e revisione per ricostruire la correlazione storica |
| RetestRun / Comparison | Baseline, contesto, esiti comparabili, copertura e possibili regressioni |
| ReportArtifact / AuditEvent | Riferimenti firmati e azioni redatte, senza ricostruire segreti cancellati |

ID e codici macchina sono indipendenti dalla lingua. Timestamp UTC; transazioni e foreign key proteggono i riferimenti. Non sovrascrivere prove storiche con risultati nuovi. Deduplicazione proposta: ID regola/famiglia, origine, metodo, route, posizione del parametro e ruolo logico. Conservare versione del fingerprint; cambi di algoritmo richiedono migrazione o mapping esplicito. Escludere token e valori volatili, senza confondere percorsi semanticamente diversi.

## Conservazione

«Tenere il sito in memoria» significa conservare inventario, configurazione, finding e cronologia su disco fino a cancellazione esplicita, non tenere tutte le pagine in RAM né clonare integralmente il sito. Prima scansione: mostrare cosa viene salvato e la quota.

Metadati e prove redatte necessarie al retest: conservazione fino a cancellazione, con politica TTL opzionale. Body grezzi e screenshot: disabilitati per default; quando richiesti hanno scadenza breve configurabile, proposta iniziale 7 giorni. Credenziali: portachiavi, riuso esplicito, durata minima necessaria. Le quote arrestano nuove acquisizioni e indicano incompletezza; non eliminano silenziosamente una baseline.

Cancellare un progetto arresta i job, elimina credenziali del progetto, record, file, cache AI, eventuali indici e temporanei. Riferimenti condivisi alle credenziali vanno disassociati senza cancellare quelli usati da altri progetti. Un registro minimo di cancellazione non contiene URL o prove. Backup e copie esportate hanno una politica esplicita: quelli gestiti dall'app vengono eliminati o scadono secondo la politica dichiarata; quelli consegnati altrove non sono revocabili dall'app. Il ripristino non deve reintrodurre dati cancellati senza avviso e decisione dell'operatore.

Non promettere cancellazione fisica garantita su SSD, snapshot o backup esterni. Non applicare automaticamente etichette di «conformità GDPR»; responsabilità e termini sul trattamento dipendono dall'uso reale.

## Esiti di retest

| Stato | Significato |
| --- | --- |
| `still_present` | Evidenza sufficiente a riprodurre il problema |
| `fixed_verified` | Controllo equivalente eseguito, prerequisiti validi, prova di conferma prevista superata |
| `not_observed` | Problema non visto, ma verifica insufficiente a confermare il fix |
| `inconclusive` | WAF, timeout, login scaduto, ambiente diverso o dati mancanti |
| `not_tested` | Regola disabilitata, scope escluso o budget esaurito prima del controllo |

`accepted_risk` e `false_positive` sono decisioni di triage motivate, non esiti tecnici di retest. Una soppressione non deve nascondere automaticamente osservazioni nuove.

Prima del confronto verificare ambiente, autorizzazione, identità e ruolo, route, rule revision, dati di test e condizioni. Recuperare token dinamici; non riprodurre ciecamente vecchie richieste autenticate. Un 404, 403 o cambio di versione da solo non prova una correzione. Se la nuova regola non è equivalente, mantenere `inconclusive` e richiedere nuova baseline motivata.

Un possibile nuovo problema è `newly_observed`; diventa `regression` soltanto se esiste evidenza comparabile che prima il controllo pertinente passava. Nuova copertura o nuove regole possono far emergere un problema preesistente. Mostrare separatamente delta finding e delta copertura. Le regressioni fuori dallo scope riesaminato non sono escluse da un retest riuscito.
