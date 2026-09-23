# Memoria del progetto / Project memory

Aggiornato / Updated: 2026-09-23. Documento persistente per riprendere il lavoro; nessun segreto o dato di target reali.

## Italiano

### Fatti confermati dall'autore

- Progetto: WebFence. Autore e responsabile: **Matteo Luigi Feroldi**.
- Repository: [Matte2599/WebFence](https://github.com/Matte2599/WebFence).
- Prodotto: **applicazione desktop nativa**, scelta esplicita; non sostituirla con una UI web o SaaS senza nuova decisione.
- Persone e sviluppatori devono poter studiare, modificare, usare e fare fork gratuitamente nei termini della licenza.
- Chiarimento esplicito: **professionisti senza autorizzazione obbligatoria; aziende con autorizzazione obbligatoria**, anche per uso interno.
- Il testo predisposto permette a professionisti indipendenti persone fisiche analisi retribuite e consegna di report a clienti, anche aziende. Non equiparare una società di consulenza a una persona fisica.
- Distribuzione a scopo di lucro e distribuzione del programma alle aziende richiedono autorizzazione di Matteo Luigi Feroldi; eventuali royalties/licenze sono concordate, non già prezzate.
- UI, documentazione e report in **italiano e inglese**.
- Analisi tradizionale e AI, profili configurabili, CVE aggiornate, report firmati, progetti persistenti fino a cancellazione, retest dei fix e regressioni.
- Modelli AI remoti frontier, locali e gestibili dal programma sono obiettivi; non c'è un modello WebFence addestrato.
- Invicti/Acunetix sono riferimenti di profondità desiderata, non equivalenza provata.

### Decisioni progettuali iniziali

- Core e desktop in Go; Fyne candidato da validare in M0. [ADR-001](DOCS/it/ADR-001-LANGUAGE.md) contiene confronto con Rust e criteri di revisione.
- Desktop a operatore singolo, core separato dalla GUI, SQLite proposto, browser e runtime AI isolati. CLI successiva e opzionale.
- Licenza scelta: **WebFence Community License 1.0**, personalizzata source-available, `LicenseRef-WebFence-Community-1.0`. Non chiamarla open source OSI.
- Il file [LICENSE](LICENSE) contiene testo inglese prevalente e traduzione italiana. Revisione legale non effettuata; CLA e contratto commerciale non sono ancora pronti.
- Nessuna scansione remota o trasmissione a provider è autorizzata dalla sola esistenza di questo progetto.
- La firma è dell'operatore/chiave configurata, non automaticamente dell'autore; non certifica la sicurezza del target.
- Un retest inconcludente non significa fix; una nuova osservazione non dimostra da sola una regressione.

### Stato reale

Creati README IT/EN, DOCS bilingui, roadmap, licenza, policy di contribuzione/sicurezza e istruzioni agenti. Repository inizializzato su `main`, remoto `origin` configurato. Nessun motore, GUI, build, scanner, test funzionale, modello o benchmark implementato. Il commit viene registrato nella cronologia Git; non inserire qui un hash autoreferenziale. Non dedurre un push dall'esistenza del remoto: verificare i riferimenti remoti quando serve.

### Prossimo lavoro

1. M0: prototipo desktop e verifica toolkit, accessibilità, keychain e packaging.
2. Definire piattaforme iniziali e versioni; avviare un core minimale con fixture isolate.
3. Selezionare driver storage e libreria firma, documentando vincoli.
4. Prima di accettare contributi sostanziali esterni per rilicenza commerciale, definire un accordo esplicito; prima della raccolta di segnalazioni sensibili, attivare un canale privato.
5. Proseguire secondo [roadmap](DOCS/ROADMAP.md), aggiornando entrambe le lingue.

Domande ancora aperte: OS/architetture iniziali, soglie hardware misurate, provider/modello, canale privato e termini economici. Non inventare email, prezzi o promesse di rilascio.

## English

### Author-confirmed facts

- Project: WebFence. Author and owner: **Matteo Luigi Feroldi**.
- Repository: [Matte2599/WebFence](https://github.com/Matte2599/WebFence).
- Product: **native desktop application**, explicitly selected; do not replace it with a web UI or SaaS without a new decision.
- Individuals and developers must be able to study, modify, use and fork it for free under its license.
- Explicit clarification: **professionals require no mandatory authorization; companies require authorization**, including internal use.
- The prepared text allows natural-person independent professionals to perform paid assessments and deliver reports to clients, including companies. Do not treat a consulting company as an individual.
- Commercial distribution and distribution of the program to companies require Matteo Luigi Feroldi's authorization; royalties/licenses are negotiated, not already priced.
- UI, documentation and reports in **Italian and English**.
- Traditional and AI analysis, configurable profiles, current CVE intelligence, signed reports, projects retained until deletion, fix retests and regressions.
- Remote frontier models, local models and product-managed models are goals; no trained WebFence model exists.
- Invicti/Acunetix are desired depth references, not proven equivalence.

### Initial design decisions

- Go core and desktop; Fyne candidate to validate in M0. [ADR-001](DOCS/en/ADR-001-LANGUAGE.md) compares Rust and records reconsideration criteria.
- Single-operator desktop, GUI-independent core, proposed SQLite store, isolated browser and AI runtime. Later optional CLI.
- Selected license: custom source-available **WebFence Community License 1.0**, `LicenseRef-WebFence-Community-1.0`. Do not call it OSI open source.
- [LICENSE](LICENSE) contains governing English text and Italian translation. Legal review has not occurred; CLA and commercial contract are not ready.
- The existence of this project does not authorize remote scans or provider uploads.
- The configured operator/key signs reports, not automatically the author; signing does not certify target security.
- Inconclusive retesting does not mean fixed; a new observation alone does not prove a regression.

### Actual state

Created IT/EN READMEs, bilingual DOCS, roadmap, license, contribution/security policies and agent instructions. Repository initialized on `main` with `origin` configured. No engine, GUI, build, scanner, functional tests, model or benchmark is implemented. Git history records the commit; do not place a self-referential hash here. Do not infer a push from the remote configuration: inspect remote refs when needed.

### Next work

1. M0: desktop prototype and toolkit, accessibility, keychain and packaging validation.
2. Define initial platforms and versions; start a minimal core with isolated fixtures.
3. Select storage driver and signing library, recording constraints.
4. Before accepting substantial external contributions for commercial relicensing, establish an explicit agreement; before collecting sensitive reports, enable a private channel.
5. Follow the [roadmap](DOCS/ROADMAP.md), updating both languages.

Open questions: initial OS/architectures, measured hardware requirements, provider/model, private contact and pricing terms. Do not invent email addresses, prices or release promises.
