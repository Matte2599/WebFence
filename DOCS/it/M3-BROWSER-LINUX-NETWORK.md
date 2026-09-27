# M3 — Prova Linux di rete negata al helper Qt

[English](../en/M3-BROWSER-LINUX-NETWORK.md) · [ADR-014](ADR-014-LINUX-BROWSER-NETWORK.md) · [Prova socket Unix](M3-BROWSER-UNIX-SCHEME.md) · [Stato M3](M3-VALIDATION.md)

`ApplyHelperNetworkIsolation` installa nel secondo helper Linux un filtro seccomp prima di Qt. Il test in un processo isolato verifica che TCP IPv4/IPv6 verso loopback fallisca con `EPERM`, anche da thread Go creati prima del filtro; controlla inoltre che un socket Unix locale funzioni e che un discendente non riacquisti socket INET. Non esegue scansioni. Il laboratorio Qt prova poi che documento e script sintetici attraversino ancora il proxy Unix sotto il filtro; il job Ubuntu 24.04 fornisce la prova su Qt WebEngine reale.

Verifica locale del 27 settembre 2026: test Linux ARM64 crosscompilato e avviato in un container Debian con rete disponibile, `TestLinuxNetworkIsolationAndInheritance` passato. La prova Qt macOS è passata dopo la modifica, senza installare il filtro Linux. La CI Linux del branch e del merge va registrata nella [matrice M3](M3-VALIDATION.md) dopo l'esecuzione.

**Copertura:** il filtro opera solo nel percorso sperimentale con schema Qt, su Linux AMD64/ARM64. Qt 6.4 nel job non permette di provare `fetch` sullo schema; il percorso HTTP precedente lo prova senza questo filtro. Il filtro consente socket Unix arbitrari e non prova che un servizio locale non possa inoltrare rete. Mancano ancora un confine completo, limiti aggregati e gli altri target OS.
