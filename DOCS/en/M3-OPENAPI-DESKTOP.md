# M3 — OpenAPI selection in the desktop

[Italiano](../it/M3-OPENAPI-DESKTOP.md) · [Core import](M3-OPENAPI-IMPORT.md) · [Roadmap](../ROADMAP.md)

The Qt Widgets/MIQT scan window offers **Import OpenAPI JSON** for the selected project. It reads a regular local file of at most 1 MiB, opens a brief managed run to check current authorization and uses the one saved origin and current GET path prefixes. The document does not choose a server: `servers` and `$ref` neither start networking nor expand scope. Admitted static GET routes appear in a non-editable choice; only the chosen route fills the seed field. Canceling leaves the previous seed intact. The import run closes before selection; a later scan opens a fresh run and repeats broker checks.

The workflow remains **one seed per scan**. It does not execute HEAD, POST, templates, OpenAPI authentication, multiple API operations or visits during import. File, format, authorization or policy errors show a message without the file path or contents. Imported routes are not retained in the database or report. The GUI still offers no M3 browser or accounts.

**Local verification on September 27, 2026:** IT/EN Qt offscreen self-test with a synthetic project and OpenAPI file, inert external `servers`, explicit choice of `/app/api`, zero hits during import, rejection of ambiguous JSON without changing the seed and a subsequent controlled scan. Go tests, documentation links and branch/merged CI must be assessed per commit. This block makes the offline inventory usable but does not close M3 API crawling.
