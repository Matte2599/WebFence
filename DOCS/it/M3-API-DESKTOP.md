# M3 — Visite API statiche nel desktop

[English](../en/M3-API-DESKTOP.md) · [Servizio API](M3-API-BATCH.md) · [Import offline](M3-OPENAPI-DESKTOP.md) · [Roadmap](../ROADMAP.md)

La finestra Qt Widgets/MIQT espone **Visita route OpenAPI selezionate** per un progetto autorizzato. Legge un documento JSON locale di massimo 1 MiB, costruisce offline le route GET statiche ammesse da scope e prefissi correnti e presenta una lista a selezione multipla. L'operatore sceglie da 1 a 32 percorsi e conferma la finestra prima delle richieste. Non viene preselezionata alcuna route. La conferma usa modalità loopback o IP pubblici fissati, budget e cadenza già indicati nel modulo; i link HTML non vengono seguiti anche se l'opzione generale è attiva.

Il servizio core reimporta il documento e rivalida le scelte in una nuova run gestita prima del traffico. Il pulsante resta disabilitato durante una scansione. Annullamento, file invalido o selezione vuota non inviano richieste; il file e i percorsi non sono salvati nel progetto o nel report. Il risultato mostra solo codici e conteggi redatti, secondo il ledger M1. Un budget insufficiente o un redirect rende la run parziale; la destinazione del redirect non viene visitata. Questa funzione non invia POST, non usa schemi di autenticazione OpenAPI e non verifica vulnerabilità specifiche delle risposte JSON.

**Verifica locale del 27 settembre 2026:** self-test Qt offscreen IT/EN con `servers` esterno inerte e due route JSON del server `httptest` selezionate esplicitamente; solo due GET, run persistita e redazione di URL/body. Regressioni Go e race detector del desktop/core passati localmente; CI del branch e del merge da verificare per commit. Browser dinamico, sessioni nel desktop e regole API restano aperti.
