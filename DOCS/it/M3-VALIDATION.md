# M3 — Stato tecnico e verifica complessiva

[English](../en/M3-VALIDATION.md) · [Roadmap](../ROADMAP.md) · [Sviluppo](DEVELOPMENT.md)

**M3 non è chiusa.** I blocchi sotto sono prove circoscritte su fixture sintetiche; nessuna scansione di target esterni è stata eseguita. Le quattro voci della roadmap restano aperte finché non sono soddisfatti i rispettivi criteri di uscita. La CI verde dimostra che codice, self-test e packaging previsti dai job passano sul commit indicato, non che il browser sia isolato o che la copertura reale sia sufficiente.

| Voce M3 | Implementato e verificabile | Mancante per la chiusura |
| --- | --- | --- |
| Browser isolato | [Gate e proxy HTTP](M3-BROWSER-PROXY.md), [helper Qt e laboratorio](M3-BROWSER-LAB.md) con documento, subresource, redirect, `fetch` e un click DOM su fixture loopback; [limiti Unix dell'helper](M3-BROWSER-RESOURCE-LIMITS.md) per file/descrittori/core dump; [prova IPC su socket Unix](M3-BROWSER-UNIX-SCHEME.md) con tre risorse sintetiche macOS; [filtro Linux](M3-BROWSER-LINUX-NETWORK.md) che nega socket INET diretti e nuove connessioni Unix nel secondo helper; [prova CDP Linux](M3-BROWSER-CDP-LAB.md) con origine HTTP, gate/broker e container senza rete. | Sandbox per pagine ostili e limiti aggregati fuori dal container; contenimento e prova del runtime su Windows e macOS; HTTPS mediato, integrazione sicura nel desktop e verifiche negative su tutte le piattaforme. |
| Account di prova | [Sessioni verificate](M3-SESSIONS.md), [run a due account](M3-CROSS-ROLE-RUN.md) e [dialogo Qt](M3-AUTH-DESKTOP.md) su loopback; [trasporto core HTTPS con IP pubblici fissati](ADR-018-AUTHENTICATED-PUBLIC-TRANSPORT.md), con conferme rinnovate a ogni run e segreti effimeri; la pagina di login esatta può fornire un campo hidden CSRF e un cookie di pre-sessione facoltativo; una rotazione host-only con stesso nome/percorso è verificata di nuovo sotto budget. | Altri flussi CSRF, MFA/OIDC o bearer, rotazioni continue durante la verifica o con cambio di nome/percorso, orchestrazione pubblica a due account (il [trasporto core HTTPS](ADR-018-AUTHENTICATED-PUBLIC-TRANSPORT.md) è disponibile), browser JavaScript autenticato e inventario persistente di identità. |
| OpenAPI, dinamica e contesto | [Import OpenAPI offline](M3-OPENAPI-IMPORT.md), [visite statiche GET selezionate nel desktop](M3-API-DESKTOP.md), [ripetizione core di `fetch` osservate](M3-OBSERVED-CRAWL.md) e primo [controllo tra ruoli](M3-CROSS-ROLE.md). | Crawler di interazioni DOM nel prodotto, parametri/query dinamici sotto policy, controlli API specifici, più risorse/ruoli e misure su un corpus SPA/API rappresentativo. |
| Famiglie di regole | `HTTP-XCTO-001` osserva un header HTML; `AUTH-CROSSROLE-001` richiede prova positiva esatta e conferma del divieto. Sei casi sintetici del controllo tra ruoli producono un finding e cinque inconcludenti. | Altre famiglie con prerequisiti, fixture positive/negative e misure pubblicate; sensibilità, specificità e falsi positivi su corpus reale non sono noti. |

La [prova di fattibilità per piattaforma](M3-BROWSER-PLATFORM-FEASIBILITY.md) e l'[ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md) documentano perché il primo criterio non può essere spuntato sulla base dei laboratori correnti. Gli esiti locali macOS richiedono una prova riproducibile e integrata nel packaging prima di scegliere un runtime.

Le credenziali, URL, query, header e corpi non sono inclusi nei risultati redatti di questi blocchi; le password della GUI non vengono salvate, ma copie create da Go, Qt o dal sistema operativo possono restare in memoria. L'assenza di finding o una risposta diversa non dimostra che l'accesso sia corretto. GET può avere effetti: le fixture locali non autorizzano prove esterne.

**Aggiornamento cumulativo dei blocchi browser su `9a4f4a6`, 27 settembre 2026:** in locale sono passati `go mod verify`, test del browser con race detector, `go test ./...`, `go vet ./...`, test e laboratorio Qt con tag `m3browserlab`, self-test desktop IT/EN, soak di 10 secondi, 79 test Python (4 saltati) e verifica dei link della documentazione. Il test Linux ARM64 in container ha provato il filtro e l'eredità sui discendenti. [CI del branch IPC](https://github.com/Matte2599/WebFence/actions/runs/36312004380), [relativo merge](https://github.com/Matte2599/WebFence/actions/runs/36312583122), [CI del branch Linux](https://github.com/Matte2599/WebFence/actions/runs/36312923254) e [relativo merge](https://github.com/Matte2599/WebFence/actions/runs/36313511877) sono verdi. Questa è una regressione intermedia, non la validazione conclusiva di M3 né prova di egress completo.

**Revisione Linux con IPC preconnesso su `1c6ce89`:** in locale sono passati il test seccomp Linux ARM64 e il laboratorio Qt WebEngine 6.4 in container, il laboratorio Qt macOS, `go test ./...`, `go vet ./...`, 79 test Python (4 saltati) e verifica di 180 documenti/1664 link locali. La [CI del branch](https://github.com/Matte2599/WebFence/actions/runs/36317757369) è verde su Linux, macOS, Windows e packaging. Prova un percorso sintetico più ristretto, non la chiusura del browser né il test finale M3.

**Prima prova CDP Linux del 27 settembre 2026:** un container ARM64 senza rete, in sola lettura e con quota di memoria/PID ha superato documento HTTP, script, `fetch`, risorsa e redirect esterni bloccati. Il processo Chromium ha ereditato seccomp e `no_new_privs`; due test negativi rifiutano frame CDP eccessivi e consumano il budget anche per richieste sconosciute. In locale sono passati `go test ./...`, `go vet ./...`, `go vet` sul laboratorio Linux e la verifica di 178 documenti/1697 link locali. La prima versione usava risposte codificate nell'helper e non era collegata al gate/broker.

**Revisione CDP con broker:** la prova ARM64 ha superato 40 ripetizioni con container senza rete, quota 1 GiB/256 PID e filesystem in sola lettura. Sono passati documento, script e `fetch` attraverso gate/proxy/broker, immagine esterna bloccata dal gate, redirect esterno fermato dal broker, quattro contatti target e tre osservazioni redatte. Il limite precedente di 128 PID causava un errore intermittente di avvio Chromium. `go test ./...`, `go vet ./...`, `go vet` del laboratorio Linux e verifica documentale locale sono passati. L'uso di `--no-sandbox`, l'assenza di HTTPS e il confinamento solo nel container impediscono di chiudere la prima voce M3.

**Prova negativa di rete Linux:** un listener TCP sintetico aperto dal genitore non è raggiungibile dall'helper filtrato (`EPERM`), sia in container senza rete sia con rete `bridge`; dieci ripetizioni locali per modalità sono passate. In una build diagnostica senza la chiamata al filtro, il medesimo controllo fallisce. `go test ./...`, `go vet ./...`, test/vet Linux del laboratorio e 181 documenti/1825 link locali sono passati. Il job CI ora ripete entrambe le modalità; non è una prova di sandbox del renderer o delle altre piattaforme.

**Prova intermedia CSRF del 27 settembre 2026:** token hidden e cookie di pre-sessione facoltativo sono confinati all’URL di login esatto e al budget condiviso; il self-test Qt IT/EN ha completato due account con 13 richieste loopback. Sono passati la suite Go, il race detector mirato, `go vet`, 79 test Python (4 saltati), il controllo di 186 documenti/1714 link locali e `git diff --check`. Questa non è la validazione finale di M3.

**Prova intermedia di rotazione del 27 settembre 2026:** due account hanno mantenuto identità separate attraverso rotazione dopo login e risorsa (14 richieste condivise); una run del controllo tra ruoli ha prodotto il finding atteso con 15 richieste. Fixture negative rifiutano dominio/percorso estesi, duplicati, cancellazione, cambio di identità, seconda rotazione e redirect nella conferma. Test concorrenti con race detector, suite Go, go vet, self-test Qt IT/EN e 79 test Python (4 saltati) sono passati; solo loopback sintetico. Questa non è la validazione finale M3.

## Test complessivo

**Verifica locale integrata del commit `b82693a`, 27 settembre 2026:** Apple Silicon, macOS 26.6.2, Go 1.27.1, Qt 6.11.2. Tutti i comandi seguenti sono terminati con codice zero; hanno usato solo fixture sintetiche e loopback.

| Area | Verifica | Esito |
| --- | --- | --- |
| Dipendenze | `go mod verify` | Tutti i moduli verificati. |
| Core e regressioni | `go test -race` sui 17 pacchetti core, poi `go test ./...` e `go vet ./...` | Passati, inclusi scanner, browser, sessioni, controlli, desktop, M1 e M2. |
| Build | `go build` dei tre comandi `webfence`, `webfence-report`, `webfence-verify` | Passate. |
| Desktop nativo | `QT_QPA_PLATFORM=offscreen webfence --self-test` | Passato in IT/EN: M0, M1, M2, OpenAPI M3 e controllo tra ruoli con due account sintetici. |
| Stabilità GUI | `webfence --soak-test=10s` offscreen | Passato, 4 cicli completi in 12 secondi. |
| Browser sperimentale | `go test -tags=m3browserlab ./experiments/m3-browser`, `go vet` con lo stesso tag e tre `go run -tags=m3browserlab ./experiments/m3-browser` offscreen | Passati: documento, click DOM, script esterno, due `fetch`, proxy e blocchi fuori scope. |
| Script e documenti | `python3 -m unittest discover -s scripts/tests`, `python3 scripts/package-project-docs.py --check .`, `git diff --check` | 79 test Python, 4 saltati, nessun errore; sul branch di validazione 166 Markdown e 1.536 collegamenti locali validi; diff pulito. |

La CI aggiunge build, self-test e packaging di sviluppo su macOS, Windows e Ubuntu ARM64/x86-64; il laboratorio WebEngine è provato su macOS e Linux, non su Windows. Per i blocchi finali, la CI del branch e quella del relativo squash merge su `main` sono distinte:

| Blocco | Branch | Merge su `main` |
| --- | --- | --- |
| Run gestita tra ruoli | [`9d898ee`](https://github.com/Matte2599/WebFence/actions/runs/36287236763) verde | [`b297358`](https://github.com/Matte2599/WebFence/actions/runs/36287857087) verde |
| Dialogo Qt a due account | [`64fa46b`](https://github.com/Matte2599/WebFence/actions/runs/36288145773) verde | [`1587732`](https://github.com/Matte2599/WebFence/actions/runs/36288784914) verde |
| Replay di `fetch` osservati | [`8182adc`](https://github.com/Matte2599/WebFence/actions/runs/36287637738) verde | [`96670a5`](https://github.com/Matte2599/WebFence/actions/runs/36289377317) verde |
| Click DOM nel laboratorio | [`dfab892`](https://github.com/Matte2599/WebFence/actions/runs/36287803019) verde | [`fd6be19`](https://github.com/Matte2599/WebFence/actions/runs/36289947405) fallita: seconda pagina raggiunta, secondo `fetch` assente entro 12 secondi. |
| Stabilità dello script della seconda pagina | [`5c3631f`](https://github.com/Matte2599/WebFence/actions/runs/36290666798) verde | [`b82693a`](https://github.com/Matte2599/WebFence/actions/runs/36291202123) verde. |

Questa verifica copre regressioni del software e delle fixture. Non sostituisce prove assistive/hardware, un target di staging autorizzato, una misura di accuratezza reale o il contenimento di rete a livello di sistema operativo. Nessuna voce M3 viene marcata completata sulla sola base dei test di laboratorio.
