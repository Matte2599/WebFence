# ADR-022 — Headless startup inside an outer OS sandbox

[Italiano](../it/ADR-022-HEADLESS-OUTER-SANDBOX.md) · [Required boundary](ADR-015-BROWSER-PLATFORM-BOUNDARY.md) · [Trials](M3-BROWSER-PLATFORM-FEASIBILITY.md)

Date: 2026-09-29. Status: **startup experiment; product adoption open**.

## Problem

Desktop Chrome requires services unavailable within the tested boundaries: LaunchServices on macOS and Crashpad on Windows. The official `chrome-headless-shell` runtime separates the headless engine from the Chrome application. The trial pins version **154.0.8037.57**, Chromium revision `73c14f6228d7cd537c855007e8f88678969cc0eb`; the archive is verified with SHA-256 before execution, and no installed browser is modified.

On macOS, packaging it as an `.app` triggers Chromium application framework/resource and helper lookups. A `.bundle` preserves standalone runtime paths. Startup also requires the browser's private Mach rendezvous. After resolving those obstacles, children exit with `sandbox initialization failed: Operation not permitted`: App Sandbox is already active when Chromium attempts a second Seatbelt confinement.

The Windows debugger trial locates the pinned runtime failure at RVA `0x254a169`: the path ends in the check following `BrokerServices::CreateAlternateDesktop(kAlternateWinstation)`. The [revision's source](https://github.com/chromium/chromium/blob/73c14f6228d7cd537c855007e8f88678969cc0eb/sandbox/policy/sandbox.cc) prepares a window station for the internal sandbox; the trial is already inside AppContainer. This explains more of the headless failure; it does not automatically attribute the earlier Edge/Chrome crashes to the same cause.

## Experimental decision

The lab tests **App Sandbox/AppContainer as the OS boundary around the whole browser tree**, outside the desktop and WebFence broker. Only this experimental path uses `--no-sandbox` to avoid the incompatible internal sandbox initialization. It does not provide the same layered protection as the Linux lab: Chromium's renderer/browser separation is lost. The flag must not be added to the desktop, general launchers or Linux trials. A runtime launched outside the boundary is not an acceptable outcome.

In the macOS probe:

- the browser is signed with App Sandbox, without network capabilities; the helper is signed to inherit the sandbox;
- a trusted waiting process reserves the PID before signing; Mach entitlements permit only the exact `org.chromium.Chromium.MachPortRendezvousServer.<PID>` name, without wildcards, LaunchServices or network permissions;
- CDP uses inherited pipes; reduced environment, temporary profile, bounded time/frames/events/diagnostics and process-group termination;
- the original runtime control executes DOM and reaches synthetic HTTP/files; the confined runtime must execute DOM, deny HTTP/outside files and read its private file;
- `sandbox_check` requires an OS sandbox for the browser, renderer, GPU and network service reported by the runtime during the fixture.

Ad hoc signing for each PID is a laboratory technique, **not a signed/notarized distribution solution**. This code does not implement a product browser launcher. Stable packaging, process identity under hostile content, HTTPS broker, local services, aggregate quotas, revocation and desktop integration remain open. This trial closes no M3 criterion. The author's requirement to support all three OSs is unchanged.

## Dependencies and limitations

The macOS probe accepts only the pinned ARM64 archive, SHA-256 `9ba4d8a9732bd7e431ce9009d1bbe1ccf7f8c5de2a9781054f500a9fe898124d`. The Windows probe uses the win64 archive, SHA-256 `bcc91b4d0f83a5457fc6ca7941fd65f2349775523d961be88526c6a66560dc75`; the Windows ARM64 runner therefore runs an emulated x64 runtime. Runtime licensing and packaging must be reviewed before distribution; binaries and archives are not committed.

Sources: [Chrome headless shell](https://developer.chrome.com/docs/automation-and-testing/headless-chrome-shell), [Chromium macOS sandbox](https://chromium.googlesource.com/chromium/src/+/main/sandbox/mac/), [Apple temporary entitlements](https://developer.apple.com/library/archive/documentation/Miscellaneous/Reference/EntitlementKeyReference/Chapters/AppSandboxTemporaryExceptionEntitlements.html), [AppContainer IPC](https://learn.microsoft.com/en-us/windows/apps/develop/communication/interprocess-communication).
