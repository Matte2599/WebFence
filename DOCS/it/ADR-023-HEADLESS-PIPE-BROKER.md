# ADR-023 — Broker HTTP(S) via pipe per i laboratori headless

[English](../en/ADR-023-HEADLESS-PIPE-BROKER.md) · [Avvio confinato](ADR-022-HEADLESS-OUTER-SANDBOX.md) · [Stato M3](M3-VALIDATION.md)

Data: 2026-10-02. Stato: **implementato nel core e nei laboratori; integrazione desktop aperta**.

## Problema e decisione

Il browser headless dell'ADR-022 parte dentro App Sandbox/AppContainer senza rete autonoma, ma la prova di avvio non carica risorse HTTP(S) tramite WebFence. Si aggiunge `internal/browser.RequestBridge`: il controller fidato riceve URL, metodo e tipo della richiesta intercettata, applica il gate e usa il broker con IP fissato. Il costruttore richiede lo stesso snapshot `BeginRun` e le stesse regole di metodo/percorso nel gate e nel broker: ID/revisione uguali di due run indipendenti non bastano. Non apre listener e non accetta header o body del browser. `GET`/`HEAD`, scope, percorso, budget, tempo, revoca, DNS e verifica TLS restano nei componenti esistenti.

Il browser riceve via [CDP Fetch](https://chromedevtools.github.io/devtools-protocol/tot/Fetch/) una risposta limitata, con i soli header di rendering già ammessi dal proxy. Cookie target, credenziali browser e autorizzazione proxy non vengono trasferiti. Il limite della risposta è 16 KiB per il body e 8 KiB per gli header; le osservazioni sono effimere, al massimo 256, senza query/header/body. La chiusura del ponte o del gate cancella le richieste in corso. Un rifiuto di policy produce 403, un fallimento del broker 502: nessuno diventa un esito di sicurezza positivo.

Il ponte non rappresenta ancora i redirect ammessi: se il broker segue un redirect e cambia URL finale, il ponte restituisce 502, evitando di mostrare il body finale sotto l'URL originale. I redirect fuori scope vengono fermati dal broker prima del contatto esterno. La navigazione completa attraverso redirect ammessi resta da implementare.

## Laboratori e verifiche

`experiments/internal/headlessfixture` crea soltanto server loopback su porte assegnate dal sistema. Per HTTP e HTTPS usa `site.test`, policy `/app`, 16 tentativi gate/broker e un certificato TLS effimero affidato solo al broker della fixture. Non accetta target esterni. Il documento carica uno script, una risorsa fuori origine e un `fetch` con header Authorization sintetico; seguono redirect vietato, richiesta lenta e richiesta dopo revoca.

La pagina deve riportare l'origine esatta, `isSecureContext=true` soltanto per HTTPS, cookie vuoti, il body API e gli stati 502/502/403. Il controller richiede cinque richieste al target, zero contatti esterni, tre osservazioni ammesse e cancellazione effettiva della richiesta lenta. Il target rifiuta nel test host o credenziali inattesi.

Su macOS il controller Python usa un broker Go separato su stdin/stdout; il processo broker è fuori dalla sandbox del browser. Su Windows lo stesso codice fixture gira nel genitore Go. CDP resta su pipe ereditate; frame 64 KiB, annidamento otto comandi, eventi e numero di richieste limitati. Le risposte fuori ordine sono associate all'ID e alla sessione in attesa. Nessuna richiesta intercettata viene continuata sullo stack di rete Chromium.

Le prove di avvio, rete diretta, file esterno/privato e sandbox OS dei discendenti restano richieste, anche dopo la navigazione mediata. La CI esegue il nuovo percorso su macOS 15/26 ARM64 e Windows Server 2022/Windows 11 ARM64; il motore Windows sul runner ARM64 resta x64 emulato. Il laboratorio Linux esistente conserva il proprio confine e la sandbox interna Chromium.

Prova locale macOS 26.6.2 ARM64: HTTP(S) mediato, origine/contesto, revoca, canary OS e test del protocollo passati. Suite Go, race detector browser/fixture, vet, self-test Qt IT/EN e 92 test Python (4 saltati) passati. La CI del branch e del merge va verificata per il commit corrente; questo non è il test conclusivo M3.

## Riproduzione e limiti

Su macOS, dalla radice del repository, con l'archivio fissato dell'ADR-022 già disponibile:

```sh
CGO_ENABLED=0 go build -o /tmp/webfence-m3-headless-broker ./experiments/m3-headless-broker
python3 experiments/m3-macos-sandbox/headless_probe.py --runtime-archive /percorso/chrome-headless-shell-mac-arm64.zip --broker-executable /tmp/webfence-m3-headless-broker
```

Su Windows la compilazione e `--headless-appcontainer` dell'ADR-022 eseguono anche le due fixture mediate. Il comando Go ausiliario accetta solo `http` o `https` e messaggi sintetici su pipe; non è un proxy di produzione.

Restano aperti packaging/firma stabili, quote aggregate macOS e Linux fuori dal container, servizi locali e contenuto ostile, sessioni browser autenticate, redirect ammessi, gestione di altri target CDP e integrazione Qt. Il modello OS esterno e il flag sperimentale dell'ADR-022 restano invariati. Nessun criterio di uscita M3 è chiuso da questo blocco.
