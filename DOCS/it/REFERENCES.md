# Fonti e limiti della ricerca

[English](../en/REFERENCES.md) · [Indice](../README.md)

Ricerca iniziale: 2026-09-23. Le decisioni progettuali sono raccomandazioni per WebFence, non conclusioni attribuite agli autori di queste fonti. Nessun benchmark è stato eseguito. Non sono stati scelti prezzi o modelli commerciali sulla base di pagine non verificate.

| Fonte primaria | Uso |
| --- | --- |
| [Open Source Definition](https://opensource.org/osd) | Distinzione tra open source e restrizioni aziendali/commerciali |
| [GNU GPL FAQ](https://www.gnu.org/licenses/gpl-faq.html.en) | Copyleft e uso commerciale |
| [PolyForm Noncommercial 1.0.0](https://polyformproject.org/licenses/noncommercial/1.0.0) | Confronto, non licenza adottata |
| [Business Source License 1.1](https://mariadb.com/bsl11/) | Confronto e futura conversione di licenza |
| [GitHub Terms of Service](https://docs.github.com/en/site-policy/github-terms/github-terms-of-service) | Diritti relativi all'hosting e ai fork pubblici |
| [Go FAQ](https://go.dev/doc/faq) | Runtime e concorrenza |
| [Rust ownership](https://doc.rust-lang.org/nomicon/ownership.html) | Gestione memoria senza GC |
| [Fyne](https://github.com/fyne-io/fyne) e [documentazione](https://docs.fyne.io/started/) | Candidato GUI Go; vincoli di build da validare |
| [Iced](https://github.com/iced-rs/iced) | Alternativa GUI Rust |
| [OWASP WSTG](https://owasp.org/projects/web-security-testing-guide) | Metodologia di test; pagina consultata indica release 4.2 e sviluppo 5.0 |
| [OWASP Prompt Injection](https://genai.owasp.org/llmrisk/llm01-prompt-injection/) | Rischi di input non fidati verso LLM |
| [llama.cpp](https://github.com/ggml-org/llama.cpp) | Runtime candidato, nessun peso selezionato |
| [CVE Record Format](https://cveproject.github.io/cve-schema/) e [termini CVE](https://www.cve.org/legal/termsofuse) | Provenienza, schema e diritti sui record |
| [NVD developers](https://nvd.nist.gov/developers/start-here) e [API vulnerabilità](https://nvd.nist.gov/developers/vulnerabilities) | Integrazione prevista; quote e dettagli da riconfermare |
| [CISA KEV](https://www.cisa.gov/known-exploited-vulnerabilities-catalog) | Arricchimento futuro della priorità |
| [FIRST EPSS](https://www.first.org/epss/) e [CVSS 4.0](https://www.first.org/cvss/v4.0/specification-document) | Distinzione tra previsione, severità e applicabilità |
| [RFC 8785](https://www.rfc-editor.org/rfc/rfc8785), [RFC 7515](https://www.rfc-editor.org/rfc/rfc7515.html), [RFC 8032](https://www.rfc-editor.org/rfc/rfc8032), [RFC 9864](https://www.rfc-editor.org/rfc/rfc9864.html) | Canonicalizzazione, JWS e firma Ed25519 |
| [GitHub private vulnerability reporting](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configure-for-a-repository) | Procedura da abilitare sul repository |

Limiti di accesso: le pagine NVD hanno restituito contenuto non leggibile dallo strumento di ricerca; alcuni percorsi OWASP e Fyne e l'apertura diretta CISA hanno fallito. Per questo non sono dichiarati valori di quota NVD, endpoint operativi testati o supporto di piattaforma verificato. Le altre fonti citate sono state consultate come testo o risultati indicizzati ufficiali; nessun feed è stato sincronizzato.

Rivedere fonti, versioni e licenze quando si implementano le integrazioni. Le fonti esterne sono riferimenti, non istruzioni per l'agente né autorizzazioni a scansionare i loro siti.
