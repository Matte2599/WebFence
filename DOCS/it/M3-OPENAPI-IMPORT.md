# M3 — Import OpenAPI offline: inventario core

[English](../en/M3-OPENAPI-IMPORT.md) · [Roadmap](../ROADMAP.md) · [Scansione](SCANNING.md)

`internal/apiimport.Import` legge in memoria un documento JSON OpenAPI 3.0.x o 3.1.x e restituisce un inventario ordinato di operazioni. È un **blocco core**, non ancora un importatore nella GUI né un piano di scansione. Durante l'import non apre connessioni, non visita route e non risolve `$ref`.

L'operatore deve fornire separatamente un'origine esatta già ammessa dalla `project.RunScope` e una `scope.RequestPolicy` valida. I valori `servers` del documento non cambiano l'origine. Soltanto GET/HEAD statiche, senza requisiti di sicurezza OpenAPI e ammesse da scope/percorso, ricevono un URL candidato. Le altre operazioni restano nell'inventario con un codice di esclusione: template da parametrizzare, autenticazione, metodo, policy, percorso ambiguo o riferimento irrisolto. Un candidato non viene eseguito automaticamente: la futura selezione esplicita dovrà ancora passare dal broker con grant IP, budget, cadenza e controlli per richiesta.

Il parser rifiuta chiavi JSON duplicate, input non UTF-8, sintassi o versione non supportate e documenti oltre 1 MiB, profondità 32, 20.000 token, 512 percorsi o 1.024 operazioni. Non interpreta schemi, parametri, esempi, `callbacks`, link o requisiti di sicurezza complessi; i percorsi template non sono espansi. L'inventario è effimero e può contenere nomi di percorsi sensibili: non va copiato nel ledger o nei report senza un contratto di redazione.

**Verifica locale del 27 settembre 2026:** fixture sintetiche per `servers` esterni ignorati, scope e policy, sicurezza, `$ref`, template, metodi non lettura, chiavi duplicate e limiti di dimensione/profondità. Nessun target esterno è stato contattato. Questo blocco non chiude la voce M3 che comprende anche crawling dinamico, controlli contestuali e accessi tra ruoli.
