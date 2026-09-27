# M3 — Terzo blocco: laboratorio Qt WebEngine

[English](../en/M3-BROWSER-LAB.md) · [ADR-010](ADR-010-QT-BROWSER-LAB.md) · [ADR-011](ADR-011-BROWSER-HELPER.md) · [Proxy](M3-BROWSER-PROXY.md) · [Roadmap](../ROADMAP.md)

`experiments/m3-browser` è una prova ripetibile **solo con fixture create dal comando**. Si esegue con Qt 6 WebEngine/MIQT 0.14.0 e i flag C++17:

```sh
CGO_CXXFLAGS='-O0 -g0 -std=c++17' QT_QPA_PLATFORM=offscreen go run -tags=m3browserlab ./experiments/m3-browser
```

Il programma non accetta URL o argomenti target. Un'origine `.test` è collegata esclusivamente al server `httptest` tramite il broker con IP `127.0.0.1` fissato. Qt WebEngine usa profilo senza persistenza e proxy HTTP autenticato su loopback. L'interceptor blocca origine/percorso/metodo non ammessi, worker e WebSocket; il proxy e il broker verificano di nuovo le richieste effettive. La fixture include documento, script, `fetch`, immagine fuori scope e redirect fuori scope. Il test riesce solo se documento/script/API e origine del redirect sono arrivati al target locale senza header di autenticazione, i quattro tentativi conteggiati coincidono, l'immagine è stata negata e la destinazione del redirect ha prodotto un errore nel browser. L'esito JavaScript è un segnale della fixture, non un'evidenza affidabile proveniente da una pagina arbitraria.

Dal quarto blocco, il laboratorio avvia il runtime Qt in un [helper supervisionato](ADR-011-BROWSER-HELPER.md). La credenziale proxy viaggia su `stdin` solo dopo l'applicazione del confine di processo. Il genitore limita tempo e output, termina i discendenti e valuta indicatori sintetici redatti. Il nuovo test del supervisore e la validazione della configurazione del figlio si eseguono con:

```sh
go test -race ./internal/browser -run '^TestRunHelper' -count=1
CGO_CXXFLAGS='-O0 -g0 -std=c++17' go test -tags=m3browserlab ./experiments/m3-browser
```

**Prova locale del 27 settembre 2026, Apple Silicon/macOS 26.6.2, Qt WebEngine 6.11.2:** il comando con helper ha restituito `PASS`. Il job CI macOS 26 è separato e va verificato sul commit del branch e del merge. Nessun collaudo Qt equivalente è ancora documentato su Windows. Il laboratorio non è una funzionalità GUI e non ha egress indipendente, limiti rigidi di processi/memoria su macOS/Linux, HTTPS o sessioni; non soddisfa il criterio browser M3.

La CI include ora un job Linux Ubuntu 24.04 con Qt WebEngine e display virtuale: test di configurazione e fixture sintetica sono passati sul branch `0da4e66` ([run 36281325352](https://github.com/Matte2599/WebFence/actions/runs/36281325352)). Verificare separatamente il commit integrato. La prova locale macOS è stata ripetuta dopo l'aggiunta del job. Non è una prova del contenimento indipendente dell'egress.

Il primo tentativo Linux ha compilato il helper ma ha restituito `browser_helper_failed` nell'esecuzione. L'allowlist dell'ambiente del supervisor ometteva `XAUTHORITY`, necessario al display virtuale con autenticazione X11; dopo averlo inoltrato, la stessa fixture è passata. La causa è coerente con questo esito, senza attribuire al job una prova di isolamento dell'egress.

Il runtime Windows resta aperto: il desktop usa MIQT/MinGW, mentre [Qt WebEngine 6.11 non compila con MinGW](https://doc.qt.io/qt-6/qtwebengine-platform-notes.html). Un helper separato con toolchain Qt compatibile è una proposta da motivare e verificare in un ADR, non una capacità implementata.
