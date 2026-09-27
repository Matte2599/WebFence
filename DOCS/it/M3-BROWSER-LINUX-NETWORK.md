# M3 — Prova Linux di rete negata al helper Qt

[English](../en/M3-BROWSER-LINUX-NETWORK.md) · [ADR-014](ADR-014-LINUX-BROWSER-NETWORK.md) · [Prova socket Unix](M3-BROWSER-UNIX-SCHEME.md) · [Stato M3](M3-VALIDATION.md)

`ApplyHelperNetworkIsolation` installa nel secondo helper Linux un filtro seccomp prima di Qt. La revisione successiva nega anche `connect`, `sendto`, `sendmmsg` e la creazione di socket Unix datagram. Il genitore passa fino a quattro connessioni Unix già aperte verso il proxy; il figlio le usa senza poter aprire connessioni nuove. `sendmsg` resta disponibile sui socket IPC interni stream/seqpacket necessari a Qt. Il test in un processo isolato verifica `EPERM` per TCP IPv4/IPv6, nuove connessioni a un servizio Unix locale e `sendto`; controlla inoltre IPC preconnesso, socketpair, thread preesistenti e discendente. Non esegue scansioni.

Verifica locale del 27 settembre 2026: test Linux ARM64 crosscompilato e avviato in un container Debian con rete disponibile, `TestLinuxNetworkIsolationAndInheritance` passato. La prova Qt macOS è passata dopo la modifica, senza installare il filtro Linux. [CI del branch `d640d0f`](https://github.com/Matte2599/WebFence/actions/runs/36312923254) e [CI dello squash `9a4f4a6` su `main`](https://github.com/Matte2599/WebFence/actions/runs/36313511877) verdi, inclusa la prova Qt Linux.

**Copertura:** il filtro opera solo nel percorso sperimentale con schema Qt, su Linux AMD64/ARM64. Qt 6.4 nel job non permette di provare `fetch` sullo schema; il percorso HTTP precedente lo prova senza questo filtro. La prima versione del filtro consentiva connessioni Unix arbitrarie; la revisione preconnette il broker e nega nuove connessioni.

**Revisione con connessioni preaperte:** il test Linux ARM64 in container e la regressione macOS del laboratorio Qt passano localmente; la prova Qt Linux/CI di questa revisione è ancora da verificare. L'helper Linux dello schema usa Qt offscreen perché `connect` nega anche il display server. Non è una prova di integrazione desktop o di egress completo: il primo percorso HTTP è senza filtro, non c'è sandbox dei file, e quote aggregate/HTTP(S)/macOS/Windows restano aperti.
