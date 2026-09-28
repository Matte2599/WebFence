# ADR-015 — M3 browser boundary across platforms

[Italiano](../it/ADR-015-BROWSER-PLATFORM-BOUNDARY.md) · [Trials](M3-BROWSER-PLATFORM-FEASIBILITY.md) · [M3 status](M3-VALIDATION.md) · [ADR-014](ADR-014-LINUX-BROWSER-NETWORK.md)

Date: 2026-09-27. Status: **OS isolation requirement confirmed by the author; runtime choices remain open**.

## Context

The desktop remains Go with Qt Widgets/MIQT, as chosen by the author. The Qt WebEngine lab is separate from the product and proves only a synthetic subset. The first M3 item requires every document, resource and redirect to remain under scope and budget, the browser to be unable to bypass the broker, and the process tree to be bounded on declared platforms. An interceptor and proxy alone do not enforce an OS boundary.

The [feasibility trials](M3-BROWSER-PLATFORM-FEASIBILITY.md) reveal three separate obstacles: on macOS the lab's Qt WebEngine does not start in a network-denied App Sandbox bundle; a custom-scheme WebKit trial needs outbound network permission to load; on Windows Qt WebEngine does not compile with the desktop's MinGW toolchain. On Linux the original seccomp filter denied new INET sockets but left other Unix sockets reachable; the [revision](M3-BROWSER-LINUX-NETWORK.md) tries parent-connected broker IPC and denies new `connect` calls. Baseline Qt 6.4 lacks `FetchApiAllowed`. These observations do not prove every alternative impossible.

## Decision

The author chose to preserve operating-system-level isolation. The browser remains a separate helper from the Go/Qt desktop; on each platform, enable the feature only after a repeatable trial shows: denial of direct egress and egress through local services; HTTP(S), redirects and subresources mediated by the broker without expanding scope; budgets and revocation; aggregate memory/process quotas; preserved session and web origin; packaging and trials. Until then, do not enable a browser in the GUI or close the first M3 item. A helper requiring extra privileges or virtualization must declare those requirements before adoption and support.

The possibility of using an OS-native engine is not a decision to implement one. Any relaxation of the isolation requirement or platform support needs an explicit author decision and roadmap change, rather than a checkmark based on the current fixtures.

The later [Linux CDP HTTPS fixture](ADR-019-CDP-HTTPS-BOUNDARY.md) checks TLS and origin in a trial container; it does not change the gate for enabling the desktop browser on any platform.

Primary sources: [Qt WebEngine on macOS and Windows](https://doc.qt.io/qt-6/qtwebengine-platform-notes.html), [Qt 6.6 Fetch API flag](https://doc.qt.io/qt-6/qwebengineurlscheme.html), [App Sandbox networking](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.security.network.client), [Landlock Unix socket ABI](https://cdn.kernel.org/doc/html/latest/userspace-api/landlock.html).
