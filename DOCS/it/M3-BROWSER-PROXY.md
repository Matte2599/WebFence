# M3 — Secondo blocco: proxy HTTP confinato

[English](../en/M3-BROWSER-PROXY.md) · [Gate](M3-BROWSER-GATE.md) · [ADR-009](ADR-009-BROWSER-HTTP-BOUNDARY.md) · [Roadmap](../ROADMAP.md)

`internal/browser.NewProxy` apre un listener effimero **solo su `127.0.0.1`**. Richiede una credenziale proxy casuale di 256 bit, generata per istanza e conservata soltanto in memoria. Dopo l'autenticazione, ammette richieste HTTP in forma assoluta, GET/HEAD e senza body. Ogni richiesta passa dal [gate M3](M3-BROWSER-GATE.md) e poi dal broker M1 con scope, policy, DNS/IP fissati, budget, rate e redirect ricontrollati. Una risposta o un errore non include URL nei messaggi del proxy.

Il proxy respinge `CONNECT` HTTPS, upgrade WebSocket, body, worker dichiarati e metodi non ammessi. Non inoltra al target gli header del client, inclusi credenziali e cookie; il broker crea una propria richiesta senza header di autenticazione. La risposta riporta solo gli header necessari alla resa e alle restrizioni base, escludendo `Set-Cookie`; `Cache-Control: no-store` evita riusi non conteggiati. Il proxy non implementa una sessione autenticata e non modifica lo scope in base al contenuto della pagina.

**Questo blocco non avvia né configura un browser.** Un browser potrebbe ignorare il proxy o usare canali alternativi: servono runtime isolato, egress indipendente, intercettazione di tutti i tipi di risorsa, limiti di processi/memoria e prove su ogni piattaforma. HTTPS, autenticazione e WebSocket non sono supportati da questo proxy. Non usarlo come prova di contenimento M3 o per target esterni.

Le prove `httptest` usano solo loopback e verificano richiesta valida, HEAD, header/CSP e mancato inoltro di cookie e credenziali, autenticazione proxy, origine terza, percorso escluso, worker, WebSocket, POST e redirect fuori scope. Non sono prove con un browser reale né misure di copertura SPA/API.
