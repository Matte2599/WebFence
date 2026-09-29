# ADR-022 — Avvio headless sotto sandbox OS esterna

[English](../en/ADR-022-HEADLESS-OUTER-SANDBOX.md) · [Confine richiesto](ADR-015-BROWSER-PLATFORM-BOUNDARY.md) · [Prove](M3-BROWSER-PLATFORM-FEASIBILITY.md)

Data: 2026-09-29. Stato: **esperimento di avvio; adozione nel prodotto aperta**.

## Problema

Chrome desktop richiede servizi non disponibili nei confini provati: LaunchServices su macOS e Crashpad su Windows. Il runtime ufficiale `chrome-headless-shell` separa il motore headless dall'applicazione Chrome. Si prova la versione **154.0.8037.57**, revisione Chromium `73c14f6228d7cd537c855007e8f88678969cc0eb`; archivio verificato con SHA-256 prima dell'esecuzione, nessun browser installato modificato.

Su macOS il confezionamento come `.app` fa cercare framework/risorse e helper da applicazione Chromium. Un bundle `.bundle` mantiene i percorsi del runtime autonomo. L'avvio richiede inoltre il rendezvous Mach privato del browser. Superati questi ostacoli, i figli terminano con `sandbox initialization failed: Operation not permitted`: App Sandbox è già attiva quando Chromium tenta un secondo confinamento Seatbelt.

La prova Windows con debugger individua l'arresto del runtime fissato all'RVA `0x254a169`: il percorso termina nel controllo successivo a `BrokerServices::CreateAlternateDesktop(kAlternateWinstation)`. Il [codice della revisione](https://github.com/chromium/chromium/blob/73c14f6228d7cd537c855007e8f88678969cc0eb/sandbox/policy/sandbox.cc) prepara una window station per la sandbox interna; la prova è già in AppContainer. Questo approfondisce il fallimento headless, non attribuisce automaticamente la stessa causa ai precedenti crash Edge/Chrome.

## Decisione sperimentale

Il laboratorio prova **App Sandbox/AppContainer come confine OS di tutto l'albero browser**, esterno al desktop e al broker WebFence. Solo questo percorso sperimentale usa `--no-sandbox` per evitare l'inizializzazione della sandbox interna incompatibile. Non è la stessa protezione a più livelli del laboratorio Linux: si perde la separazione Chromium fra renderer e browser. Il flag non deve essere aggiunto al desktop, ai launcher generici o alle prove Linux. Un runtime avviato fuori dal confine non è un esito accettabile.

Nel probe macOS:

- il browser è firmato con App Sandbox, senza capacità di rete; il helper è firmato per ereditare la sandbox;
- un processo fidato in attesa riserva il PID prima della firma; gli entitlement Mach consentono soltanto il nome esatto `org.chromium.Chromium.MachPortRendezvousServer.<PID>`, senza wildcard, LaunchServices o permessi di rete;
- CDP usa pipe ereditate; ambiente ridotto, profilo temporaneo, tempo/frame/eventi/diagnostica limitati e terminazione del gruppo processi;
- il controllo sul runtime originale esegue DOM e raggiunge HTTP/file sintetici; quello confinato deve eseguire DOM, negare HTTP/file esterno e leggere il file privato;
- `sandbox_check` richiede una sandbox OS per browser, renderer, GPU e servizio di rete restituiti dal runtime durante la fixture.

La firma ad hoc per ogni PID è una tecnica di laboratorio, **non una soluzione di distribuzione firmata/notarizzata**. Il codice non contiene un launcher browser di prodotto. Restano da risolvere packaging stabile, identità dei processi in presenza di contenuto ostile, broker HTTPS, servizi locali, quote aggregate, revoca e integrazione desktop. Nessun criterio M3 viene chiuso con questa prova. Il requisito dell'autore di supportare tutti e tre gli OS resta invariato.

## Dipendenze e limiti

Il probe macOS accetta solo l'archivio ARM64 fissato, SHA-256 `9ba4d8a9732bd7e431ce9009d1bbe1ccf7f8c5de2a9781054f500a9fe898124d`. Quello Windows usa l'archivio win64, SHA-256 `bcc91b4d0f83a5457fc6ca7941fd65f2349775523d961be88526c6a66560dc75`; il runner Windows ARM64 esegue quindi un runtime x64 emulato. La licenza e il packaging del runtime devono essere esaminati prima della distribuzione; binari e archivi non sono commessi.

Fonti: [Chrome headless shell](https://developer.chrome.com/docs/automation-and-testing/headless-chrome-shell), [sandbox Chromium macOS](https://chromium.googlesource.com/chromium/src/+/main/sandbox/mac/), [entitlement temporanei Apple](https://developer.apple.com/library/archive/documentation/Miscellaneous/Reference/EntitlementKeyReference/Chapters/AppSandboxTemporaryExceptionEntitlements.html), [IPC AppContainer](https://learn.microsoft.com/en-us/windows/apps/develop/communication/interprocess-communication).
