# M3 — Osservazioni delle richieste browser

[English](../en/M3-DYNAMIC-OBSERVATIONS.md) · [Laboratorio Qt](M3-BROWSER-LAB.md) · [Proxy](M3-BROWSER-PROXY.md) · [Roadmap](../ROADMAP.md)

`browser.NewObservedProxy` e `NewObservedProxyWithSession` conservano fino a 256 osservazioni **solo in memoria**, per una singola istanza di proxy/run. Registrano soltanto richieste GET/HEAD ammesse dal gate e completate dal broker, con metodo, tipo risorsa, stato e percorsi iniziale/finale. Query, header, cookie, body e credenziale proxy non entrano nell'osservazione. Quando la quota è piena o un percorso supera 1024 byte, cresce il conteggio degli omessi. `Observations` restituisce una copia; il chiamante non può alterare il registro interno.

La fixture Qt esegue JavaScript che avvia `fetch('/app/api')`: il parent vede quella route nel proxy, mentre il redirect verso un'origine esterna e la subresource esclusa non diventano osservazioni. La versione con sessione usa il cookie solo nel parent, già verificato, e registra il percorso senza la query. Le osservazioni sono dati non fidati: non sono seed automaticamente schedulati e non vanno persistite nel ledger senza una regola di redazione. Il proxy resta HTTP e il browser non è ancora integrato nel desktop.

**Verifiche locali del 27 settembre 2026:** test con race detector per limite, copia, query rimossa, richieste negate, sessione e redirect con credenziali; laboratorio Qt macOS su fixture sintetica. La CI multipiattaforma del branch e del merge va verificata per commit. Questo blocco prova discovery di richieste JavaScript, non crawling con navigazioni o interazioni DOM: la terza voce M3 resta aperta.
