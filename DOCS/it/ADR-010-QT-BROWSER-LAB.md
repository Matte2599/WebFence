# ADR-010 — Qt WebEngine nel laboratorio browser M3

[English](../en/ADR-010-QT-BROWSER-LAB.md) · [Laboratorio M3](M3-BROWSER-LAB.md) · [ADR-002](ADR-002-GUI.md) · [ADR-009](ADR-009-BROWSER-HTTP-BOUNDARY.md)

Data: 2026-09-26. Stato: **adottato per il laboratorio M3**, non per una funzione desktop distribuibile.

## Decisione e prova

Usiamo Qt WebEngine 6.11.2 tramite MIQT 0.14.0 nel laboratorio sintetico. Mantiene lo stack Qt scelto dall'autore e offre un profilo [off-the-record](https://doc.qt.io/qt-6/qwebengineprofile.html), un [interceptor delle richieste](https://doc.qt.io/qt-6/qwebengineurlrequestinterceptor.html) e [supporto al proxy Qt Network](https://doc.qt.io/qt-6/qtwebengine-overview.html#proxy-support). Il laboratorio crea una fixture HTTP locale, la espone con origine `.test` tramite IP loopback fissato e usa il proxy autenticato M3. Una pagina carica script, esegue `fetch` e tenta una subresource di altra origine; la prova locale ha osservato i tre GET ammessi attraverso proxy/broker e il blocco della subresource. Un redirect verso origine esclusa viene respinto dal broker. Nessun target esterno è previsto dal comando.

Il laboratorio ha un flag di build dedicato e non entra nel desktop o nei pacchetti. La CI macOS 26 verifica separatamente la prova; i suoi esiti vanno letti nel run del commit pertinente. La prova locale non misura sicurezza della sandbox di Chromium né sostituisce i collaudi sulle altre piattaforme.

## Limiti e riesame

Il prototipo usa profilo effimero e interceptor nello stesso processo dell'eseguibile di prova. Non impone ancora un limite di processi/memoria, un firewall egress indipendente o la mediazione di HTTPS. La scelta di Qt WebEngine per il prodotto richiede un helper separato, IPC che non esponga la credenziale proxy, collaudi di contenimento su macOS/Windows/Linux, packaging e revisione dei materiali di licenza aggiuntivi. Se questi vincoli non saranno soddisfatti, il driver andrà riesaminato in un nuovo ADR. Il laboratorio non chiude la prima voce M3.
