# M3 — Terzo blocco: laboratorio Qt WebEngine

[English](../en/M3-BROWSER-LAB.md) · [ADR-010](ADR-010-QT-BROWSER-LAB.md) · [Proxy](M3-BROWSER-PROXY.md) · [Roadmap](../ROADMAP.md)

`experiments/m3-browser` è una prova ripetibile **solo con fixture create dal comando**. Si esegue con Qt 6 WebEngine/MIQT 0.14.0 e i flag C++17:

```sh
CGO_CXXFLAGS='-O0 -g0 -std=c++17' QT_QPA_PLATFORM=offscreen go run -tags=m3browserlab ./experiments/m3-browser
```

Il programma non accetta URL o argomenti target. Un'origine `.test` è collegata esclusivamente al server `httptest` tramite il broker con IP `127.0.0.1` fissato. Qt WebEngine usa profilo senza persistenza e proxy HTTP autenticato su loopback. L'interceptor blocca origine/percorso/metodo non ammessi, worker e WebSocket; il proxy e il broker verificano di nuovo le richieste effettive. La fixture include documento, script, `fetch`, immagine fuori scope e redirect fuori scope. Il test riesce solo se documento/script/API sono arrivati al target locale senza header di autenticazione, le richieste conteggiate coincidono e l'immagine è stata negata. L'esito JavaScript è un segnale della fixture, non un'evidenza affidabile proveniente da una pagina arbitraria.

**Prova locale del 26 settembre 2026, Apple Silicon/macOS 26.6.2, Qt WebEngine 6.11.2:** il comando ha restituito `PASS`. Il job CI macOS 26 è separato e va verificato sul commit del branch e del merge. Nessun collaudo equivalente è ancora documentato su Windows o Linux. Il laboratorio non è una funzionalità GUI, non ha limite di processi/memoria, egress indipendente, HTTPS o sessioni; non soddisfa il criterio browser M3.
