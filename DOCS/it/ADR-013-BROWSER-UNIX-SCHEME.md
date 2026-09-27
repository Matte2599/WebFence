# ADR-013 — Socket Unix e schema Qt nel laboratorio browser

[English](../en/ADR-013-BROWSER-UNIX-SCHEME.md) · [Laboratorio](M3-BROWSER-UNIX-SCHEME.md) · [ADR-011](ADR-011-BROWSER-HELPER.md) · [Stato M3](M3-VALIDATION.md)

Data: 2026-09-27. Stato: **esperimento adottato solo nel laboratorio sintetico**.

## Decisione

Un secondo percorso del laboratorio macOS collega il genitore e l'helper Qt con un socket Unix in una directory privata, usando lo stesso gate, broker e segreto effimero del proxy HTTP. L'helper registra `wfsite` come schema Qt prima di creare l'applicazione e inoltra solo tre risorse GET della fixture al genitore. Questo consente di esercitare documento, script esterno e `fetch` senza un listener TCP per quel percorso. La prova precedente con proxy HTTP rimane per controllare redirect e richieste fuori scope.

Il flag Qt `FetchApiAllowed`, disponibile da Qt 6.6, è necessario per `fetch` sullo schema personalizzato. MIQT 0.14.0 non espone il nome del flag: il laboratorio usa il valore Qt `0x100`. Il job Ubuntu 24.04 usa Qt 6.4, perciò esegue la prova HTTP precedente e i test del proxy Unix, ma non questo percorso Qt.

## Confini e limiti

Il socket Unix non impedisce a Qt o ai suoi processi di aprire connessioni TCP: non costituisce un confine di egress. Il gestore dello schema usa una lista di tre path della fixture e non media un sito arbitrario. `QWebEngineUrlRequestJob::reply` espone MIME e body ma non riproduce fedelmente status e header HTTP; lo schema cambia anche l'origine web. Per queste ragioni non adottiamo ancora questo percorso nel desktop e non lo presentiamo come sostituto del proxy HTTP(S).

Prima della prima voce M3 servono un meccanismo OS che neghi ogni uscita diretta dal gruppo browser, una mediazione fedele e sotto policy di HTTP(S), quote di memoria/processi e prove negative sulle piattaforme target. La scelta del meccanismo dovrà avere un ADR e collaudi propri; i flag Chromium e l'interceptor Qt non bastano.
