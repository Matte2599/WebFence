# M3 — Static API visits in the desktop

[Italiano](../it/M3-API-DESKTOP.md) · [API service](M3-API-BATCH.md) · [Offline import](M3-OPENAPI-DESKTOP.md) · [Roadmap](../ROADMAP.md)

The Qt Widgets/MIQT scan window offers **Visit selected OpenAPI routes** for an authorized project. It reads a local JSON document of at most 1 MiB, builds the static GET candidates admitted by current scope and path prefixes offline, then shows a multiple-selection list. The operator chooses 1 to 32 paths and confirms the dialog before requests. No route is preselected. Confirmation uses the loopback or manually pinned public IP mode, budget and pacing already entered in the form; HTML links are not followed even when the general option is enabled.

The core service reimports the document and revalidates the choices in a fresh managed run before traffic. The button is disabled during a scan. Canceling, an invalid file or an empty selection sends no requests; the file and paths are not saved in the project or report. Results show only redacted codes and counts in the M1 ledger. An insufficient budget or a redirect yields a partial run; the redirect destination is not visited. This function does not send POST, use OpenAPI authentication schemes or test JSON responses for API-specific vulnerabilities.

**Local verification on September 27, 2026:** IT/EN Qt offscreen self-test with inert external `servers` and two explicitly selected JSON routes on an `httptest` server; only two GET requests, a persisted run and URL/body redaction. Go regressions and desktop/core race detector passed locally; branch and merged CI must be checked per commit. Dynamic browsing, desktop sessions and API rules remain open.
