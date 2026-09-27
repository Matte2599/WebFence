# M3 — Linux trial of denied Qt helper networking

[Italiano](../it/M3-BROWSER-LINUX-NETWORK.md) · [ADR-014](ADR-014-LINUX-BROWSER-NETWORK.md) · [Unix socket trial](M3-BROWSER-UNIX-SCHEME.md) · [M3 status](M3-VALIDATION.md)

`ApplyHelperNetworkIsolation` installs a seccomp filter in the second Linux helper before Qt. A separate-process test verifies that loopback IPv4/IPv6 TCP fails with `EPERM`, that local Unix IPC works and that a descendant cannot regain INET sockets. It performs no scan. The Qt lab then checks that a synthetic document and script still cross the Unix proxy under the filter; the Ubuntu 24.04 job provides a trial with real Qt WebEngine.

Local verification on September 27, 2026: the Linux ARM64 test was cross-compiled and run in a Debian container with networking available; `TestLinuxNetworkIsolationAndInheritance` passed. The macOS Qt trial passed after this change without installing the Linux filter. Branch and merged Linux CI will be recorded in the [M3 matrix](M3-VALIDATION.md) after they run.

**Coverage:** the filter applies only to the experimental Qt scheme path on Linux AMD64/ARM64. Qt 6.4 in the job cannot exercise `fetch` on this scheme; the previous HTTP path exercises it without this filter. The filter permits arbitrary Unix sockets and does not prove that a local service cannot forward traffic. A complete boundary, aggregate limits and the other target OSes remain open.
