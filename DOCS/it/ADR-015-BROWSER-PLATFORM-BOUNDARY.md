# ADR-015 — Confine browser sulle piattaforme M3

[English](../en/ADR-015-BROWSER-PLATFORM-BOUNDARY.md) · [Prove](M3-BROWSER-PLATFORM-FEASIBILITY.md) · [Stato M3](M3-VALIDATION.md) · [ADR-014](ADR-014-LINUX-BROWSER-NETWORK.md)

Data: 2026-09-27. Stato: **proposta di percorso, non adozione di un nuovo motore**.

## Contesto

Il desktop rimane Go con Qt Widgets/MIQT, come scelto dall'autore. Il laboratorio Qt WebEngine è separato dal prodotto e dimostra solo un sottoinsieme sintetico. La prima voce M3 richiede che ogni richiesta di documento, risorsa e redirect resti sotto scope e budget, che il browser non possa aggirare il broker e che l'albero di processi sia limitato sulle piattaforme dichiarate. Interceptor e proxy non impongono da soli il confine OS.

Le [prove di fattibilità](M3-BROWSER-PLATFORM-FEASIBILITY.md) mostrano tre ostacoli distinti: su macOS il Qt WebEngine del laboratorio non parte in un bundle App Sandbox senza rete; una prova WebKit con schema personalizzato richiede il permesso di rete in uscita per caricare la pagina; su Windows Qt WebEngine non compila con la toolchain MinGW del desktop. Su Linux il filtro seccomp esistente nega nuovi socket INET ma lascia accessibili altri socket Unix e il Qt 6.4 della baseline non supporta `FetchApiAllowed`. Questi esiti non dimostrano l'impossibilità di ogni alternativa.

## Decisione provvisoria

Non abilitare il browser nella GUI né chiudere la prima voce M3. Mantenere separati desktop e helper browser. Prima di scegliere un runtime per ciascun OS, esigere una prova ripetibile con: egress diretto e tramite servizi locali negato; traffico HTTP(S), redirect e subresource mediato dal broker senza allargare scope; budget e revoca; quote aggregate di memoria/processi; sessione e origine web preservate; packaging e collaudo sulle piattaforme target. Un helper che richieda privilegi aggiuntivi o virtualizzazione deve dichiararli prima dell'adozione e del supporto.

La sola possibilità di usare un motore nativo per OS non costituisce una decisione di implementarlo. Un'eventuale revisione del requisito di isolamento o del supporto di piattaforma richiede una decisione esplicita dell'autore e una modifica della roadmap, non una spunta ottenuta dalle fixture correnti.

Fonti primarie: [Qt WebEngine su macOS e Windows](https://doc.qt.io/qt-6/qtwebengine-platform-notes.html), [flag Qt 6.6 per Fetch API](https://doc.qt.io/qt-6/qwebengineurlscheme.html), [App Sandbox e rete](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.security.network.client), [Landlock ABI per socket Unix](https://cdn.kernel.org/doc/html/latest/userspace-api/landlock.html).
