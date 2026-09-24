# M1 — Local project store

[Italiano](../it/M1-PROJECT-STORE.md) · [Project and authorization](M1-PROJECT-AUTHORIZATION.md) · [SQLite ADR](ADR-004-STORAGE-SIGNATURE.md) · [Roadmap](../ROADMAP.md)

`internal/storage` implements the **first M1 metadata persistence** with SQLite. It is a Go core component: the desktop does not call it yet and does not save operator projects. It contains no credentials, scan results, evidence, reports or site copies.

## Implemented contract

`storage.Open(ctx, path)` requires an absolute path with an existing parent directory and creates a new file with `0600` permissions on Unix. It rejects non-regular files and, on Unix, existing files accessible to group/others. The caller chooses and protects the directory on a trusted local filesystem; the API does not yet choose the app data directory or solve every symlink race or Windows ACL issue. Public errors are codes without paths or project data.

Schema **v1** contains `projects` and `project_origins`, with a foreign key and cascading delete. The authorization reference is descriptive text, not the original document; it is still potentially sensitive metadata **in plaintext** in SQLite. Origins are canonical and ordered. `CreateProject` inserts a project and its origins in one transaction; an existing ID is rejected. `LoadProject` and `ListProjects` reconstruct immutable projects from a consistent read. An expired declaration remains readable for management and future editing, but `BeginRun` still rejects new runs. `DeleteProject` removes only rows in this schema; it is not secure physical erasure of pages, WAL, backups or future artifacts.

Opening checks the version, required tables, `quick_check` and foreign keys; an unrelated database or newer version is rejected. Connections use WAL, `synchronous=FULL`, enabled foreign keys, `trusted_schema=OFF` and a bounded lock timeout. Version 1 is created atomically from an empty file; there is no v1→v2 migration yet. Synthetic tests cover reopen after commit, listing, cascading delete, expiry after restore, duplicates, rollback halfway through insertion, incompatible schemas, tampered records and context cancellation. CI runs the race-enabled suite on its configured native platforms.

## Limits and next steps

This is **configuration persistence**, not the complete M1 workflow. IT/EN project management UI, authorization renewal and revisions, disk quotas, verified backup/recovery, run data and results, stopping jobs before deletion, evidence files and removal of all app-managed copies remain open. The local filesystem and permissions/ACLs on each real system need dedicated trials. Do not copy an open database without accounting for WAL/SHM; backup strategy is a separate block. No external target was scanned for these tests.
