# M3 — Visite API statiche selezionate

[English](../en/M3-API-BATCH.md) · [Import OpenAPI](M3-OPENAPI-IMPORT.md) · [Visite controllate](M1-CONTROLLED-CRAWL.md) · [Roadmap](../ROADMAP.md)

`scanner.RunAPICrawl` riceve un documento OpenAPI JSON e **da 1 a 32 percorsi GET scelti esplicitamente dall'operatore**. Importa il documento offline sotto una breve run gestita e accetta soltanto percorsi statici classificati come candidati dall'importatore. L'origine, la policy GET, i grant IP e i budget vengono dal piano dichiarato, mai da `servers` o `$ref` del documento. Percorsi duplicati, non presenti, template, autenticati, esclusi o metodi diversi sono rifiutati prima della rete.

Una nuova run gestita passa i seed a `RunCrawl`, che ricontrolla autorizzazione, scope, policy e trasporto con IP fissati prima del primo invio. Le visite sono sequenziali, senza seguire link, redirect o inviare form; un redirect interrompe la run come parziale. Restano gli stessi limiti di richieste e corpo del crawler M1. Il ledger conserva codici e conteggi redatti, senza URL o risposta. Una revisione di autorizzazione tra import e avvio richiede la nuova verifica; l'import non concede accesso autonomo.

**Verifica locale del 27 settembre 2026:** fixture `httptest` con due route JSON selezionate, `servers` esterno inerte, due richieste GET e ledger redatto; selezioni negate o ambigue senza richieste; race detector del pacchetto. La CI del branch e del merge va verificata per commit. Questo è un servizio core: la GUI seleziona ancora un solo seed e non offre una scansione API multi-route. Non esegue controlli di sicurezza specifici per API JSON, route con parametri, autenticazione OpenAPI o crawling JavaScript. La terza voce M3 rimane aperta.
