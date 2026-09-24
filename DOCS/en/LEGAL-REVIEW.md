# M0 — Legal review dossier

[Italiano](../it/LEGAL-REVIEW.md) · [Licensing](LICENSING.md) · [M0 matrix](M0-VALIDATION.md)

Status: technical preparation, **no professional legal opinion obtained**. This dossier does not change LICENSE or activate a CLA or commercial agreement. The author confirmed that professional review still needs to be arranged.

## Material for review

- [WebFence Community License 1.0](../../LICENSE), governing English text and Italian translation.
- [CONTRIBUTING](../../CONTRIBUTING.md): contributor copyright and absence of automatic commercial rights.
- [ADR-002 Qt](ADR-002-GUI.md) and [ADR-004 SQLite/JWS](ADR-004-STORAGE-SIGNATURE.md).
- [Inventory and packaging](M0-PACKAGING.md), [macOS native source package](M0-SOURCE-MATERIALS.md#macos-source-package-accompanying-the-bundle) and [Windows sources](M0-WINDOWS-SOURCES.md): verified materials available with explicit limitations and provenance.
- [Cocoa rebuilding and replacement](M0-QT-REPLACEMENT.md), tested locally and in CI; covers the plugin only, not every library.
- [Review of 15 macOS recipes](../evidence/homebrew-recipe-review-2026-09-23.md), distinguishing build patches/resources from test fixtures and Linux branches.
- [Mapping of nine macOS Qt binaries and upstream attributions](../evidence/qt-component-review-2026-09-23.md): verified technical associations, not a complete embedded inventory or legal approval.
- [Ledger of metadata in 21 Windows recipes](../evidence/windows-license-metadata-2026-09-24.md): rechecked MSYS2 declarations, not final binary licenses; `custom`, `OR` choices, embedded components and notices require review.
- [Offline winpthreads Git verification](../evidence/windows-vcs-2026-09-24.md): recipe checksum reproduced from a locked pack without checkout or recipe execution.
- [Eight detached Windows source signatures](../evidence/windows-signatures-2026-09-24.md): offline OpenPGP verification with seven fingerprint-pinned public keys; signer identity, key trust, binary signatures and distribution obligations remain open.
- [Correspondence of Windows notices](../evidence/windows-notice-linkage-2026-09-24.md): 74 files in the first CI; [selection expanded to 136 sidecars](../evidence/windows-attribution-sidecars-2026-09-24.md) with checks of 27 Qt references. Presence of texts does not establish notice completeness or legal duties.
- [Ledger of 18 non-Qt libraries and Go executable](../evidence/macos-nonqt-component-review-2026-09-23.md): technical provenance and upstream declarations; individual binary terms still require review.
- [libintl follow-up](../evidence/macos-nonqt-component-review-2026-09-23.md): recipe and source indicate LGPL 2.1 or later for the component, but exact binary composition and distribution obligations still require review.
- [Reporting](REPORTING.md): operator signatures, no security certification, template rights separate from user data.

Author requirements to preserve: free individual/independent professional use including paid engagements; companies require authorization, including internal use; commercial/company distribution requires agreement; delivery of reports to clients allowed; optional negotiated royalties, never automatic. The product is source-available, not OSI open source.

## Questions for the legal opinion

| Area | Required review | Current outcome |
| --- | --- | --- |
| Definitions | Individual professional, consulting company, employee, sole proprietorship; clients controlling the program | Not reviewed |
| Grants | Forks/public hosting, professional use, redistribution, hosted services, report exception | Not reviewed |
| Contract validity | Acceptance, governing law/venue, consumers and mandatory law, governing language, termination/reinstatement | Not reviewed; no invented jurisdiction or tax identity |
| Qt and third parties | Separate library rights; notices, corresponding source, replacement/relinking and execution of modified versions | Inventories and material packages available; Cocoa replacement tested. Embedded mapping, compatibility and completeness require review |
| Contributions | Ownership, employer authorization, commercial licensing grant, patents and separate acceptance | No active CLA |
| Commercial agreements | Scope/versions, installations, duration, support, pricing/royalties, liability and applicable data processing | No active standard offer or agreement |

For dynamically linked Qt, merely including `.dll` files/frameworks is insufficient: packages must respect library-license rights, including modification and use of the result. Check that WebFence terms and installers do not restrict them. Do not automatically assign LGPL terms to GPL-only Qt modules; inspect actual bundled components. [Official Qt guidance](https://www.qt.io/development/open-source-lgpl-obligations).

## Specification for a contributor agreement

The agreement must identify parties and contributions (commit/PR), preserve contributor copyright and expressly define a nonexclusive grant allowing Matteo Luigi Feroldi to integrate, modify, distribute and sublicense those contributions in community and commercial WebFence releases. Duration, irrevocability, transferability, patent treatment, ownership representations, liability and possible compensation require reviewed wording; this specification does not grant them.

Provide variants for natural persons and entities holding corporate rights, proof of signing authority, an inventory of third-party material and separate acceptance recording the text version/hash. No PR checkbox should simulate an unavailable agreement. Do not store signatures, identity documents or private contact information in the public repository. Contribution rights do not automatically grant the contributor's company a product-use license.

[Contributor Agreements](https://contributoragreements.org/ca-cla-chooser/) models distinguish inbound rights, outbound licenses and patent options: these are reviewer references, not an agreement adopted by WebFence. No external form was filled or submitted.

## Closure criterion

Obtain an identifiable opinion covering precise text versions, apply agreed corrections in IT/EN, approve the terms version and activate acceptance before integrating substantial contributions intended for relicensing. Publicly record only date, versions/hashes and approval state; keep documents and personal data private. Commercial offerings also require their corresponding agreements. This dossier alone does not close M0-06.
