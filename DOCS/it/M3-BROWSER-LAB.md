# M3 — Terzo blocco: laboratorio Qt WebEngine

[English](../en/M3-BROWSER-LAB.md) · [ADR-010](ADR-010-QT-BROWSER-LAB.md) · [ADR-011](ADR-011-BROWSER-HELPER.md) · [Proxy](M3-BROWSER-PROXY.md) · [Roadmap](../ROADMAP.md)

`experiments/m3-browser` è una prova ripetibile **solo con fixture create dal comando**. Si esegue con Qt 6 WebEngine/MIQT 0.14.0 e i flag C++17:

```sh
CGO_CXXFLAGS='-O0 -g0 -std=c++17' QT_QPA_PLATFORM=offscreen go run -tags=m3browserlab ./experiments/m3-browser
```

Il programma non accetta URL o argomenti target. Un'origine `.test` è collegata esclusivamente al server `httptest` tramite il broker con IP `127.0.0.1` fissato. Qt WebEngine usa profilo senza persistenza e proxy HTTP autenticato su loopback. L'interceptor blocca origine/percorso/metodo non ammessi, worker e WebSocket; il proxy e il broker verificano di nuovo le richieste effettive. La fixture include documento, script, `fetch`, click programmato su un link DOM verso una seconda pagina, uno script esterno e un nuovo `fetch`, immagine fuori scope e redirect fuori scope. Il test richiede sette richieste effettive identiche nei conteggi di broker e target locale, senza header di autenticazione; il gate può contare anche richieste respinte prima del broker. Conferma il blocco dell'immagine e del redirect e registra sei osservazioni ammesse, compresa la route caricata dopo il click. L'esito JavaScript è un segnale della fixture, non un'evidenza affidabile proveniente da una pagina arbitraria.

Dal quarto blocco, il laboratorio avvia il runtime Qt in un [helper supervisionato](ADR-011-BROWSER-HELPER.md), che applica anche [limiti di risorse Unix](M3-BROWSER-RESOURCE-LIMITS.md). La credenziale proxy viaggia su `stdin` solo dopo l'applicazione del confine di processo. Il genitore limita tempo e output, termina i discendenti e valuta indicatori sintetici redatti. Il test del supervisore e la validazione della configurazione del figlio si eseguono con:

```sh
go test -race ./internal/browser -run '^TestRunHelper' -count=1
CGO_CXXFLAGS='-O0 -g0 -std=c++17' go test -tags=m3browserlab ./experiments/m3-browser
```

**Prova locale del 27 settembre 2026, Apple Silicon/macOS 26.6.2, Qt WebEngine 6.11.2:** il comando con helper e click DOM ha restituito `PASS` in quattro ripetizioni precedenti e dopo i nuovi limiti Unix; test Go con tag passati. Il job CI macOS 26 è separato e va verificato sul commit del branch e del merge. Nessun collaudo Qt equivalente è ancora documentato su Windows. Il laboratorio non è una funzionalità GUI e non ha egress indipendente, limiti di memoria o quota aggregata di processi su macOS/Linux, HTTPS o sessioni; non soddisfa il criterio browser M3.

La CI include ora un job Linux Ubuntu 24.04 con Qt WebEngine e display virtuale: test di configurazione e fixture sintetica sono passati sul branch `0da4e66` ([run 36281325352](https://github.com/Matte2599/WebFence/actions/runs/36281325352)). Verificare separatamente il commit integrato. La prova locale macOS è stata ripetuta dopo l'aggiunta del job. Non è una prova del contenimento indipendente dell'egress.

Il primo tentativo Linux ha compilato il helper ma ha restituito `browser_helper_failed` nell'esecuzione. L'allowlist dell'ambiente del supervisor ometteva `XAUTHORITY`, necessario al display virtuale con autenticazione X11; dopo averlo inoltrato, la stessa fixture è passata. La causa è coerente con questo esito, senza attribuire al job una prova di isolamento dell'egress.

Il runtime Windows resta aperto: il desktop usa MIQT/MinGW, mentre [Qt WebEngine 6.11 non compila con MinGW](https://doc.qt.io/qt-6/qtwebengine-platform-notes.html). Un helper separato con toolchain Qt compatibile è una proposta da motivare e verificare in un ADR, non una capacità implementata.

La CI del branch `4f7e2c0` ha mostrato un quinto tentativo contato dal gate, ma respinto prima del broker: quattro richieste hanno raggiunto broker e target. Il laboratorio ora confronta questi ultimi conteggi esatti e richiede che il gate abbia contato almeno tutte le richieste inoltrate; richieste ulteriori consumano budget ma non sono considerate traffico target. Questo cambiamento richiede nuova CI verde.

La [CI del merge `cf504cf`](https://github.com/Matte2599/WebFence/actions/runs/36282644664) ha mostrato una fluttuazione del laboratorio macOS: il timer fisso di 5 secondi ha terminato la pagina prima dei `fetch`. La correzione `24a7523` ha superato la CI finale. La [CI del merge DOM `fd6be19`](https://github.com/Matte2599/WebFence/actions/runs/36289947405) ha poi raggiunto la seconda pagina senza eseguire lo script inline e il secondo `fetch` entro 12 secondi: cinque richieste target e quattro osservazioni. La fixture ora carica uno script esterno dalla seconda pagina e attende i due segnali dinamici per massimo 20 secondi, sotto il limite parent di 30 secondi. Dieci ripetizioni locali macOS sono passate; la nuova CI va verificata per commit.
