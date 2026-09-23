# ADR-004 — SQLite and JWS signing

[Italiano](../it/ADR-004-STORAGE-SIGNATURE.md) · [Index](../README.md)

Date: 2026-09-23. Status: selection accepted for M0 foundations; M1/M2 application integration is not implemented. Keychain access remains a separate decision requiring verification.

## Decision and alternatives

| Component | Choice | Rationale and cost |
| --- | --- | --- |
| SQLite | `github.com/mattn/go-sqlite3` **v1.14.52**, embedded SQLite **3.53.4** | `database/sql` API, upstream C engine embedded in the module; Qt already requires a C toolchain. Adds C compilation and requires native checks on each target. |
| JWS | `github.com/lestrrat-go/jwx/v4` **v4.5.0** | Supports the RFC 9864 `Ed25519` identifier. Compatible with Go 1.27.1 without `GOEXPERIMENT`; restricted to JWS with an explicit key. |

`modernc.org/sqlite` v1.59.0 is the CGO-free alternative: useful for a future pure-Go CLI, but it does not remove CGO from the Qt desktop and adds the translated engine runtime. No comparative benchmark was performed; no speed or memory advantage is claimed. We do not use the Qt SQL plugin: core storage must remain toolkit-independent.

`go-jose/v4` v4.1.5 was examined but exposes `EdDSA`, not the identifier required by the profile. jwx v3.3.0 supports the profile; we choose v4 on the already pinned Go version to avoid a later migration. v4.5.0 includes the upstream JSON member-name escaping fix. Direct/transitive dependencies are pinned in `go.mod`/`go.sum`; `option/v3` is version **alpha1**, an API evolution risk to reassess when updating. No automatic runtime updates.

## SQLite experiment

`internal/foundation/sqlite_test.go` contains tests only, without a product store or schema. Each test uses synthetic temporary files and URIs constructed with `net/url`. Configuration: one connection per pool, WAL, `synchronous=FULL`, enabled foreign keys, 50 ms busy timeout and `trusted_schema=OFF` applied by a hook to every new connection.

Tests: commit/rollback, reopen and integrity, Unicode text and SQL handled as a parameter, foreign keys, reinitialization after connection replacement, readers not seeing uncommitted writes, second writer rejected with `SQLITE_BUSY`, recovery after releasing the lock, cancellation of a long query and pool wait. Context expiry does not replace the busy handler's native timeout.

WAL requires a compatible local filesystem; no network folders or synchronization of an active DB. These tests do not establish power-loss recovery, backups, migrations, encryption, quotas or secure deletion. Those flows and actual storage permissions/ACLs belong to M1. Do not copy an open DB while ignoring WAL/SHM files.

## Implemented JWS contract

`internal/signature.Sign` and `Verify` operate on exact bytes. Feasibility profile: compact serialization, embedded payload from 1 byte to 1 MiB, protected header up to 1 KiB containing **only** `alg=Ed25519`, `kid` and `typ=webfence-manifest+jws`. `kid` is 1–64 ASCII alphanumeric, `_` or `-` characters; the internal type is unregistered. Duplicate/unknown headers, invalid UTF-8, different algorithms, `crit`, `b64`, `jku`, `jwk`, JSON serialization and noncanonical base64url are rejected.

The caller supplies a trusted public key and expected ID separately from the document. No key downloads, persistence, payload logging or algorithm fallback. Stable errors without sensitive data. Synthetic private keys are generated during tests, never committed. Tests cover cross-verification with `crypto/ed25519`, unrelated keys, alteration of all three segments, valid signatures over prohibited profiles, limits, concurrency and fuzzing.

This is not yet the report verifier: JCS, manifest schema, file hashes/paths, trust store, revocation, rotation and keychain integration remain pending. Successful verification authenticates bytes against the supplied key; it does not establish identity, observation accuracy or site security. The GUI does not yet use SQLite or JWS.

## Verification and licenses

Race-enabled tests, complete Go tests, vet and module verification passed locally on macOS ARM64. The test prints SQLite 3.53.4. Local 20-second JWS fuzzing completed 2,507,150 executions without failure. `govulncheck` v1.8.0 with `-test ./...` found no known vulnerabilities in the analyzed Go code at check time; it does not examine all native Qt/SQLite libraries or constitute certification. CI for commit `f1d2a84` passed on all four targets, macOS ARM64, Windows Server x86-64 and Ubuntu x86-64/ARM64 ([run 35863431196](https://github.com/Matte2599/WebFence/actions/runs/35863431196)): race suite, build, vet, Qt self-tests, JWS fuzzing on Linux x86-64 and macOS bundle. No implied Windows 10/11 or interactive Debian trial.

Added module licenses are MIT (mattn, jwx, dsig, option, fastjson), checked in pinned sources. SQLite retains its public-domain terms. Preserve texts/notices in future packages containing them; this inventory does not close Qt licensing or distribution legal review.

Primary sources: [driver and DSNs](https://github.com/mattn/go-sqlite3/tree/v1.14.52), [WAL](https://www.sqlite.org/wal.html), [trusted_schema](https://www.sqlite.org/pragma.html#pragma_trusted_schema), [jwx v4.5.0](https://github.com/lestrrat-go/jwx/releases/tag/v4.5.0), [jwx requirements](https://github.com/lestrrat-go/jwx/tree/v4.5.0), [go-jose v4.1.5](https://github.com/go-jose/go-jose/blob/v4.1.5/shared.go), [RFC 9864](https://www.rfc-editor.org/rfc/rfc9864.html). Versions were resolved with Go and checked against sources; search indexes may show earlier releases.
