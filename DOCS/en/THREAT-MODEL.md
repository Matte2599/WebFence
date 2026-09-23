# WebFence threat model

[Italiano](../it/THREAT-MODEL.md) · [Index](../README.md)

Status: product security requirements to verify before releases. Targets, pages, feeds, imported projects and AI responses are untrusted. An authorized target may still be malicious toward the scanner.

## Assets and boundaries

Protect credentials, projects and client data, signing keys, report integrity, the operator's machine and target availability. Main boundaries: GUI/core, core/target, core/browser, core/LLM, import/storage, updater/executables and export/recipient. The first product has no tenant isolation: a shared local account is not organizational separation.

| Threat | Planned control | Acceptance evidence |
| --- | --- | --- |
| SSRF and DNS rebinding | Central policy, verified connected IP, redirect and IPv6 controls | Private destination after redirect or DNS change rejected |
| Browser escaping scope | Isolated profile, broker/egress, subresource and WebSocket controls | Alternative-channel fixture cannot contact excluded hosts |
| Overload or destructive action | Budgets, exclusions, decompression limits, global stop, no automatic forms | Slow responses and hostile compression respect limits |
| Credential theft | Keychain, origin scope, no cross-origin forwarding | Foreign-host redirect receives no token or cookie |
| Cross-project contamination | Project-scoped access, separate AI caches and contexts | Same URL in two projects shares no evidence or credentials |
| Prompt injection | Model has no authority; code limits schemas and tools | Target text cannot expand scope or access files |
| Report XSS / injection | Escaped text, rendering without scripts or remote fetches | Hostile evidence markup stays inert in UI and HTML |
| Malicious import | Size/schema limits, no traversal/symlinks/archive bombs | Import cannot write outside the authorized directory |
| Report tampering | Standard manifest/signature, externally trusted key | File alteration or key substitution is not accepted as trusted |
| Supply chain | Pinned dependencies/releases, SBOM, signed updater, rights review | Altered update or unexpected version rejected |
| Provider exfiltration | Explicit remote mode and minimization | Disabled AI produces no provider traffic |
| Key abuse | Restricted signing access and separate keys | Browser/LLM cannot read the private key |

## Identity and credentials

The GUI runs as the user; ordinary HTTP scans do not require administrator privileges. Target authentication uses test accounts. Logins and sessions are separated by identity and project; an expired session interrupts dependent checks. Do not log passwords, bearer tokens, cookies or complete personal payloads.

A future control API requires authentication, resource authorization, origin/CSRF checks where applicable and protection from DNS rebinding toward loopback. A local port is not automatically safe.

## Limits of this model

A compromised operator machine may read process memory and alter the program. Signing and encryption do not solve that problem. Browser isolation must be tested on each operating system; mechanisms differ. The application cannot guarantee that an arbitrary target assigns no effects to apparently harmless requests.

Before adding browsers, AI, plugins, an updater or shared users, update this matrix and negative tests. Product vulnerabilities follow [SECURITY.md](../../SECURITY.md); target vulnerabilities remain confidential to authorized parties.
