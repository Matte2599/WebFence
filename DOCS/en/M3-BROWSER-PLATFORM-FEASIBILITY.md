# M3 — Feasibility of the browser boundary by platform

[Italiano](../it/M3-BROWSER-PLATFORM-FEASIBILITY.md) · [ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md) · [M3 status](M3-VALIDATION.md) · [Roadmap](../ROADMAP.md)

Local trials on September 27, 2026, on Apple Silicon/macOS 26.6.2, using only synthetic pages and loopback addresses. The initial trials used temporary probe programs and ad hoc signed bundles outside the repository; the later CDP trial uses versioned material but is not yet a product choice. No external target was contacted.

| Trial | Observed outcome | Limit of inference |
| --- | --- | --- |
| Minimal App Sandbox bundle without `network.client`, Go program opening TCP to `127.0.0.1:9` | `operation not permitted`; the signed control without sandbox runs. | Confirms TCP denial in the trial bundle, not a working browser engine. |
| Versioned App Sandbox bundle in `experiments/m3-macos-sandbox`, local Google Chrome started headless over CDP | The ad hoc signed child receives `EPERM`/`EACCES` on the loopback TCP canary; its container profile is created, but Chrome exits with `abort trap` before the `Browser.getVersion` response. | Shows child TCP denial and failed CDP startup in this configuration. The subsequent OS diagnosis below identifies a denied LaunchServices lookup; it does not measure HTTPS broker/quotas or prove that every macOS runtime is impossible. |
| Qt WebEngine 6.11.2, synthetic HTML page | Without App Sandbox, `LOADED true`; in the App Sandbox bundle, after bundling the Qt offscreen plugin, Chromium fails at `bootstrap_check_in org.chromium.Chromium.MachPortRendezvousServer.<pid>: Permission denied (1100)`. `QTWEBENGINE_DISABLE_SANDBOX=1` also fails. | Another packaging arrangement has not been ruled out; [Qt documentation](https://doc.qt.io/qt-6/qtwebengine-platform-notes.html) warns that App Sandbox interferes with Chromium initialization. |
| Native WebKit with a `wfsite` scheme, synthetic document, script and `fetch` | Passes without App Sandbox; the App Sandbox bundle without `network.client` times out without a loaded page; with `network.client` it passes; removing permission makes it time out again. | Outbound permission allows arbitrary TCP connections according to [Apple](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.security.network.client); it is not a page egress boundary. Not every WebKit configuration was explored. |
| Initial Linux ARM64 filter in a container and Ubuntu 24.04 job | The [seccomp filter](M3-BROWSER-LINUX-NETWORK.md) denies direct INET sockets while Unix IPC works; the initial branch's Linux Qt lab passed. | In this initial version, Unix sockets to other local services remained reachable; the job's Qt 6.4 does not test scheme `fetch`. |
| Versioned AppContainer probe in `experiments/m3-windows-appcontainer`, Windows Server 2022 amd64 and Windows 11 ARM64 CI on September 29, 2026 | On both runners the unrestricted parent reaches a loopback listener on a system-assigned port. The child confirms an AppContainer token with no network capabilities, is assigned to a Job Object capped at four processes and 512 MiB, and times out on the same TCP connection. | Shows only that this listener was unreachable in these configurations. It does not start a browser, exercise the limits under load, test every egress path or test Windows 11 x64/Windows 10. The timeout alone does not identify a specific blocking filter. |
| Windows 10/11 | Initial trials do not start a confined browser; the new headless trial is described below. [Qt WebEngine](https://doc.qt.io/qt-6/qtwebengine-platform-notes.html) does not compile with the desktop's MinGW toolchain. | The supervisor's Job Object does not replace a Windows browser and network trial. |

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

For Edge/Chrome, ordinary CI runs protocol tests, the canary and the Chrome positive control through `run-browser.ps1 -ControlOnly -Browser Chrome`. The manual **M3 Windows browser feasibility (known unsupported)** workflow also attempts confined startup: a missing response remains a failed job. Green ordinary CI **does not certify this experimental trial**. A [subsequent ARM64 run](https://github.com/Matte2599/WebFence/actions/runs/36563284174/job/109389186163) detected a 20-second timeout in the Edge control after its version response, while Chrome completed CDP/DOM; the cause is unidentified. Therefore the ordinary control uses Chrome; Edge remains in the optional diagnostic comparison. `-Browser Chrome` or `-Browser Edge` explicitly select an engine in the script; the default `All` tests both when available. To reproduce on Windows, from the repository root:

```powershell
$env:CGO_ENABLED = '0'
go build -o m3-appcontainer.exe ./experiments/m3-windows-appcontainer
if ($LASTEXITCODE -ne 0) { throw 'Lab compilation failed' }
./experiments/m3-windows-appcontainer/run-browser.ps1
```

The command discovers already installed browsers at standard paths and returns an error if no browser is found or a trial fails. It installs no software. The temporary profile grants access only to the created AppContainer and receives a low integrity label; installed browser directories remain unchanged. Logs and CDP frames are bounded. Protocol tests cover oversized frames, closed pipes, CDP errors, missing sessions/targets and failed DOM execution.

**Outcome of the Edge/Chrome trial: confined startup remains unavailable with those browsers.** Engine IPC initialization inside the boundary must be fixed and retested before connecting the HTTPS broker, origin/session behavior and negative tests. General egress, local services, quotas under load and packaging remain unverified. This block neither closes M3 nor constitutes its final comprehensive test.

## Subsequent macOS diagnosis

On September 29, the local crash report and OS events for the 11:02:50 trial (Chrome PID 75546) were inspected. The sandbox denies the lookup of `com.apple.coreservices.launchservicesd`; immediately afterward, `_RegisterApplication` reports that it cannot obtain the ASN identifier and calls `abort`. The report confirms `SIGABRT` in the HIServices / `TransformProcessType` stack. This identifies the recorded reason for that startup: it does not prove that granting service access resolves every obstacle or preserves denial of egress through local services. No entitlement was expanded and no raw system report was committed. The macOS runtime remains open.


## Headless startup inside an outer OS boundary

[ADR-022](ADR-022-HEADLESS-OUTER-SANDBOX.md) describes the experimental fix: pinned official `chrome-headless-shell`, macOS paths consistent with a standalone executable, PID-specific Mach IPC and inherited OS sandbox. App Sandbox/AppContainer replace Chromium's internal sandbox in this trial; the Linux lab retains its own configuration. The desktop remains Go/Qt Widgets and the browser is not yet enabled in the GUI.

Run the macOS probe with:

```sh
python3 experiments/m3-macos-sandbox/headless_probe.py --runtime-archive /path/to/chrome-headless-shell-mac-arm64.zip
```

The archive must be the [official 154.0.8037.57 mac-arm64 build](https://storage.googleapis.com/chrome-for-testing-public/154.0.8037.57/mac-arm64/chrome-headless-shell-mac-arm64.zip): the script verifies the ADR's hash before extraction or execution. It does not download automatically, requires Python 3 and `codesign`, signs only temporary copies and uses only local fixtures. The trial identifier's OS container may remain, without per-run profiles. The old `run.sh` still reproduces the desktop Chrome failure.

The new Windows mode is `m3-appcontainer.exe --headless-appcontainer <absolute-headless-shell-executable-path>`. CI downloads and verifies the pinned win64 archive, grants read/execute only to the temporary directory containing public runtime files, and launches the probe. The profile grants access to the unique AppContainer SID; the Job Object retains 16 processes/1 GiB. The new mode does not change `--browser`, which retains the earlier internal-sandbox trial.

Checks require working CDP and DOM, local HTTP reachable in the control but not contacted by the confined browser, an outside file denied and a profile file readable. macOS checks the OS sandbox of processes reported by the runtime; Windows checks the token, SID, absence of capabilities and Job Object membership of browser/renderer/GPU/network service. The same outside file is used by the control and then the confined trial; the parent confirms its contents still exist. Windows Chromium may report `ERR_FILE_NOT_FOUND` when the path is not visible to the process: the test accepts that outcome too, without attributing a specific ACL error code to it. The Windows DNS rule explicitly excludes `127.0.0.1`: without that exclusion the positive control also failed, so the block would not have demonstrated AppContainer enforcement.

The full trial passes on macOS 26.6.2 ARM64, including three consecutive repetitions and a fresh container identifier. A diagnostic variant with `network.client` fails the network assertion as required. Dedicated macOS 15/26 jobs passed on commit `c48fb26`; Windows CDP/DOM startup passed on both runners on commit `a5265a5`. Extended Windows checks (token/SID/capabilities/Job, HTTP and files), without a debugger, passed on both runners in the dedicated step of [`a5a01d4`](https://github.com/Matte2599/WebFence/actions/runs/36572050007); that run’s separate desktop Chrome control failed after environment reduction. The launcher now preserves only the OS paths needed to identify Chrome’s default directory, in addition to minimal variables; arbitrary variables and credentials are not inherited. The Chrome control remains a CI gate alongside the new headless trials. This evidence does not certify Internet/local-service egress, hostile-content resilience, macOS quotas or runtime distribution; it is not the final M3 test.

## HTTP(S) broker connection

[ADR-023](ADR-023-HEADLESS-PIPE-BROKER.md) adds the anonymous pipe bridge: the parent applies the gate, pinned IPs and TLS while the browser receives CDP replies without target cookies. HTTP/HTTPS fixtures check origin, secure context, script, fetch, outside resource/redirect and in-flight revocation. The macOS probe can select `--broker-executable` with the compiled Go program; Windows headless mode runs the fixtures automatically. Dedicated jobs require this path too. Local trials pass on macOS 26.6.2 ARM64; this remains lab validation and does not complete distribution, quotas, browser sessions or desktop integration.
