# M3 — Stato tecnico e verifica complessiva

[English](../en/M3-VALIDATION.md) · [Roadmap](../ROADMAP.md) · [Sviluppo](DEVELOPMENT.md)

**M3 non è chiusa.** I blocchi sotto sono prove circoscritte su fixture sintetiche; nessuna scansione di target esterni è stata eseguita. Le quattro voci della roadmap restano aperte finché non sono soddisfatti i rispettivi criteri di uscita. La CI verde dimostra che codice, self-test e packaging previsti dai job passano sul commit indicato, non che il browser sia isolato o che la copertura reale sia sufficiente.

| Voce M3 | Implementato e verificabile | Mancante per la chiusura |
| --- | --- | --- |
| Browser isolato | [Gate e proxy HTTP](M3-BROWSER-PROXY.md), [helper Qt e laboratorio](M3-BROWSER-LAB.md) con documento, subresource, redirect, `fetch` e un click DOM su fixture loopback; [limiti Unix dell'helper](M3-BROWSER-RESOURCE-LIMITS.md) per file/descrittori/core dump; [prova IPC su socket Unix](M3-BROWSER-UNIX-SCHEME.md) con tre risorse sintetiche macOS. | Egress indipendente dal browser, limiti di memoria e numero aggregato di processi su Linux/macOS, contenimento e prova del runtime su Windows, HTTPS mediato, integrazione sicura nel desktop e verifiche negative su tutte le piattaforme. |
| Account di prova | [Sessioni verificate](M3-SESSIONS.md), [run a due account](M3-CROSS-ROLE-RUN.md) e [dialogo Qt](M3-AUTH-DESKTOP.md) limitati a loopback, con conferme rinnovate a ogni run e segreti effimeri. | Login con CSRF/MFA/OIDC o bearer, rotazione cookie, grant pubblici autenticati, browser JavaScript autenticato e inventario persistente di identità. |
| OpenAPI, dinamica e contesto | [Import OpenAPI offline](M3-OPENAPI-IMPORT.md), [visite statiche GET selezionate nel desktop](M3-API-DESKTOP.md), [ripetizione core di `fetch` osservate](M3-OBSERVED-CRAWL.md) e primo [controllo tra ruoli](M3-CROSS-ROLE.md). | Crawler di interazioni DOM nel prodotto, parametri/query dinamici sotto policy, controlli API specifici, più risorse/ruoli e misure su un corpus SPA/API rappresentativo. |
| Famiglie di regole | `HTTP-XCTO-001` osserva un header HTML; `AUTH-CROSSROLE-001` richiede prova positiva esatta e conferma del divieto. Sei casi sintetici del controllo tra ruoli producono un finding e cinque inconcludenti. | Altre famiglie con prerequisiti, fixture positive/negative e misure pubblicate; sensibilità, specificità e falsi positivi su corpus reale non sono noti. |

Le credenziali, URL, query, header e corpi non sono inclusi nei risultati redatti di questi blocchi; le password della GUI non vengono salvate, ma copie create da Go, Qt o dal sistema operativo possono restare in memoria. L'assenza di finding o una risposta diversa non dimostra che l'accesso sia corretto. GET può avere effetti: le fixture locali non autorizzano prove esterne.

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
