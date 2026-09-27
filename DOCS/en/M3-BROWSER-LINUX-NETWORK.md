# M3 — Linux trial of denied Qt helper networking

[Italiano](../it/M3-BROWSER-LINUX-NETWORK.md) · [ADR-014](ADR-014-LINUX-BROWSER-NETWORK.md) · [Unix socket trial](M3-BROWSER-UNIX-SCHEME.md) · [M3 status](M3-VALIDATION.md)

`ApplyHelperNetworkIsolation` installs a seccomp filter in the second Linux helper before Qt. A later revision also denies `connect`, `sendto` with an address, `sendmmsg` and Unix datagram socket creation. The parent passes up to four already connected Unix proxy descriptors; the child uses them without opening new connections. `sendmsg` and destination-free `sendto` remain available on the internal stream/seqpacket IPC needed by Qt. A separate-process test checks `EPERM` for IPv4/IPv6 TCP, new connections to a local Unix service and `sendto` with an address; it also checks preconnected IPC, a socketpair, existing threads and a descendant. It performs no scan.

Local verification on September 27, 2026: the Linux ARM64 test was cross-compiled and run in a Debian container with networking available; `TestLinuxNetworkIsolationAndInheritance` passed. The macOS Qt trial passed after this change without installing the Linux filter. [Branch `d640d0f` CI](https://github.com/Matte2599/WebFence/actions/runs/36312923254) and [squashed `main` `9a4f4a6` CI](https://github.com/Matte2599/WebFence/actions/runs/36313511877) passed, including the Linux Qt lab.

**Coverage:** the filter applies only to the experimental Qt scheme path on Linux AMD64/ARM64. Qt 6.4 in the job cannot exercise `fetch` on this scheme; the previous HTTP path exercises it without this filter. The initial filter permitted arbitrary Unix connections; the revision preconnects the broker and denies new connections.

**Preconnected descriptor revision:** the Linux ARM64 container test and macOS Qt lab regression pass locally; the revised Linux Qt/CI trial is still pending. The Linux scheme helper uses Qt offscreen because `connect` also denies the display server. This does not prove desktop integration or complete egress control: the first HTTP path is unfiltered, there is no filesystem sandbox, and aggregate quotas/HTTP(S)/macOS/Windows remain open.
