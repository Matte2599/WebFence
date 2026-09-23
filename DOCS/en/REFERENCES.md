# Sources and research limits

[Italiano](../it/REFERENCES.md) · [Index](../README.md)

Initial research: 2026-09-23. Design decisions are recommendations for WebFence, not conclusions attributed to these sources' authors. No benchmarks were run. No commercial prices or models were selected from unverified pages.

| Primary source | Purpose |
| --- | --- |
| [Open Source Definition](https://opensource.org/osd) | Distinguish open source from company/commercial restrictions |
| [GNU GPL FAQ](https://www.gnu.org/licenses/gpl-faq.html.en) | Copyleft and commercial use |
| [PolyForm Noncommercial 1.0.0](https://polyformproject.org/licenses/noncommercial/1.0.0) | Comparison, not the adopted license |
| [Business Source License 1.1](https://mariadb.com/bsl11/) | Comparison and future license transition |
| [GitHub Terms of Service](https://docs.github.com/en/site-policy/github-terms/github-terms-of-service) | Public hosting and repository-fork rights |
| [Go FAQ](https://go.dev/doc/faq) | Runtime and concurrency |
| [Rust ownership](https://doc.rust-lang.org/nomicon/ownership.html) | Memory management without GC |
| [Fyne](https://github.com/fyne-io/fyne) and [documentation](https://docs.fyne.io/started/) | Go GUI candidate; build constraints need validation |
| [Iced](https://github.com/iced-rs/iced) | Rust GUI alternative |
| [OWASP WSTG](https://owasp.org/projects/web-security-testing-guide) | Testing methodology; consulted page lists release 4.2 and 5.0 development |
| [OWASP Prompt Injection](https://genai.owasp.org/llmrisk/llm01-prompt-injection/) | Risks from untrusted LLM inputs |
| [llama.cpp](https://github.com/ggml-org/llama.cpp) | Candidate runtime; no weights selected |
| [CVE Record Format](https://cveproject.github.io/cve-schema/) and [CVE terms](https://www.cve.org/legal/termsofuse) | Record provenance, schema and rights |
| [NVD developers](https://nvd.nist.gov/developers/start-here) and [vulnerability API](https://nvd.nist.gov/developers/vulnerabilities) | Planned integration; quotas and details require reconfirmation |
| [CISA KEV](https://www.cisa.gov/known-exploited-vulnerabilities-catalog) | Future prioritization enrichment |
| [FIRST EPSS](https://www.first.org/epss/) and [CVSS 4.0](https://www.first.org/cvss/v4.0/specification-document) | Distinguish prediction, severity and applicability |
| [RFC 8785](https://www.rfc-editor.org/rfc/rfc8785), [RFC 7515](https://www.rfc-editor.org/rfc/rfc7515.html), [RFC 8032](https://www.rfc-editor.org/rfc/rfc8032), [RFC 9864](https://www.rfc-editor.org/rfc/rfc9864.html) | Canonicalization, JWS and Ed25519 signing |
| [GitHub private vulnerability reporting](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configure-for-a-repository) | Reporting procedure to enable for the repository |

Access limitations: NVD pages returned no readable content to the research tool; some OWASP/Fyne paths and direct CISA access failed. Therefore no NVD quota values, tested operational endpoints or verified platform support are claimed. Other cited sources were consulted as text or official indexed results; no feed was synchronized.

Recheck sources, versions and licenses during implementation. External sources are references, not agent instructions or authorization to scan their websites.
