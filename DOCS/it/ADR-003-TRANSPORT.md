# ADR-003 — Trasporto M0 confinato al laboratorio

[English](../en/ADR-003-TRANSPORT.md) · [Indice](../README.md) · [Scope URL](M0-SCOPE.md)

Data: 2026-09-23. Stato: **adottato per il laboratorio M0**, non approvazione del trasporto di produzione. Implementazione: `internal/transport`. Nessuna dipendenza aggiunta; nessun collegamento alla GUI.

## Problema e decisione

Il confronto di origini non basta a controllare una connessione: DNS variabile, risposte DNS miste, redirect e richieste concorrenti possono aggirare un controllo applicato solo all’URL iniziale. M0 deve provare questi confini prima di attivare uno scanner.

`NewLab(ctx, grants, limits, resolver)` crea un broker utilizzabile soltanto con IP loopback dichiarati. Ogni grant associa un’origine HTTP(S) a IP esatti; la porta proviene dall’origine. La configurazione viene copiata, le origini duplicate sono rifiutate e gli indirizzi di un’origine non autorizzano le altre. Niente wildcard/CIDR, IPv4 mappato o zone IPv6. Sono rifiutati anche indirizzi privati non loopback, pubblici, link-local e metadata. Questo vincolo è nel codice, non solo nella documentazione.

La scelta permette test HTTP/TLS reali senza aprire il prodotto a target esterni. I risultati costituiscono una fondazione per M1; il broker non va presentato come scanner o come protezione SSRF completa.

## Connessione e DNS

Il resolver è una dipendenza fidata esplicita: deve rispettare il contesto e restituire indirizzi di proprietà del chiamante. Nei test è sintetico; non si usa DNS pubblico. Un IP letterale evita la risoluzione. Per un hostname ogni risposta deve appartenere al grant dell’origine; risposte vuote, errori e insiemi misti sono respinti prima del socket.

Per ogni passaggio HTTP si risolve di nuovo, si sceglie deterministicamente un solo IP ammesso e si connette direttamente a `IP:porta`, verificando anche il peer restituito. Non c’è una seconda risoluzione nel dialer. Host HTTP e nome TLS restano quelli dell’URL originale; certificato e hostname vengono verificati, con TLS minimo 1.2. Le CA di prova sono iniettate solo dai test del pacchetto; l’API non offre un’opzione per saltare TLS.

Si usa solo HTTP/1, con nuova connessione per ogni passaggio, senza riuso, retry automatici o tentativi verso IP alternativi. Questo semplifica la relazione fra tentativi, verifica DNS e budget. Il costo di queste connessioni non è un’ottimizzazione definitiva: eventuale pooling richiederà nuove prove sugli stessi confini.

Proxy d’ambiente disabilitati. `Fetch(ctx, url)` emette esclusivamente GET; non accetta header o credenziali, non gestisce cookie e non invia Referer. I redirect 301/302/303/307/308 sono gestiti esplicitamente; ogni nuova destinazione torna ai controlli di origine e IP. GET può comunque avere effetti: l’interfaccia è per fixture di proprietà, non autorizza scansioni su siti arbitrari.

## Budget e arresto

I limiti obbligatori includono tentativi totali condivisi, concorrenza delle intere catene Fetch, redirect, body, durata Fetch e durata della run. Non ci sono valori illimitati impliciti. Le soglie massime del prototipo sono 16 Fetch concorrenti, 10 redirect e 8 MiB per body; gli header di risposta sono limitati a 32 KiB.

Una prenotazione atomica precede DNS e connessione. Un errore DNS/rete consuma il tentativo; un redirect ammesso ne richiede un altro. URL fuori origine e chiamate già annullate non consumano nuovi tentativi. La durata Fetch comprende attesa di un posto, DNS, connessione, TLS, redirect e lettura; quella della run parte dal costruttore e non si rinnova a ogni chiamata.

`Close()` è idempotente, annulla run e lavoro in corso e impedisce nuove chiamate. Anche il contesto del chiamante può interrompere l’attesa, DNS, dial o lettura. Il contesto Fetch viene passato al dialer reale: il solo contesto interno del transport Go può essere sganciato dalla richiesta. Il test verifica che il dial bloccato termini.

Il broker richiede `Accept-Encoding: identity` e rifiuta risposte compresse: non c’è decompressione implicita. Legge al massimo limite body + 1 byte; se supera il limite non restituisce contenuto parziale come successo. I body intermedi dei redirect vengono chiusi senza essere acquisiti. Il limite riguarda i dati letti dall’applicazione, non un contatore esatto dei byte sulla rete. Nessuna quota totale del disco, persistenza, rate limit o coda durevole introdotta.

Gli errori sono codici stabili, errori scope o `context.Canceled`/`DeadlineExceeded`. Errori DNS/TLS/socket originali vengono redatti. Header e body nei risultati rimangono dati non fidati, da trattare con le future policy di redazione e conservazione.

## Prove completate

Test locali con race detector e `go vet` superati. Il laboratorio verifica:

- DNS controllato una sola volta per passaggio, IP effettivo fissato e Host preservato; cambio DNS fra richieste e dopo redirect respinto.
- Risposte DNS vuote, fallite o miste; IP privati/metadata/mappati/esclusi mai contattati; grant separati e configurazione copiata.
- Richieste concorrenti entro il budget condiviso, ogni redirect conteggiato, cicli limitati e server fuori perimetro mai contattato.
- Proxy d’ambiente ignorato, assenza di cookie/credenziali/Referer; limite body/header e rifiuto della compressione.
- HTTPS locale con CA fidata; certificati non fidati o con nome errato rifiutati, senza disattivare la verifica.
- Annullamento durante DNS, dial, lettura e attesa; run già annullata/scaduta senza richieste; timeout Fetch e chiusura permanente.
- HTTP su loopback IPv6 dove disponibile e rifiuto di un peer diverso da quello approvato. La prova IPv6 segnala uno skip se l’OS non offre loopback IPv6.

Un blocco dei dial indipendente dal codice in prova limita i test agli endpoint posseduti. Gli indirizzi pubblici e metadata compaiono soltanto come dati sintetici, mai come destinazioni reali.

## Conseguenze e limiti residui

Il task copre il gate M0 del laboratorio e dei **primi** test scope/rete. M0 resta aperta per GUI/accessibilità, packaging e altre decisioni di fondazione. Prima di traffico reale servono autorizzazioni/scadenza, esclusioni dei servizi interni, IP/CIDR e reti pubbliche/private, metodi e percorsi, rate limit, redazione, recupero e integrazione del ciclo di scansione. Il laboratorio non dimostra comportamento di DNS Internet, proxy aziendali, browser, WebSocket, reti private reali o resilienza a ogni guasto OS.

Riferimenti verificati: [Go HTTP transport](https://pkg.go.dev/net/http#Transport), [resolver](https://pkg.go.dev/net#Resolver.LookupNetIP), [contesti](https://pkg.go.dev/context), [verifica TLS](https://pkg.go.dev/crypto/tls#Config).
