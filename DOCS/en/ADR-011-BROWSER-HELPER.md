# ADR-011 — Supervised Qt helper for the M3 lab

[Italiano](../it/ADR-011-BROWSER-HELPER.md) · [Lab](M3-BROWSER-LAB.md) · [ADR-010](ADR-010-QT-BROWSER-LAB.md) · [Roadmap](../ROADMAP.md)

Date: 2026-09-27. Status: **adopted for the synthetic lab only**; it does not authorize browser use in the desktop.

## Decision

The lab starts Qt WebEngine in a second process, initially without navigating. The parent creates a private temporary directory, restricts inherited environment variables, installs the process-tree termination boundary and only then sends the `.test` origin, loopback proxy endpoint and ephemeral credential over `stdin`. No secret appears in arguments or output. `stdout` returns only synthetic flags/counts. The parent imposes a 25-second lifetime and 64 KiB combined output limit; timeout, failure or excess terminates the tree. macOS/Linux use a process group; the child also applies [resource limits](M3-BROWSER-RESOURCE-LIMITS.md) before reading configuration. Windows uses a Job Object with kill-on-close, at most 16 processes and a 1 GiB job memory limit. The Chromium flag for four renderers is only a hint, not a hard limit. Child configuration accepts only `http://site.test:<port>` as origin and `http://127.0.0.1:<port>` as proxy; the parent gate and broker still validate actual requests.

## Evidence and limits

Supervisor tests use a synthetic executable to exercise excessive output, failure, a private environment variable and a descendant surviving timeout. The local macOS Qt trial on September 27, 2026 returned `PASS` after moving it to the child. Branch and merge CI must be checked for their respective commits; the `internal/browser` test binary was cross-compiled for Windows locally, but the Windows Job Object has not been executed here.

The separate process does not prove an independent egress firewall or filesystem sandbox. The added Unix limits are per process and exclude memory on macOS; the aggregate process count still has no hard quota. The browser is not part of the desktop, does not mediate HTTPS and has no sessions. The proxy credential remains in child memory during the trial. Negative runtime trials on Windows and Linux, containment of alternate network paths, WebEngine packaging and Qt license-material review remain necessary before the first M3 criterion. This ADR alone does not close it.
