# ADR-005 — Native credential storage

[Italiano](../it/ADR-005-CREDENTIALS.md) · [Index](../README.md)

Date: 2026-09-23. Status: M0 implementation selected, cross-platform runtime verification pending. This is a core foundation, not a credential-management UI or report-key lifecycle.

## Decision

`internal/credentials` provides `New`, `Get`, `Set` and `Delete`. Only the fixed WebFence namespace is exposed. IDs contain 1–64 lowercase ASCII letters, digits, `_` or `-`; secrets contain 1–2048 arbitrary bytes. Lowercase avoids Windows' case-insensitive target-name aliases. There is no listing, bulk deletion, plaintext file fallback, environment-secret fallback, logging or automatic unlocking. Invalid input and canceled contexts are rejected before native access. Native diagnostics are reduced to stable errors without IDs or secrets; missing, locked/interaction-required, ambiguous and unavailable remain distinct when the backend permits it.

| System | Adapter | Limits |
| --- | --- | --- |
| macOS | Small CGO adapter to Security.framework's classic keychain API; default keychain, exact service/account | Supports development binaries without Data Protection entitlements. These Apple APIs are deprecated: retain this as an M0 feasibility choice and reassess signed distribution. |
| Windows | `github.com/danieljoos/wincred` v1.2.3, generic Credential Manager item | Current user, `PersistLocalMachine`; no enterprise roaming. This is not a per-application security boundary against other code running as that user. |
| Linux | `github.com/godbus/dbus/v5` v5.2.2 and a restricted Secret Service adapter | Existing session service and unlocked default collection required. No service activation, collection creation, unlock or prompt execution. |

The adapters have no GUI dependency. SQLite, report signing and provider credentials are not connected to this store yet. `Set` replaces an exact item; concurrent callers have last-writer semantics, not compare-and-swap. A failed/canceled native write can have an uncertain outcome: callers must reconcile by reading, not assume rollback.

## Native behavior and boundaries

macOS classic keychain uses a process-wide interaction switch. Every operation in this package is serialized, disables UI and restores the previous value before returning. A locked keychain fails before item lookup, and required authentication fails without opening a dialog. Do not introduce independent in-process Security.framework callers while this adapter exists: the lock cannot coordinate external libraries. Cancellation works before entry and while waiting for that lock; synchronous Security.framework calls cannot be interrupted once entered. No detached goroutine continues a mutation after returning. Do not call from the Qt event thread. The future signed app must re-evaluate Data Protection/LAContext and credential migration. The modern “no UI” query attribute alone does not guarantee suppression for legacy keychain items, as the SDK documents.

Windows calls are also synchronous and cannot be interrupted after entry. OS access-denied maps to locked/interaction-required; unavailable logon sessions remain unavailable. Locking the screen is not a tested or promised denial boundary. Generic credential blobs are not hardware-backed keys or non-exportable signing identities.

Linux accepts only one Unix-domain D-Bus endpoint (`path` or `abstract`, optional GUID). TCP, autolaunch, multiple endpoints, ambiguous keys and NULs are rejected. A private connection uses EXTERNAL authentication, pins the current Secret Service unique bus owner, and applies a three-second operation context including dialing, authentication and calls. It closes its own connection; it never closes a shared application bus. A missing/locked default collection fails closed; multiple matching items fail as ambiguous. Secret transfer uses the specification's `plain` algorithm over the local bus; it adds no transfer encryption. At-rest protection belongs to the OS service. The local session bus and service are trusted; this does not defend against a compromised account, bus daemon or administrator.

Secret buffers owned by the wrapper are cleared after writes and rejected reads; successful reads belong to the caller. Go, native libraries and the OS can retain copies. No secure-erasure guarantee is made. The store has no recovery, rotation, backup, ACL editor or user-facing error translations yet; those belong to integration.

## Alternatives examined

`zalando/go-keyring` v0.2.8 uses the `security` command on macOS and Linux prompt waits without caller context. Its macOS secret is passed through stdin, not argv; this decision is about command identity and lifecycle control. `99designs/keyring` v1.2.2 includes more backends than required and its macOS read checks empty results before the error, potentially hiding an unavailable/locked read as missing. `keybase/go-keychain` v0.0.1 exposes SecItem wrappers, but does not supply the required per-operation legacy interaction control. No benchmark or general security ranking of these projects is claimed. The selected Windows and D-Bus modules are MIT; transitive `x/sys` is BSD-3-Clause, pinned in the module graph. Native Apple/Windows services retain their platform terms.

## Verification

Local macOS ARM64: race-enabled unit and real native round-trip tests passed; binary values, maximum size, update, reopen, exact-ID isolation, delete and missing states. A separately created temporary keychain is locked to verify Get/Set/Delete denial and then deleted; personal keychains are not locked. An absent explicit keychain fails closed. Unit tests cover validation without I/O, cancellation while waiting, zero-value stores, error redaction and buffer handling. Full Go tests, vet and module verification passed. Linux/Windows test binaries cross-compiled locally; this is not native execution evidence. Four-platform CI integration is added and awaits its actual run.

Linux tests include rejected bus addresses, absent sockets and cancellation during an unresponsive authentication handshake. `scripts/test-keychain-linux.sh` starts a private D-Bus session and GNOME Keyring with disposable XDG directories and a synthetic password. It verifies absent service before startup, then real read/write/delete and locked-collection failures. It never runs the locked-collection test on a personal desktop bus. Windows tests use a random service prefix and delete only their synthetic item; they do not lock the user's session. Native Windows lock/unavailable-session behavior still needs a controlled interactive trial, not an invented equivalence to Linux/macOS.

## Sources

[Apple classic interaction control](https://developer.apple.com/documentation/security/seckeychainsetuserinteractionallowed(_:)), local macOS SDK Security headers; [Windows credential types and persistence](https://learn.microsoft.com/en-us/windows/win32/api/wincred/ns-wincred-credentialw); [wincred v1.2.3](https://github.com/danieljoos/wincred/tree/v1.2.3); [godbus v5.2.2](https://github.com/godbus/dbus/tree/v5.2.2); [Secret Service specification](https://specifications.freedesktop.org/secret-service/latest/), [plain transfer](https://specifications.freedesktop.org/secret-service/latest/ch07s02.html).

First CI `aa722e3`: native macOS tests passed; both Linux runners exposed an incorrect rejection of binary reads. GNOME Keyring always returns `text/plain`, including after an `application/octet-stream` write ([upstream source](https://github.com/GNOME/gnome-keyring/blob/main/daemon/dbus/gkd-secret-secret.c)). The adapter now preserves opaque bytes without interpreting MIME; session, parameters and size remain checked. The launcher waits for service ownership without activating a second daemon. Fix awaits the next CI run.
