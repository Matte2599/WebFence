# ADR-014 — Filtro Linux dei socket nel laboratorio Qt

[English](../en/ADR-014-LINUX-BROWSER-NETWORK.md) · [Prova](M3-BROWSER-LINUX-NETWORK.md) · [ADR-013](ADR-013-BROWSER-UNIX-SCHEME.md) · [Stato M3](M3-VALIDATION.md)

Data: 2026-09-27. Stato: **esperimento adottato solo nel laboratorio Linux**.

## Decisione

Prima di creare Qt, il secondo helper Linux applica `no_new_privs` e un filtro seccomp BPF con `TSYNC` a tutti i thread. Il filtro è ereditato dai discendenti: permette solo socket `AF_UNIX` stream/seqpacket e IPC `sendmsg` su connessioni già stabilite; nega la creazione di socket datagram, `connect`, `sendmmsg`, `sendto` con destinazione esplicita, altre famiglie e `io_uring_setup`, e termina in caso di ABI inattesa. `sendto` senza indirizzo resta disponibile sui socket già connessi: Qt WebEngine lo usa per il proprio IPC. Il genitore conserva i socket TCP del broker e apre quattro connessioni al proxy Unix dell'[ADR-013](ADR-013-BROWSER-UNIX-SCHEME.md) **prima** di avviare l'helper. Le passa sui descrittori 3–6, senza consegnare socket INET. Se il filtro non si installa, l'helper fallisce prima di caricare contenuti.

`TSYNC` è essenziale perché il runtime Go può avere più thread prima dell'installazione: un filtro applicato al solo thread chiamante lascerebbe vie di rete. Il laboratorio verifica `EPERM` per socket IPv4/IPv6, nuove connessioni Unix e `sendto` con destinazione, oltre a IPC preconnesso, socketpair interno e filtro ereditato da un discendente. Il secondo helper usa il plugin Qt offscreen su Linux: dopo il filtro non può aprire il socket del display server. Su Ubuntu 24.04 il Qt 6.4 disponibile non ha `FetchApiAllowed`: il percorso dello schema controlla documento e script, mentre la prova HTTP precedente continua a controllare redirect, subresource e `fetch`.

## Limiti

Il filtro impedisce la connessione diretta a nuovi socket Unix e la creazione di socket INET, ma non è una sandbox di file system né una prova generale contro ogni via di IPC o descrittore ereditato. Il browser non è integrato nel desktop; il primo helper HTTP resta senza filtro. Il [job Qt Linux della revisione](https://github.com/Matte2599/WebFence/actions/runs/36317757369) passa, ma servono ancora HTTP(S) fedele, quote aggregate, prove macOS/Windows e integrazione desktop prima della prima voce M3.

Fonti: [Linux seccomp BPF](https://kernel.org/doc/html/latest/userspace-api/seccomp_filter.html), [semantica `TSYNC`](https://man7.org/linux/man-pages/man2/seccomp.2.html), [flag Qt `FetchApiAllowed`](https://doc.qt.io/qt-6/qwebengineurlscheme.html).
