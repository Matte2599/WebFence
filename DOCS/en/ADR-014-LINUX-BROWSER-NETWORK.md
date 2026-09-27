# ADR-014 — Linux socket filter in the Qt lab

[Italiano](../it/ADR-014-LINUX-BROWSER-NETWORK.md) · [Trial](M3-BROWSER-LINUX-NETWORK.md) · [ADR-013](ADR-013-BROWSER-UNIX-SCHEME.md) · [M3 status](M3-VALIDATION.md)

Date: 2026-09-27. Status: **adopted only in the Linux lab**.

## Decision

Before creating Qt, the second Linux helper applies `no_new_privs` and a seccomp BPF filter with `TSYNC` to every thread. Descendants inherit the filter: it allows creation of `AF_UNIX` sockets only, rejects other families and `io_uring_setup` with `EPERM`, and terminates on an unexpected syscall ABI. The parent retains the broker's TCP sockets and communicates with the helper through the Unix proxy from [ADR-013](ADR-013-BROWSER-UNIX-SCHEME.md). If the filter cannot be installed, the helper fails before loading content.

`TSYNC` matters because the Go runtime can have several threads before installation: a filter on the calling thread alone would leave network paths. The supervisor passes only standard pipes to the child, with no network descriptors. The lab checks `EPERM` for IPv4/IPv6 sockets, working Unix IPC and filter inheritance by a descendant process. Ubuntu 24.04's Qt 6.4 lacks `FetchApiAllowed`: the scheme path checks a document and script, while the previous HTTP trial still checks redirects, subresources and `fetch`.

## Limitations

The filter prevents direct INET socket creation in the group, but permits connections to **any** Unix socket accessible to the user. A local service could forward browser traffic; this seccomp filter cannot inspect the path passed to `connect`. It is not proof of complete independent egress or a filesystem sandbox. The first HTTP helper remains unfiltered; no browser is enabled in the desktop. An OS boundary for Unix sockets, faithful HTTP(S), aggregate quotas, macOS/Windows trials and desktop integration are still required for the first M3 item.

Sources: [Linux seccomp BPF](https://kernel.org/doc/html/latest/userspace-api/seccomp_filter.html), [`TSYNC` semantics](https://man7.org/linux/man-pages/man2/seccomp.2.html), [Qt `FetchApiAllowed` flag](https://doc.qt.io/qt-6/qwebengineurlscheme.html).
