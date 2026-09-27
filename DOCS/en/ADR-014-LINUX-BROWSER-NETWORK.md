# ADR-014 — Linux socket filter in the Qt lab

[Italiano](../it/ADR-014-LINUX-BROWSER-NETWORK.md) · [Trial](M3-BROWSER-LINUX-NETWORK.md) · [ADR-013](ADR-013-BROWSER-UNIX-SCHEME.md) · [M3 status](M3-VALIDATION.md)

Date: 2026-09-27. Status: **adopted only in the Linux lab**.

## Decision

Before creating Qt, the second Linux helper applies `no_new_privs` and a seccomp BPF filter with `TSYNC` to every thread. Descendants inherit the filter: it permits `AF_UNIX` stream/seqpacket sockets and `sendmsg` on established internal IPC; it denies datagram socket creation, `connect`, `sendmmsg`, `sendto` with an explicit destination, other families and `io_uring_setup`, and terminates on an unexpected syscall ABI. `sendto` without a destination remains available on connected sockets: Qt WebEngine uses it for internal IPC. The parent retains the broker's TCP sockets and opens four connections to the Unix proxy from [ADR-013](ADR-013-BROWSER-UNIX-SCHEME.md) **before** starting the helper. It passes these on descriptors 3–6, with no INET sockets. If the filter cannot be installed, the helper fails before loading content.

`TSYNC` matters because the Go runtime can have several threads before installation: a filter on the calling thread alone would leave network paths. The lab checks `EPERM` for IPv4/IPv6 sockets, new Unix connections and `sendto` with a destination, plus preconnected IPC, an internal socketpair and filter inheritance by a descendant. The second helper uses Qt's offscreen plugin on Linux: it cannot open a display-server socket after filtering. Ubuntu 24.04's Qt 6.4 lacks `FetchApiAllowed`: the scheme path checks a document and script, while the previous HTTP trial still checks redirects, subresources and `fetch`.

## Limitations

The filter prevents direct connections to new Unix sockets and creation of INET sockets, but it is not a filesystem sandbox or general proof against every IPC route or inherited descriptor. The browser is not integrated into the desktop; the first HTTP helper remains unfiltered. The revised Linux Qt trial must pass CI before treating this experimental path as verified. Faithful HTTP(S), aggregate quotas, macOS/Windows trials and desktop integration are still required for the first M3 item.

Sources: [Linux seccomp BPF](https://kernel.org/doc/html/latest/userspace-api/seccomp_filter.html), [`TSYNC` semantics](https://man7.org/linux/man-pages/man2/seccomp.2.html), [Qt `FetchApiAllowed` flag](https://doc.qt.io/qt-6/qwebengineurlscheme.html).
