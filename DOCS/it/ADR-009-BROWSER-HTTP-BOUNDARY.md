# ADR-009 — Confine HTTP locale per il browser M3

[English](../en/ADR-009-BROWSER-HTTP-BOUNDARY.md) · [Proxy M3](M3-BROWSER-PROXY.md) · [ADR-008](ADR-008-PINNED-PUBLIC-TRANSPORT.md)

Data: 2026-09-26. Stato: **adottato per il prototipo core M3**; runtime browser e uso su target esterni non approvati.

## Contesto e decisione

Il browser pianificato produce navigazioni, risorse e richieste JavaScript indipendenti dal crawler M1. Consentirgli l'accesso diretto alla rete aggirerebbe la policy Go e gli IP fissati. Il primo percorso di rete M3 è quindi un proxy HTTP locale autenticato: dopo il [gate](M3-BROWSER-GATE.md), il broker M1 esegue ogni GET/HEAD con verifiche di scope, percorso, DNS, peer, budget, cadenza e redirect. Il proxy ascolta su una porta effimera di `127.0.0.1`, richiede una credenziale casuale per istanza e non inoltra header o cookie del client.

Per questo blocco rifiutiamo `CONNECT` e tutti i metodi con body. Un tunnel HTTPS non consentirebbe al proxy di vedere ogni percorso e metodo interno senza un ulteriore confine; introdurre ora una CA MITM aggiungerebbe gestione di certificati e rischi non giustificati da un browser ancora assente. Il broker segue i redirect ammessi e rifiuta quelli fuori scope prima del nuovo collegamento.

## Conseguenze e prossima decisione

Il proxy permette di collaudare il percorso HTTP su fixture locali, ma non obbliga un futuro browser a usarlo. La sua credenziale difende il listener da chiamanti locali non autorizzati, non è una sandbox del processo. Rimangono assenti runtime, egress indipendente, limiti di processi/memoria, HTTPS, sessioni, cookie e verifica delle vie alternative. La scelta tra Qt WebEngine e un driver separato, insieme al modello HTTPS e ai vincoli OS, richiede una decisione successiva con prove su macOS, Windows e Linux. Nessuna voce M3 è chiusa da questo ADR.
