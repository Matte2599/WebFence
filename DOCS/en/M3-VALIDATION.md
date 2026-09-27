# M3 — Technical status and overall validation

[Italiano](../it/M3-VALIDATION.md) · [Roadmap](../ROADMAP.md) · [Development](DEVELOPMENT.md)

**M3 is not closed.** The blocks below are narrow trials on synthetic fixtures; no external target scan was run. The four roadmap items remain open until their respective exit criteria are met. Green CI shows that code, self-tests and packaging specified by the jobs pass for the cited commit; it does not prove browser isolation or sufficient real-world coverage.

| M3 item | Implemented and verifiable | Missing for closure |
| --- | --- | --- |
| Isolated browser | [HTTP gate and proxy](M3-BROWSER-PROXY.md), [Qt helper and lab](M3-BROWSER-LAB.md) with document, subresource, redirect, `fetch` and one DOM click on loopback fixtures; [Unix helper limits](M3-BROWSER-RESOURCE-LIMITS.md) for files/descriptors/core dumps and per-process Linux virtual address space. | Independent browser egress containment, macOS memory and aggregate Linux/macOS memory/process limits, Windows runtime containment and trial, mediated HTTPS, safe desktop integration and negative tests on every platform. |
| Test accounts | [Verified sessions](M3-SESSIONS.md), [managed two-account run](M3-CROSS-ROLE-RUN.md) and [Qt dialog](M3-AUTH-DESKTOP.md) limited to loopback, with confirmations renewed for every run and ephemeral secrets. | CSRF/MFA/OIDC or bearer login, cookie rotation, authenticated public grants, authenticated JavaScript browsing and persistent identity inventory. |
| OpenAPI, dynamic and context | [Offline OpenAPI import](M3-OPENAPI-IMPORT.md), [selected static GET visits in the desktop](M3-API-DESKTOP.md), [core replay of observed `fetch` paths](M3-OBSERVED-CRAWL.md) and a first [cross-role check](M3-CROSS-ROLE.md). | Product DOM interaction crawler, dynamic parameters/queries under policy, API-specific checks, more resources/roles and measurements on a representative SPA/API corpus. |
| Rule families | `HTTP-XCTO-001` observes an HTML header; `AUTH-CROSSROLE-001` requires exact positive evidence and confirmation that access is forbidden. Six synthetic cross-role cases yield one finding and five inconclusive results. | Further families with prerequisites, positive/negative fixtures and published measures; real-corpus sensitivity, specificity and false positives are unknown. |

Credentials, URLs, queries, headers and bodies are absent from the redacted results of these blocks; GUI passwords are not saved, but Go, Qt or OS copies may remain in memory. No finding, or a different response, does not prove that access control is correct. GET can have effects: local fixtures do not authorize external trials.

## Overall test

**Integrated local verification of commit `b82693a`, September 27, 2026:** Apple Silicon, macOS 26.6.2, Go 1.27.1, Qt 6.11.2. Every command below exited successfully; only synthetic fixtures and loopback were used.

| Area | Verification | Outcome |
| --- | --- | --- |
| Dependencies | `go mod verify` | All modules verified. |
| Core and regressions | `go test -race` on 17 core packages, then `go test ./...` and `go vet ./...` | Passed, including scanner, browser, sessions, checks, desktop, M1 and M2. |
| Builds | `go build` for `webfence`, `webfence-report`, `webfence-verify` | Passed. |
| Native desktop | `QT_QPA_PLATFORM=offscreen webfence --self-test` | Passed in IT/EN: M0, M1, M2, M3 OpenAPI and the synthetic two-account cross-role check. |
| GUI stability | `webfence --soak-test=10s` offscreen | Passed, 4 complete cycles in 12 seconds. |
| Experimental browser | `go test -tags=m3browserlab ./experiments/m3-browser`, `go vet` with the same tag and three offscreen `go run -tags=m3browserlab ./experiments/m3-browser` invocations | Passed: document, DOM click, external script, two `fetch` requests, proxy and out-of-scope blocks. |
| Scripts and documents | `python3 -m unittest discover -s scripts/tests`, `python3 scripts/package-project-docs.py --check .`, `git diff --check` | 79 Python tests, 4 skipped, no errors; 166 Markdown files and 1,536 valid local links on the validation branch; clean diff. |

CI adds builds, self-tests and development packaging on macOS, Windows and Ubuntu ARM64/x86-64; the WebEngine lab runs on macOS and Linux, not Windows. Branch CI and the corresponding squash-merge CI on `main` are distinct for the final blocks:

| Block | Branch | Merge on `main` |
| --- | --- | --- |
| Managed cross-role run | [`9d898ee`](https://github.com/Matte2599/WebFence/actions/runs/36287236763) green | [`b297358`](https://github.com/Matte2599/WebFence/actions/runs/36287857087) green |
| Two-account Qt dialog | [`64fa46b`](https://github.com/Matte2599/WebFence/actions/runs/36288145773) green | [`1587732`](https://github.com/Matte2599/WebFence/actions/runs/36288784914) green |
| Replay of observed `fetch` paths | [`8182adc`](https://github.com/Matte2599/WebFence/actions/runs/36287637738) green | [`96670a5`](https://github.com/Matte2599/WebFence/actions/runs/36289377317) green |
| DOM click in the lab | [`dfab892`](https://github.com/Matte2599/WebFence/actions/runs/36287803019) green | [`fd6be19`](https://github.com/Matte2599/WebFence/actions/runs/36289947405) failed: second page reached, second `fetch` absent within 12 seconds. |
| Second-page script stability | [`5c3631f`](https://github.com/Matte2599/WebFence/actions/runs/36290666798) green | [`b82693a`](https://github.com/Matte2599/WebFence/actions/runs/36291202123) green. |

This verification covers software and fixture regressions. It does not replace assistive/hardware trials, an authorized staging target, a real-world accuracy measurement or OS-level network containment. No M3 item is marked complete based only on lab tests.
