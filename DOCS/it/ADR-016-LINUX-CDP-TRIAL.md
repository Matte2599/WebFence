# ADR-016 — Prova Chromium/CDP su Linux

[English](../en/ADR-016-LINUX-CDP-TRIAL.md) · [Prova](M3-BROWSER-CDP-LAB.md) · [ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md) · [Stato M3](M3-VALIDATION.md)

Data: 2026-09-27. Stato: **prova sperimentale; runtime di prodotto non scelto**.

## Contesto e decisione

Lo schema Qt sperimentale non mantiene la semantica delle origini HTTP e la versione Qt della baseline Ubuntu non esegue `fetch` su quello schema. Serve una prova ripetibile che conservi una vera origine HTTP mentre il processo browser non può aprire connessioni proprie. Il desktop resta Go e Qt Widgets/MIQT.

Si aggiunge un laboratorio Linux separato, compilabile solo con `m3cdplab`. Il supervisore Go avvia un helper con limiti e scadenza; l'helper installa il filtro seccomp prima di creare Chromium headless. Il collegamento CDP usa pipe ereditate, non una porta TCP. `Fetch.requestPaused` serve risposte sintetiche per un documento HTTP, JavaScript e `fetch`, e blocca una risorsa e un redirect fuori origine. Il laboratorio conta le richieste, limita frame e durata e controlla `/proc` per `seccomp` e `no_new_privs` ereditati dal processo Chromium. La CI esegue il programma in un container senza rete, con filesystem di sola lettura e quote aggregate di memoria e PID.

Questa scelta **non adotta Chromium nel prodotto**. Il laboratorio usa `--no-sandbox` per partire come utente non privilegiato nel container: il suo filtro Linux resta attivo, ma viene disattivata la sandbox interna del renderer Chromium. Non fornisce protezione di file/processi adeguata a pagine ostili, né collega CDP al gate, al broker, alle run gestite o alla GUI. Le risposte sono codificate nella fixture; HTTPS, cookie, revoca, distribuzione e licenze del browser non sono risolti. Prima di qualunque uso su target, occorrono un confine OS completo e un adattatore mediato dal broker, con prove negative ripetibili e revisione delle dipendenze.

Fonti primarie: [pipe DevTools Chromium](https://chromium.googlesource.com/chromium/src/+/HEAD/content/public/browser/devtools_agent_host.h), [dominio Fetch CDP](https://chromedevtools.github.io/devtools-protocol/tot/Fetch/), [modalità headless Chrome](https://developer.chrome.com/docs/automation-and-testing/headless).
