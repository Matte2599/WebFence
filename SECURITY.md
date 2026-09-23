# Sicurezza / Security

## Italiano

### Stato supportato

WebFence ha un prototipo desktop M0 con esempi sintetici offline; non è uno scanner operativo. Non esistono release eseguibili supportate. Il canale **GitHub Private vulnerability reporting** è stato abilitato e verificato nelle impostazioni del repository il 2026-09-23; il riepilogo Security lo indica come Enabled. La policy sarà aggiornata con le versioni supportate prima della prima release.

### Segnalare un problema di WebFence

Usare **Report a vulnerability** nella [pagina Security Advisories](https://github.com/Matte2599/WebFence/security/advisories). Se il canale risulta temporaneamente indisponibile, aprire soltanto una issue con titolo **Private security contact request**, senza descrizione tecnica sensibile, e attendere che Matteo Luigi Feroldi indichi un canale privato. Non pubblicare prove, credenziali, dati personali o dettagli di vulnerabilità non ancora corretta nelle issue pubbliche.

La segnalazione privata dovrebbe contenere versione/commit, piattaforma, impatto, prerequisiti, riproduzione in laboratorio e prove redatte. Non ci sono SLA o ricompense promessi. Coordinare con il maintainer la divulgazione dopo valutazione e correzione.

### Limiti delle verifiche

L'autorizzazione a contribuire non consente test sui sistemi dell'autore o di terzi. Preferire fixture locali e usare soltanto target autorizzati. Per un problema trovato in un sito analizzato da WebFence, rivolgersi al titolare di quel sito tramite il suo canale: non caricare il report nel repository WebFence.

Il [modello delle minacce](DOCS/it/THREAT-MODEL.md) descrive controlli previsti, non garanzie già implementate. La firma di un report non certifica la sicurezza del target.

## English

### Supported state

WebFence has an M0 desktop prototype with offline synthetic examples; it is not an operational scanner. There are no supported executable releases. **GitHub Private vulnerability reporting** was enabled and verified in repository settings on 2026-09-23; the Security overview shows Enabled. This policy will list supported versions before the first release.

### Reporting a WebFence vulnerability

Use **Report a vulnerability** on the [Security Advisories page](https://github.com/Matte2599/WebFence/security/advisories). If the channel is temporarily unavailable, open only an issue titled **Private security contact request**, without sensitive technical details, and wait for Matteo Luigi Feroldi to provide a private channel. Do not post evidence, credentials, personal data or details of an unpatched vulnerability in public issues.

A private report should include version/commit, platform, impact, prerequisites, lab reproduction and redacted evidence. No response SLA or bounty is promised. Coordinate disclosure with the maintainer after assessment and remediation.

### Assessment boundaries

Permission to contribute does not authorize testing the author's or third parties' systems. Prefer local fixtures and use only authorized targets. For a vulnerability found in a site assessed by WebFence, contact that site's owner through their own channel; do not upload the report to the WebFence repository.

The [threat model](DOCS/en/THREAT-MODEL.md) describes planned controls, not implemented guarantees. A report signature does not certify target security.
