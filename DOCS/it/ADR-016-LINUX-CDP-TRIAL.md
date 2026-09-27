# ADR-016 — Prova Chromium/CDP su Linux

[English](../en/ADR-016-LINUX-CDP-TRIAL.md) · [Prova](M3-BROWSER-CDP-LAB.md) · [Evoluzione ADR-017](ADR-017-CDP-BROKER-BOUNDARY.md) · [ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md)

Data: 2026-09-27. Stato: **prova sperimentale; runtime di prodotto non scelto**.

## Contesto e decisione

Lo schema Qt sperimentale non mantiene la semantica delle origini HTTP e la versione Qt della baseline Ubuntu non esegue `fetch` su quello schema. Serve una prova ripetibile che conservi una vera origine HTTP mentre il processo browser non può aprire connessioni proprie. Il desktop resta Go e Qt Widgets/MIQT.

La prima versione aggiungeva un laboratorio Linux separato, compilabile solo con `m3cdplab`. Il supervisore Go avviava un helper con limiti e scadenza; l'helper installava il filtro seccomp prima di creare Chromium headless. Il collegamento CDP usava pipe ereditate, non una porta TCP. `Fetch.requestPaused` serviva risposte sintetiche codificate per documento HTTP, JavaScript e `fetch`, e bloccava risorsa e redirect fuori origine. Il laboratorio contava le richieste, limitava frame e durata e controllava `/proc` per `seccomp` e `no_new_privs` ereditati da Chromium. La CI eseguiva il programma in un container senza rete, con filesystem di sola lettura e quote aggregate di memoria e PID. L'[ADR-017](ADR-017-CDP-BROKER-BOUNDARY.md) registra il successivo collegamento al gate e al broker.

Questa scelta **non adotta Chromium nel prodotto**. Il laboratorio usa `--no-sandbox` per partire come utente non privilegiato nel container: il suo filtro Linux resta attivo, ma viene disattivata la sandbox interna del renderer Chromium. Non fornisce protezione di file/processi adeguata a pagine ostili. HTTPS, cookie, revoca, distribuzione e licenze del browser non sono risolti. Prima di qualunque uso su target occorrono un confine OS completo, prove negative ripetibili e revisione delle dipendenze.

Fonti primarie: [pipe DevTools Chromium](https://chromium.googlesource.com/chromium/src/+/HEAD/content/public/browser/devtools_agent_host.h), [dominio Fetch CDP](https://chromedevtools.github.io/devtools-protocol/tot/Fetch/), [modalità headless Chrome](https://developer.chrome.com/docs/automation-and-testing/headless).
