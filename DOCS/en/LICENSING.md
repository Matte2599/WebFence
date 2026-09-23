# Licensing, contributions and governance

[Italiano](../it/LICENSING.md) · [Index](../README.md)

Decision dated 2026-09-23, based on Matteo Luigi Feroldi's requirements and clarifications. [LICENSE](../../LICENSE) contains the applicable text; this guide summarizes it. The English license text governs over the translation subject to mandatory applicable law.

## Decision

**WebFence Community License 1.0**, project identifier `LicenseRef-WebFence-Community-1.0`, plus separate commercial agreements where applicable. Do not use MIT, GPL or “OSI approved” badges or invent a standard SPDX identifier. There is no automatic fee: royalties and charges exist only when agreed.

| Scenario | Rule |
| --- | --- |
| Individual studying, modifying or using WebFence for a personal noncommercial project | Free |
| Personal public fork and free sharing with eligible individuals | Allowed with notices and license |
| Independent professional personally performing a paid assessment | Free, even for a company client |
| Delivering a report to a company client | Allowed; the client does not receive the program |
| Employee or contractor operating the scanner as a company's internal tool | Author's written authorization |
| Consulting company or incorporated practice using the program | Author's written authorization |
| Distribution of the program to a company, even free of charge | Author's written authorization |
| Software sales, OEM, rental, SaaS/API access | Author's written authorization |

An eligible professional is a self-employed natural person, including an individual professional practice; a company is not exempt merely because it provides consultancy. An employee's personal use outside company activities remains allowed. Software authorization does not authorize scanning third-party targets.

## Why not a standard license

| Alternative | Conflict with the requirement |
| --- | --- |
| MIT / Apache-2.0 | Permit commercial and company exploitation without specific author permission. |
| GPL / AGPL | Copyleft is not a commercial-use prohibition; a second commercial license does not make purchase mandatory for all company uses. |
| PolyForm Noncommercial | Does not express the required exception for paid professional assignments; also has its own grants for noncommercial organizations. |
| Business Source License 1.1 | Includes a future transition to an open license; does not retain the requested control indefinitely for every version. |
| Creative Commons NC | Not selected for software licensing; does not itself resolve the professional/company distinction. |

The source-available classification follows from usage restrictions: the [OSI definition](https://opensource.org/osd) does not allow reserving business use. Comparison references: [GPL FAQ](https://www.gnu.org/licenses/gpl-faq.html.en), [PolyForm Noncommercial](https://polyformproject.org/licenses/noncommercial/1.0.0), [BSL 1.1](https://mariadb.com/bsl11/).

## Contributions and relicensing rights

Contributors retain copyright. A PR, DCO or signed commit does not automatically grant the author all rights needed for alternative commercial licenses. Before including external contributions in commercial releases, obtain an explicit agreement granting those rights or exclude those contributions from the relevant distribution.

A CLA has not yet been prepared or activated. [CONTRIBUTING.md](../../CONTRIBUTING.md) requires resolving this before integrating substantial external contributions intended for commercial distribution too. Do not infer assignment from silence.

## Dependencies, models and data

The WebFence license does not change licenses of libraries, engines, imported rules, AI weights, datasets or CVE data. Before distributing them, record inventory, version, provenance, license, notices and compatibility with both distribution modes. A separate process does not automatically remove obligations. Do not copy third-party scanner templates without review.

The custom text is an initial technical draft, not a standard license already validated by a lawyer. Professional review should cover definitions, enforceability, consumer protection, contributor agreements and commercial contracts. No prices, tax identifiers, addresses, email addresses, jurisdiction clauses or payment terms have been invented.

Decision owner: Matteo Luigi Feroldi. Public license changes must have a version and changelog; they do not retroactively revoke rights properly obtained for earlier copies.

M0 preparation: [reviewer dossier and contributor agreement specification](LEGAL-REVIEW.md). This is neither an obtained legal opinion nor an active CLA.
