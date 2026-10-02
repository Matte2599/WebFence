# ADR-023 — HTTP(S) pipe broker for headless labs

[Italiano](../it/ADR-023-HEADLESS-PIPE-BROKER.md) · [Confined startup](ADR-022-HEADLESS-OUTER-SANDBOX.md) · [M3 status](M3-VALIDATION.md)

Date: 2026-10-02. Status: **implemented in the core and labs; desktop integration open**.

## Problem and decision

The ADR-022 headless browser starts inside App Sandbox/AppContainer without independent networking, but the startup trial does not load HTTP(S) resources through WebFence. `internal/browser.RequestBridge` is added: the trusted controller receives the intercepted URL, method and request type, applies the gate and uses the pinned broker. It opens no listener and accepts no browser headers or bodies. `GET`/`HEAD`, scope, path, budget, time, revocation, DNS and TLS verification remain in the existing components.

The browser receives a bounded response through [CDP Fetch](https://chromedevtools.github.io/devtools-protocol/tot/Fetch/), with only the rendering headers already allowed by the proxy. Target cookies, browser credentials and proxy authorization are not transferred. Responses are limited to a 16 KiB body and 8 KiB of headers; observations are ephemeral, at most 256, without queries/headers/bodies. Closing the bridge or gate cancels in-flight requests. Policy denial yields 403, broker failure 502: neither becomes a positive security outcome.

The bridge does not yet represent admitted redirects: if the broker follows a redirect and changes the final URL, the bridge returns 502, preventing the final body from being displayed under the original URL. The broker stops out-of-scope redirects before external contact. Full navigation through admitted redirects remains to be implemented.

## Labs and verification

`experiments/internal/headlessfixture` creates only loopback servers on OS-assigned ports. For HTTP and HTTPS it uses `site.test`, `/app` policy, 16 gate/broker attempts and an ephemeral TLS certificate trusted only by the fixture broker. It accepts no external targets. The document loads a script, an off-origin resource and a `fetch` with a synthetic Authorization header; a denied redirect, slow request and post-revocation request follow.

The page must report the exact origin, `isSecureContext=true` only for HTTPS, empty cookies, the API body and statuses 502/502/403. The controller requires five target requests, zero outside contacts, three admitted observations and actual cancellation of the slow request. The test rejects unexpected host or credentials at the target.

On macOS the Python controller uses a separate Go broker over stdin/stdout; the broker process is outside the browser sandbox. On Windows the same fixture code runs in the Go parent. CDP remains on inherited pipes; frames are limited to 64 KiB, nesting to eight commands, with bounded events and request counts. Out-of-order replies are associated with the waiting ID and session. No intercepted request is continued through Chromium's network stack.

Startup, direct networking, outside/private file and descendant OS sandbox checks remain required, including after mediated navigation. CI runs the new path on macOS 15/26 ARM64 and Windows Server 2022/Windows 11 ARM64; the Windows engine on the ARM64 runner remains emulated x64. The existing Linux lab retains its boundary and Chromium's internal sandbox.

Local macOS 26.6.2 ARM64 trial: mediated HTTP(S), origin/context, revocation, OS canaries and protocol tests passed. The Go suite, browser/fixture race detector, vet, IT/EN Qt self-test and 92 Python tests (4 skipped) passed. Branch and merge CI must be checked for the current commit; this is not the final M3 validation.

## Reproduction and limits

On macOS, from the repository root, with the ADR-022 pinned archive already available:

```sh
CGO_ENABLED=0 go build -o /tmp/webfence-m3-headless-broker ./experiments/m3-headless-broker
python3 experiments/m3-macos-sandbox/headless_probe.py --runtime-archive /path/chrome-headless-shell-mac-arm64.zip --broker-executable /tmp/webfence-m3-headless-broker
```

On Windows the ADR-022 build and `--headless-appcontainer` command also run both mediated fixtures. The auxiliary Go command accepts only `http` or `https` and synthetic pipe messages; it is not a production proxy.

Stable packaging/signing, aggregate macOS and Linux quotas outside the container, local services and hostile content, authenticated browser sessions, admitted redirects, other CDP target handling and Qt integration remain open. The ADR-022 outer OS model and experimental flag are unchanged. This block closes no M3 exit criterion.
