# WebFence

**Web security analysis, verifiable evidence and fix validation.**

[Italiano](README.md) · [Documentation](DOCS/README.md) · [Roadmap](DOCS/ROADMAP.md) · [License](LICENSE)

**Status: M0 in progress — first runnable desktop prototype, using synthetic examples only.** The Go/Qt Widgets GUI loads 10,000 rows, filters them and displays evidence in IT/EN. The scanning engine is not implemented; there are no supported releases or security benchmarks. See the [M0 report](DOCS/en/QT-DESKTOP.md).

**The author selected Qt** after the [practical comparison](DOCS/en/GUI-COMPARISON.md). The main command uses Qt Widgets through MIQT; [accepted ADR](DOCS/en/ADR-002-GUI.md). Full accessibility and distribution remain open gates.

The core includes a first [allowed-origin check](DOCS/en/M0-SCOPE.md), with an HTTP lab confined to loopback tests. A [lab transport](DOCS/en/ADR-003-TRANSPORT.md) now adds DNS/IP checks, TLS, budgets and cancellation, restricted to loopback. The GUI remains offline; no production scanner.

The foundations also include SQLite experiments and a restricted Ed25519 JWS component: [M0 choices and checks](DOCS/en/ADR-004-STORAGE-SIGNATURE.md). Project persistence and signed reports in the GUI are not yet available. A separate [native credential adapter](DOCS/en/ADR-005-CREDENTIALS.md) is now verified natively on all four CI targets, including unavailable states ([run](https://github.com/Matte2599/WebFence/actions/runs/35870058803)).

The macOS bundle includes a [temporary Qt Cocoa correction](DOCS/en/ADR-006-QT-COCOA.md) for the reproduced assistive crash; compact layout checked at 200% and [CI passed on all four targets](https://github.com/Matte2599/WebFence/actions/runs/35873709410). Actual screen-reader and distribution trials remain open.

The macOS bundle now collects [provenance and available notices](DOCS/en/M0-PACKAGING.md), explicitly recording materials still missing for distribution.

Procedures for [Debian and Windows packaging](DOCS/en/ADR-007-PACKAGING.md) are also available. Verified amd64/arm64 `.deb` files in separate Debian runtimes and the Windows ZIP with system-only PATH; [CI `32655b0`, six passing jobs](https://github.com/Matte2599/WebFence/actions/runs/35877356234). Development packages: real-desktop trials and full distribution-material review remain open.

The [prolonged stability procedure](DOCS/en/M0-STABILITY.md) repeats GUI flows and records Go memory/RSS. Updated Cocoa session completed: over 30 minutes without crashes or AX warnings and without observed sustained RSS growth. Uneven cadence and limitations documented; actual reader trials remain open.


## Why WebFence

AI tools allow individual developers to build substantial applications. Faster development calls for security checks that are equally accessible and rigorous. This is the project's motivation, not evidence that AI-generated code is invariably less secure.

WebFence is intended to assess authorized websites and APIs, connect each finding to collected evidence and validate fixes over time. It will combine traditional checks, known-vulnerability intelligence and optional AI analysis. The professional coverage of platforms such as Invicti/Acunetix is a long-term reference, not an established capability or an affiliation.

## Planned capabilities

| Area | Goal |
| --- | --- |
| Scanning | Discover websites, APIs and, at a later stage, JavaScript applications and authenticated workflows. |
| Profiles | Independently configure request budgets, concurrency, depth, duration and intrusiveness. |
| Traditional engine | Versioned rules, reproducible checks, evidence and explicit coverage reporting. |
| AI engine | Remote frontier-model providers, local runtimes and optional model packages, subject to resources and licensing. |
| Vulnerability intelligence | Updated CVE cache and enrichment; distinguish version matches from verified vulnerabilities. |
| Reports | IT/EN results with evidence, priority, limitations, remediation and verifiable cryptographic signatures. |
| Project memory | Retain inventory and history until deletion; handle credentials and raw evidence under separate retention policies. |
| Fix validation | Targeted retests, baseline comparisons and regression discovery within the reassessed scope. |

All these capabilities are **planned**. Milestones and acceptance criteria are in the [roadmap](DOCS/ROADMAP.md).

## Technical direction

**Go is the recommended language for the core and native desktop application**, with Qt Widgets/MIQT selected for the GUI. The first deployment will target a single operator. SQLite is the proposed local store. The scanning browser and AI inference will run as separate components when needed; a CLI can reuse the core later.

Rust remains an option for bounded components if measurements justify it. The comparison and reconsideration criteria are in the [language ADR](DOCS/en/ADR-001-LANGUAGE.md). Desktop delivery is confirmed; the toolkit still needs accessibility, performance and packaging validation.

## Product principles

- Scan only owned or explicitly authorized systems, with engine-enforced scope and limits.
- AI is optional: traditional analysis must work without an LLM; no external-provider uploads by default.
- Keep evidence, AI hypotheses and CVE matches distinguishable.
- Failed, blocked or incomplete checks must never become a “secure” result.
- A signature establishes integrity and provenance relative to a trusted key; it does not certify the absence of vulnerabilities.

## Getting started today

1. Read the [product vision and requirements](DOCS/en/PRODUCT.md).
2. Review the [architecture](DOCS/en/ARCHITECTURE.md), [threat model](DOCS/en/THREAT-MODEL.md) and [roadmap](DOCS/ROADMAP.md).
3. To contribute, start with [CONTRIBUTING.md](CONTRIBUTING.md). The [development guide](DOCS/en/DEVELOPMENT.md) contains prototype commands.
4. Persistent decisions live in [MEMORY.md](MEMORY.md); agent instructions are in [AGENTS.md](AGENTS.md).

## Run the prototype

With Go 1.27.1 and the native dependencies in the [development guide](DOCS/en/DEVELOPMENT.md):

```sh
CGO_CXXFLAGS='-O2 -g -std=c++17' go run ./cmd/webfence
```

On Apple Silicon macOS, create a local bundle with `sh scripts/package-macos.sh`. This is a GUI laboratory: `.invalid` URLs are inert text. The development build is not a signed/notarized installer.

Requested platforms: Apple Silicon macOS, Windows 10/11 x86-64, Debian and derivatives on x86-64/ARM64. Actual support depends on the [M0 matrix](DOCS/en/QT-DESKTOP.md). The [UX direction](DOCS/en/UX.md) calls for a restrained, traditional desktop, guided workflow and progressively available advanced tools.

Dependency materials: [source and notice collector](DOCS/en/M0-SOURCE-MATERIALS.md) verified against 15 upstream archives; optional attachment of 247 notices before macOS bundle signing. Distribution review remains open.

## License and author

Author and project owner: **Matteo Luigi Feroldi** — [Matte2599](https://github.com/Matte2599).

The project uses the **WebFence Community License 1.0**, a custom source-available license. Study, modification, forks, personal noncommercial use and use by independent professionals are free, including paid assessments and reports delivered to clients. Free sharing is subject to [LICENSE](LICENSE).

Use by companies, distribution of the program to companies, software resale and offering access to the program as a service require prior written authorization from Matteo Luigi Feroldi, which may involve an agreed fee or royalties. Delivering a professional report to a company does not distribute the program to it.

These restrictions are incompatible with the [Open Source Definition](https://opensource.org/osd). WebFence is therefore not described as OSI open source software. The [licensing guide](DOCS/en/LICENSING.md) explains the decision and alternatives. The custom text requires legal review before being used as the basis of commercial contracts.

For license requests, open an issue titled **Licensing inquiry**, without confidential information; the author can provide a private channel. For vulnerabilities in WebFence, follow [SECURITY.md](SECURITY.md).
