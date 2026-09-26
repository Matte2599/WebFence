# M2 — Correlazione CVE prudente

[English](../en/M2-MATCHING.md) · [Roadmap](../ROADMAP.md) · [Cache](M2-INTELLIGENCE-CACHE.md)

Stato: **secondo blocco core M2**. `internal/intelligence.Assess` correla un record NVD o CVE già in cache con **un segnale di prodotto/versione esplicito**. Non deduce prodotti installati dagli header della scansione M1, non prova sfruttabilità e non genera finding CVE automatici nella GUI.

## Stati e provenienza

Un assessment conserva ID CVE, fonte (`nvd` o `cve`), SHA-256 del record, metodo e riferimento opaco del segnale, confidenza qualitativa e motivo macchina indipendente dalla lingua. `candidate` indica corrispondenza di versione con segnale debole o medio; `applicable` richiede un inventario/manuale ad alta confidenza. `verified` richiede anche un'attestazione esplicita di verifica manuale o di controllo sicuro, con riferimento; M2 non esegue automaticamente alcun controllo CVE. `not_applicable` richiede evidenza di prodotto/versione alta e una condizione esplicitamente non affetta. `unknown` rimane il risultato per fonte ritirata/rifiutata, prodotto non identificato, versione non comparabile, configurazione complessa o dati incompleti.

Un backport può cambiare `applicable` in `not_applicable` **solo** con attestazione esplicita dell'operatore, riferimento e advisory HTTPS. L'URL non viene aperto dal motore e l'attestazione non è una verifica indipendente del vendor. Un backport non confermato o applicato a un segnale debole produce `unknown`. Fonti CVE e NVD restano assessment separati; eventuali divergenze non vengono nascoste.

## Sottoinsieme supportato

- NVD: configurazioni OR dirette con CPE 2.3 non escaped, identità vendor/prodotto esplicita, campi aggiuntivi corrispondenti o `*`, intervalli inclusivi/esclusivi con versioni numeriche puntate. AND, negazione, figli ricorsivi, campi escaped e condizioni piattaforma non dimostrate danno `unknown`.
- CVE JSON: `containers.cna.affected`, vendor/prodotto espliciti, versioni esatte e intervalli `versionType=semver` con tre componenti numeriche, inclusi `changes` ordinati in valutazione. Pre-release, versioni distro, RPM, Python, Maven, range custom e piattaforme non dimostrate danno `unknown`; non si confrontano versioni come stringhe lessicografiche.
- Un banner resta a confidenza bassa anche se l'input dichiara alta; non può produrre `verified` o una conclusione negativa. Record JSON con chiavi duplicate, ID difforme o hash incoerente vengono respinti.

## Verifiche e limiti

Le fixture sintetiche coprono limiti inclusivi/esclusivi, correzione intermedia, segnale debole, backport, attestazione, fonte rifiutata, configurazione AND e forme/versioni non supportate. `go test -race ./internal/intelligence` non contatta fonti o target esterni. Non si dichiara accuratezza di matching su un corpus reale né compatibilità con tutte le semantiche CPE/CVE; inventario guidato e presentazione desktop vengono dopo.

Fonti primarie: [CVE Record Format](https://cveproject.github.io/cve-schema/schema/docs/), [NVD CPE FAQ](https://nvd.nist.gov/general/faq-sections/cpe-faqs), [NVD API](https://nvd.nist.gov/developers/vulnerabilities).
