# ADR-014 — Filtro Linux dei socket nel laboratorio Qt

[English](../en/ADR-014-LINUX-BROWSER-NETWORK.md) · [Prova](M3-BROWSER-LINUX-NETWORK.md) · [ADR-013](ADR-013-BROWSER-UNIX-SCHEME.md) · [Stato M3](M3-VALIDATION.md)

Data: 2026-09-27. Stato: **esperimento adottato solo nel laboratorio Linux**.

## Decisione

Prima di creare Qt, il secondo helper Linux applica `no_new_privs` e un filtro seccomp BPF con `TSYNC` a tutti i thread. Il filtro è ereditato dai discendenti: consente solo la creazione di socket `AF_UNIX`, rifiuta con `EPERM` altre famiglie e `io_uring_setup`, e termina in caso di ABI inattesa. Il genitore conserva i socket TCP del broker e comunica con l'helper tramite il proxy Unix dell'[ADR-013](ADR-013-BROWSER-UNIX-SCHEME.md). Se il filtro non si installa, l'helper fallisce prima di caricare contenuti.

`TSYNC` è essenziale perché il runtime Go può avere più thread prima dell'installazione: un filtro applicato al solo thread chiamante lascerebbe vie di rete. Il supervisore passa al figlio solo pipe standard e non inoltra descrittori di rete. Il laboratorio verifica `EPERM` per socket IPv4/IPv6, IPC Unix funzionante e filtro ereditato da un processo discendente. Su Ubuntu 24.04 il Qt 6.4 disponibile non ha `FetchApiAllowed`: il percorso dello schema controlla documento e script, mentre la prova HTTP precedente continua a controllare redirect, subresource e `fetch`.

## Limiti

Il filtro impedisce la creazione diretta di socket INET nel gruppo, ma consente la connessione a **qualsiasi** socket Unix accessibile all'utente. Un servizio locale potrebbe quindi inoltrare traffico per conto del browser; seccomp non può ispezionare il path passato a `connect` con questo filtro. Non è una prova di egress indipendente completo, né una sandbox di file system. Il primo helper HTTP resta senza filtro; non è stato abilitato un browser nel desktop. Servono un confine OS anche sui socket Unix, HTTP(S) fedele, quote aggregate, prove macOS/Windows e integrazione desktop prima della prima voce M3.

Fonti: [Linux seccomp BPF](https://kernel.org/doc/html/latest/userspace-api/seccomp_filter.html), [semantica `TSYNC`](https://man7.org/linux/man-pages/man2/seccomp.2.html), [flag Qt `FetchApiAllowed`](https://doc.qt.io/qt-6/qwebengineurlscheme.html).
