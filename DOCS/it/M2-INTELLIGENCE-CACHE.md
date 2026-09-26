# M2 — Cache CVE/NVD

[English](../en/M2-INTELLIGENCE-CACHE.md) · [Roadmap](../ROADMAP.md) · [Specifiche intelligence](VULNERABILITY-INTELLIGENCE.md)

Stato: **primo blocco core M2**. `internal/intelligence` offre due adattatori distinti: pagine NVD CVE API 2.0 per una finestra esplicita e record CVE Program per ID espliciti. Nessun feed viene scaricato all'avvio, nessuna query al sito analizzato viene inviata alle fonti. Il [desktop M2](M2-VALIDATION.md) espone l'aggiornamento manuale; nessun database CVE è incluso nei pacchetti.

## Contratto implementato

- La cache SQLite è separata dal database dei progetti. Conserva record JSON delle fonti, stato, date dichiarate, SHA-256 del contenuto e acquisizione UTC. Il solo hash non autentica la fonte; la connessione normale usa HTTPS verso gli endpoint ufficiali. Le fonti `cve` e `nvd` non vengono fuse.
- Una sincronizzazione NVD richiede `start`/`end` UTC scelti dal chiamante, al massimo 120 giorni. Scarica pagine da 500 risultati, con limiti di 128 pagine, 16 MiB per risposta, 1 MiB per record e 512 MiB per file principale. Il watermark e tutti i record della finestra vengono confermati insieme; errore, annullamento o pagina malformata lasciano l'ultima cache valida. Una finestra successiva può sovrapporsi; `NextWindow` suggerisce un'ora di sovrapposizione. Una cache parziale non equivale all'intero catalogo CVE.
- L'adattatore CVE recupera fino a 32 ID espliciti per operazione, con commit unico. Un aggiornamento fallito non sostituisce i record precedenti. La cache conserva anche stati ritirati/rifiutati se la fonte li restituisce.
- `Snapshot.Status` espone `fresh`, `stale`, `offline`, `unavailable` usando una soglia scelta dal chiamante; `offline` indica l'uso intenzionale senza rete, non freschezza. `WindowStart` e `Watermark` dichiarano la copertura temporale della sincronizzazione, non la completezza del catalogo.
- Le chiamate HTTP non seguono redirect; rispettano `Retry-After` entro 30 secondi e fanno al massimo tre tentativi per risposte transitorie. Limiti temporali, dimensioni, pagine e numero di record sono espliciti. Endpoint personalizzati sono ammessi solo su loopback per test sintetici.

## Verifica e limiti

`go test -race ./internal/intelligence` copre paginazione, provenienza, riapertura offline, isolamento delle fonti, rollback dopo feed non valido/indisponibile e rifiuto di endpoint esterni personalizzati. I test non contattano NVD o CVE reali. Non si è dimostrata la copertura di CVE storiche, l'autenticità di import offline o la compatibilità con ogni record fonte. Il desktop richiede sempre un'azione esplicita per aprire rete verso le fonti.

Fonti primarie: [NVD API 2.0](https://nvd.nist.gov/developers/vulnerabilities), [CVE Services](https://github.com/CVEProject/cve-services), [catalogo ufficiale CVE](https://github.com/CVEProject/cvelistV5). Parametri e limiti delle fonti vanno rivalutati agli aggiornamenti.
