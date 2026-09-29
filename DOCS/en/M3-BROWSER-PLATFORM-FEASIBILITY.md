# M3 — Feasibility of the browser boundary by platform

[Italiano](../it/M3-BROWSER-PLATFORM-FEASIBILITY.md) · [ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md) · [M3 status](M3-VALIDATION.md) · [Roadmap](../ROADMAP.md)

Local trials on September 27, 2026, on Apple Silicon/macOS 26.6.2, using only synthetic pages and loopback addresses. The initial trials used temporary probe programs and ad hoc signed bundles outside the repository; the later CDP trial uses versioned material but is not yet a product choice. No external target was contacted.

| Trial | Observed outcome | Limit of inference |
| --- | --- | --- |
| Minimal App Sandbox bundle without `network.client`, Go program opening TCP to `127.0.0.1:9` | `operation not permitted`; the signed control without sandbox runs. | Confirms TCP denial in the trial bundle, not a working browser engine. |
| Versioned App Sandbox bundle in `experiments/m3-macos-sandbox`, local Google Chrome started headless over CDP | The ad hoc signed child receives `EPERM`/`EACCES` on the loopback TCP canary; its container profile is created, but Chrome exits with `abort trap` before the `Browser.getVersion` response. | Shows child TCP denial and failed CDP startup in this configuration. It does not identify the abort cause, measure HTTPS broker/quotas, or prove that every macOS runtime is impossible. |
| Qt WebEngine 6.11.2, synthetic HTML page | Without App Sandbox, `LOADED true`; in the App Sandbox bundle, after bundling the Qt offscreen plugin, Chromium fails at `bootstrap_check_in org.chromium.Chromium.MachPortRendezvousServer.<pid>: Permission denied (1100)`. `QTWEBENGINE_DISABLE_SANDBOX=1` also fails. | Another packaging arrangement has not been ruled out; [Qt documentation](https://doc.qt.io/qt-6/qtwebengine-platform-notes.html) warns that App Sandbox interferes with Chromium initialization. |
| Native WebKit with a `wfsite` scheme, synthetic document, script and `fetch` | Passes without App Sandbox; the App Sandbox bundle without `network.client` times out without a loaded page; with `network.client` it passes; removing permission makes it time out again. | Outbound permission allows arbitrary TCP connections according to [Apple](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.security.network.client); it is not a page egress boundary. Not every WebKit configuration was explored. |
| Initial Linux ARM64 filter in a container and Ubuntu 24.04 job | The [seccomp filter](M3-BROWSER-LINUX-NETWORK.md) denies direct INET sockets while Unix IPC works; the initial branch's Linux Qt lab passed. | In this initial version, Unix sockets to other local services remained reachable; the job's Qt 6.4 does not test scheme `fetch`. |
| Versioned AppContainer probe in `experiments/m3-windows-appcontainer`, Windows Server 2022 amd64 and Windows 11 ARM64 CI on September 29, 2026 | On both runners the unrestricted parent reaches a loopback listener on a system-assigned port. The child confirms an AppContainer token with no network capabilities, is assigned to a Job Object capped at four processes and 512 MiB, and times out on the same TCP connection. | Shows only that this listener was unreachable in these configurations. It does not start a browser, exercise the limits under load, test every egress path or test Windows 11 x64/Windows 10. The timeout alone does not identify a specific blocking filter. |
| Windows 10/11 | No working confined browser runtime; the subsequent startup trial is described below. [Qt WebEngine](https://doc.qt.io/qt-6/qtwebengine-platform-notes.html) does not compile with the desktop's MinGW toolchain. | The supervisor's Job Object does not replace a Windows browser and network trial. |

Baseline Ubuntu 24.04.0 uses Linux kernel 6.8 ([Canonical](https://documentation.ubuntu.com/kteam-docs/public/reference/hwe-kernels.html)); Landlock restriction of pathname Unix socket resolution is available only from ABI 9 ([Linux kernel](https://cdn.kernel.org/doc/html/latest/userspace-api/landlock.html)). HWE updates can change the installed kernel, so capability must be detected at runtime. This matrix motivates [ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md), but does not close the first M3 item.

Run the versioned macOS probe with `sh experiments/m3-macos-sandbox/run.sh` from the repository root: it needs Go, `codesign`, and a local Chrome copy at the default path, overridable with `WF_M3_CHROME`. It ad hoc signs a temporary bundle without `network.client`, uses an OS-assigned loopback port, and contacts no external target. The command exits nonzero when CDP fails to start. macOS also creates a container for the trial identifier `org.webfence.m3-sandbox-probe`; the temporary bundle and profile are removed, while the system-managed directory may remain. The September 29, 2026 local result is the failure in the table, not a passing product test.

The Windows probe is built and run by the `m3-windows-appcontainer-lab` CI job on [Windows Server 2022 and Windows 11 ARM64](https://github.com/actions/runner-images?tab=readme-ov-file#available-images). It creates a temporary AppContainer profile without capabilities, uses only the synthetic local listener, and requires a successful unrestricted control connection before testing the child. It requests profile deletion on normal exit. [Microsoft's AppContainer guidance](https://learn.microsoft.com/en-us/windows/win32/secauthz/implementing-an-appcontainer) describes process creation with explicit capabilities; [AppContainer loopback](https://learn.microsoft.com/en-us/windows/security/operating-system-security/network-security/windows-firewall/troubleshooting-uwp-firewall) has its own restrictions. Thus the observed timeout alone does not establish Internet denial or a viable browser runtime.

## Windows browser startup — September 29, 2026

The probe now accepts `--browser` followed by an absolute Edge or Chrome path. It first confirms the existing AppContainer canary, then runs a browser control outside AppContainer and repeats inside AppContainer, without network capabilities. Both modes use separate temporary profiles, an explicit inherited-handle list for CDP and diagnostics, a **16-process / 1 GiB** Job Object assigned before resuming the process, and a 20-second deadline. The suspended browser's token is checked. Chromium's sandbox is not disabled. The test awaits `Browser.getVersion`, creates an `about:blank` page and verifies a DOM change through JavaScript.

| Runner | Browser reported by control | CDP/DOM control | Browser in AppContainer |
| --- | --- | --- | --- |
| Windows Server 2022 amd64 | Edge 152.0.4191.66 | Passed | CDP pipe closed before version response; exit `0xc0000005`. Cause unidentified. |
| Windows 11 ARM64 | Edge 153.0.4234.48 | Passed | CDP pipe closed before version response; exit `0xc0000005`. Cause unidentified. |
| Both runners | Chrome 153.0.8010.53 | Passed | Crashpad reports `CreateNamedPipe: Access is denied (0x5)`; exit `0x80000003` before CDP version response. |

Evidence: [comparative branch run](https://github.com/Matte2599/WebFence/actions/runs/36561733556), commit `a60fb62`. The architecture refers to the runner and Go probe; installed browser architectures were not verified. The log identifies Chrome's failed operation, but does not prove that fixing it alone produces a safe, working runtime. [AppContainer IPC restrictions](https://learn.microsoft.com/en-us/windows/apps/develop/communication/interprocess-communication) require attention to pipe namespaces. No flag grants network capabilities or disables isolation.

Ordinary CI runs protocol tests, the canary and positive controls through `run-browser.ps1 -ControlOnly`. The manual **M3 Windows browser feasibility (known unsupported)** workflow also attempts confined startup: a missing response remains a failed job. Green ordinary CI **does not certify this experimental trial**. To reproduce on Windows, from the repository root:

```powershell
$env:CGO_ENABLED = '0'
go build -o m3-appcontainer.exe ./experiments/m3-windows-appcontainer
if ($LASTEXITCODE -ne 0) { throw 'Lab compilation failed' }
./experiments/m3-windows-appcontainer/run-browser.ps1
```

The command discovers already installed browsers at standard paths and returns an error if no browser is found or a trial fails. It installs no software. The temporary profile grants access only to the created AppContainer and receives a low integrity label; installed browser directories remain unchanged. Logs and CDP frames are bounded. Protocol tests cover oversized frames, closed pipes, CDP errors, missing sessions/targets and failed DOM execution.

**Outcome: the Windows runtime remains unavailable.** Engine IPC initialization inside the boundary must be fixed and retested before connecting the HTTPS broker, origin/session behavior and negative tests. General egress, local services, quotas under load and packaging remain unverified. This block neither closes M3 nor constitutes its final comprehensive test.

## Subsequent macOS diagnosis

On September 29, the local crash report and OS events for the 11:02:50 trial (Chrome PID 75546) were inspected. The sandbox denies the lookup of `com.apple.coreservices.launchservicesd`; immediately afterward, `_RegisterApplication` reports that it cannot obtain the ASN identifier and calls `abort`. The report confirms `SIGABRT` in the HIServices / `TransformProcessType` stack. This identifies the recorded reason for that startup: it does not prove that granting service access resolves every obstacle or preserves denial of egress through local services. No entitlement was expanded and no raw system report was committed. The macOS runtime remains open.
