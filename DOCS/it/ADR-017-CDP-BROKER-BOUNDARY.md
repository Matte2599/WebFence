# ADR-017 — CDP mediato dal broker nel laboratorio Linux

[English](../en/ADR-017-CDP-BROKER-BOUNDARY.md) · [Laboratorio](M3-BROWSER-CDP-LAB.md) · [ADR-016](ADR-016-LINUX-CDP-TRIAL.md) · [Stato M3](M3-VALIDATION.md)

Data: 2026-09-27. Stato: **prova sperimentale, solo fixture sintetiche**.

## Contesto e decisione

La prima prova CDP serviva risposte codificate direttamente nell'helper: conservava l'origine HTTP nel renderer e dimostrava l'eredità del filtro Linux, ma non esercitava le policy WebFence. Il percorso sperimentale viene ora collegato a un'origine `httptest` locale tramite progetto/run dichiarati, gate, broker con IP loopback fissato e proxy Unix autenticato. Il genitore apre otto connessioni al proxy prima di avviare l'helper; il figlio le riceve come descrittori e installa seccomp prima di Chromium. Il client HTTP del figlio può usare soltanto questi descrittori e non segue redirect per proprio conto. CDP intercetta ogni richiesta e trasmette status, corpo limitato e tipo essenziale della risposta del proxy al renderer.

La fixture verifica quattro contatti con il target locale: documento, script, `fetch` e endpoint che risponde con redirect esterno. Il gate nega la risorsa fuori scope senza contattare il target; il broker rifiuta il redirect dopo la risposta iniziale. Si controllano conteggi del target, budget gate/broker e osservazioni redatte. Le richieste sconosciute consumano il budget CDP. Il container CI senza rete aggiunge quote aggregate di memoria e PID alla prova.

Non è un adattatore browser di prodotto. Il laboratorio resta Linux/container, con Chromium `--no-sandbox`, fixture unica, HTTP soltanto e credenziale di proxy effimera in memoria. La sandbox dei file, il confinamento fuori dal container, TLS/HTTPS, cookie e sessioni, revoca durante la pagina, profili per utenti e integrazione desktop richiedono nuove prove. Il desktop Go/Qt Widgets/MIQT e il requisito di isolamento OS dell'ADR-015 restano invariati; la prima voce M3 è aperta.
