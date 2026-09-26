# M3 — Primo blocco: ammissione delle richieste browser

[English](../en/M3-BROWSER-GATE.md) · [Roadmap](../ROADMAP.md) · [Architettura](ARCHITECTURE.md)

`internal/browser.Gate` è il primo confine di policy per M3, **non un browser avviabile**. Richiede lo scope di una run gestita, la policy GET/HEAD con percorsi espliciti e limiti di richieste, concorrenza e durata. Il chiamante deve applicare `Admit` a ogni navigazione, redirect, subresource e richiesta JavaScript prima della rete e rilasciare la lease alla fine. Ogni tentativo, anche negato, consuma il budget; errori e conteggi non contengono URL o credenziali. Revoca, scadenza e cancellazione interrompono anche l'attesa di uno slot.

La policy iniziale ammette solo documenti, subresource e `fetch` GET/HEAD. Blocca worker e WebSocket finché un adattatore non dimostra il contenimento anche di quei canali. Il controllo di origine e percorso riusa `project.RunScope` e `scope.RequestPolicy`: nessun server indicato dal contenuto del sito amplia lo scope. Le richieste con metodo o schema non supportato vengono negate.

Il gate non apre socket, non conosce l'IP effettivo, non impedisce a un processo browser di aggirarlo e non applica limiti a memoria o numero di processi. **Non è collegato alla GUI né a un browser** e non abilita scansioni dinamiche o autenticate. Prima di attivare il browser servono un adattatore, il passaggio obbligato dal broker con IP fissati per ogni richiesta, un limite verificato dei processi e prove negative su redirect, subresource, worker, WebSocket e canali alternativi. Un interceptor applicativo isolato non sostituisce un confine di egress indipendente.

I test sintetici di questo blocco coprono ammissioni valide, origini terze, percorsi esclusi e ambigui, metodi non ammessi, schemi locali, worker/WebSocket, budget, concorrenza, chiusura e revoca. Non contattano target esterni e non misurano ancora la copertura di SPA/API.
