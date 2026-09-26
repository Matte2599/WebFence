# M2 — CVE/NVD cache

[Italiano](../it/M2-INTELLIGENCE-CACHE.md) · [Roadmap](../ROADMAP.md) · [Intelligence specification](VULNERABILITY-INTELLIGENCE.md)

Status: **first M2 core block**. `internal/intelligence` has separate adapters for an explicit NVD CVE API 2.0 date window and explicitly selected CVE Program IDs. No feed is downloaded on startup, no target inventory is sent to the sources, and no CVE database is bundled. The [M2 desktop](M2-VALIDATION.md) exposes manual updates.

## Implemented contract

- The SQLite cache is separate from the project database. It keeps source JSON records, state, source dates, content SHA-256 and UTC acquisition time. A hash alone does not authenticate the source; normal connections use HTTPS to official endpoints. `cve` and `nvd` sources are never silently merged.
- NVD synchronization requires caller-selected UTC `start`/`end`, at most 120 days apart. It fetches pages of 500 with limits of 128 pages, 16 MiB per response, 1 MiB per record and 512 MiB for the main file. The watermark and all records in a window commit together; errors, cancellation and invalid pages preserve the last valid cache. A later window can overlap; `NextWindow` suggests one hour of overlap. A partial cache is not the full CVE catalog.
- The CVE adapter fetches up to 32 explicit IDs per operation in one transaction. A failed update cannot replace previous records. Withdrawn or rejected source states are retained when returned.
- `Snapshot.Status` derives `fresh`, `stale`, `offline` or `unavailable` from a caller-selected age threshold; `offline` describes intentional network-free use, not freshness. `WindowStart` and `Watermark` declare synchronized time coverage, not catalog completeness.
- HTTP calls do not follow redirects; they honor `Retry-After` up to 30 seconds and make no more than three attempts for transient responses. Time, response size, page and record limits are explicit. Custom endpoints are accepted only on loopback for synthetic tests.

## Verification and limits

`go test -race ./internal/intelligence` covers pagination, provenance, offline reopen, source separation, rollback after an invalid/unavailable feed and rejection of custom external endpoints. Tests contact no real NVD or CVE service. They do not establish historical CVE coverage, authenticity of offline imports or compatibility with every upstream record. The desktop always requires an explicit action before opening source network connections.

Primary sources: [NVD API 2.0](https://nvd.nist.gov/developers/vulnerabilities), [CVE Services](https://github.com/CVEProject/cve-services), [official CVE catalog](https://github.com/CVEProject/cvelistV5). Source parameters and limits need review when upstream changes.
