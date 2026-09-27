# M3 — Prova browser con socket Unix e schema Qt

[English](../en/M3-BROWSER-UNIX-SCHEME.md) · [ADR-013](ADR-013-BROWSER-UNIX-SCHEME.md) · [Laboratorio Qt](M3-BROWSER-LAB.md) · [Stato M3](M3-VALIDATION.md)

`NewObservedUnixProxy` accetta una directory locale privata (`0700`), apre `broker.sock` e serve le richieste tramite il gate e il broker esistenti. Non pubblica un endpoint TCP. Il test verifica una GET ammessa, il rifiuto di un'origine estranea, le osservazioni redatte e il rifiuto di una directory non privata. Non è una sandbox del processo chiamante.

Su macOS con Qt 6.11.2, il laboratorio opzionale avvia un secondo helper supervisionato. La configurazione è ricevuta su `stdin` e limita origine, segreto e socket. Uno schema Qt serve documento, script e `fetch` di tre path sintetici; ciascuna richiesta attraversa il proxy Unix e i conteggi di gate/broker/target sono controllati. Il primo helper continua a verificare il proxy HTTP, la navigazione DOM, le subresource e il blocco di redirect/origini estranee. Nessun target esterno viene contattato.

Verifica locale: `go test -race ./internal/browser -run 'TestUnixProxy|TestProxy' -count=1`, `CGO_CXXFLAGS='-O0 -g0 -std=c++17' go test -tags=m3browserlab ./experiments/m3-browser -count=1` e `CGO_CXXFLAGS='-O0 -g0 -std=c++17' QT_QPA_PLATFORM=offscreen go run -tags=m3browserlab ./experiments/m3-browser` sono passati su Apple Silicon/macOS 26.6.2 il 27 settembre 2026. La CI del branch e del merge sarà registrata nella [matrice M3](M3-VALIDATION.md) dopo l'esecuzione.

Il percorso con schema Qt non funziona su Qt 6.4 del job Linux per l'assenza di `FetchApiAllowed`; lì resta la prova HTTP. Lo schema non conserva semantica HTTP completa e non impedisce al browser di usare socket TCP propri. L'isolamento indipendente, HTTPS, quote aggregate, Windows e integrazione desktop restano aperti.
