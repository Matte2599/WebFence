# M3 — Browser helper resource limits

[Italiano](../it/M3-BROWSER-RESOURCE-LIMITS.md) · [ADR-011](ADR-011-BROWSER-HELPER.md) · [M3 status](M3-VALIDATION.md) · [Roadmap](../ROADMAP.md)

The Qt lab calls `browser.ApplyHelperResourceLimits` in the child process before reading configuration, creating Qt or starting descendants. A failure to apply the limits stops the child; the parent returns `browser_helper_failed` without forwarding output or configuration. More restrictive inherited limits are never raised. The parent retains its deadline, output limit and process-tree termination.

On macOS and Linux, the child lowers both the hard and current open-file limit to **1024**, maximum created-file size to **64 MiB**, and core-dump size to **zero**. Descendants inherit these limits. A synthetic test checks the values in the child and one descendant and confirms the parent keeps its own limits; the local macOS Qt fixture still passes.

**Limits of this evidence:** no memory limit is applied on macOS or Linux. macOS 26.6.2 rejected `RLIMIT_AS` with `EINVAL`; the first Linux CI run with a 16 GiB `RLIMIT_AS` loaded the document without executing its script. [V8 typically reserves about 1 TiB of virtual address space for its sandbox](https://chromium.googlesource.com/v8/v8.git/+/HEAD/src/sandbox/GLOSSARY.md), so the virtual-address limit was removed; the exact cause of the Linux failure was not proven. This block does not cap the total process count: `RLIMIT_NPROC` counts processes for the entire user, rather than just this helper. The helper is trusted code applying limits before handling pages; this is not a network or file-system sandbox. The existing Windows Job Object remains the separate mechanism described in ADR-011. This block does not enable the browser in the desktop.
