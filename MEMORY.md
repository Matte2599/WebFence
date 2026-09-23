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
- Piattaforme confermate: macOS **solo Apple Silicon**, Windows **10/11 x86-64**, Linux **Debian e derivati x86-64/ARM64**; nessun 32 bit.
- UX: semplice e sobria, moderna ma familiare come gli strumenti desktop Windows 2015–2020; percorso guidato e approfondimenti per esperti. Vedi [UX](DOCS/it/UX.md).
- A ogni task aggiornare documentazione, memoria e roadmap, poi commit e push; durante il task quando necessario. Questa autorizzazione è persistente.
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

Fondazione documentale completata. Primo task M0 implementato: Go 1.27.1/Fyne 2.8.1, desktop offline, 10.000 record sintetici a richiesta, filtri, prove inerti, copia e lingua IT/EN persistente; test automatici e workflow CI. CI verificata sul codice `b9556c5`: test e build macOS ARM64, Windows Server x86-64, Ubuntu x86-64/ARM64 superati nella run 35845953673. Linux richiede anche `libwayland-dev`. Le build non attestano l’esecuzione della GUI sui sistemi remoti. Build e bundle locali verificati su macOS 26.6.2 ARM64/Xcode 27. Il [resoconto M0](DOCS/it/M0-DESKTOP.md) distingue esiti locali e CI. **Accessibilità Fyne non approvata**: bridge opzionale con albero incompleto e timeout dello strumento di interazione; build normale senza quel tag. Scanner, storage, scope/rete, CVE, firma, AI e keychain assenti. Nessun benchmark di sicurezza o release supportata. Repository su `main`, remoto `origin` configurato. Il commit viene registrato nella cronologia Git; non inserire qui un hash autoreferenziale. Non dedurre un push dall'esistenza del remoto: verificare i riferimenti remoti quando serve.

### Prossimo lavoro

1. M0: risolvere accessibilità e rivalutare il toolkit sullo stesso scenario; non dichiarare Fyne definitivo.
2. Completare verifiche OS/versioni minime, packaging e keychain; aggiungere laboratorio isolato e primi test scope/rete.
3. Selezionare driver storage e libreria firma, documentando vincoli.
4. Prima di accettare contributi sostanziali esterni per rilicenza commerciale, definire un accordo esplicito; prima della raccolta di segnalazioni sensibili, attivare un canale privato.
5. Proseguire secondo [roadmap](DOCS/ROADMAP.md), aggiornando entrambe le lingue.

Domande ancora aperte: versioni minime macOS/Debian, toolkit accessibile, soglie hardware misurate, provider/modello, canale privato e termini economici. Non inventare email, prezzi o promesse di rilascio.

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
- Confirmed platforms: **Apple Silicon only** macOS, **Windows 10/11 x86-64**, **Debian and derivatives x86-64/ARM64**; no 32-bit support.
- UX: simple and restrained, modern but familiar like Windows desktop tools from 2015–2020; guided workflow with expert depth. See [UX](DOCS/en/UX.md).
- Update documentation, memory and roadmap for every task, then commit and push; also during work when necessary. This authorization persists.
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

Documentation foundation complete. First M0 task implemented: Go 1.27.1/Fyne 2.8.1, offline desktop, 10,000 on-demand synthetic records, filters, inert evidence, copy and persistent IT/EN language; automated tests and CI workflow. CI verified for code `b9556c5`: tests and macOS ARM64, Windows Server x86-64, Ubuntu x86-64/ARM64 builds passed in run 35845953673. Linux also requires `libwayland-dev`. Builds do not establish GUI execution on remote systems. Local build and bundle verified on macOS 26.6.2 ARM64/Xcode 27. The [M0 report](DOCS/en/M0-DESKTOP.md) separates local and CI results. **Fyne accessibility is not approved**: optional bridge with incomplete tree and interaction-tool timeout; normal build excludes that tag. Scanner, storage, scope/network, CVE, signing, AI and keychain are absent. No security benchmark or supported release. Repository on `main`, with `origin` configured. Git history records the commit; do not place a self-referential hash here. Do not infer a push from the remote configuration: inspect remote refs when needed.

### Next work

1. M0: resolve accessibility and reassess the toolkit against the same scenario; do not declare Fyne final.
2. Complete OS/minimum-version, packaging and keychain checks; add the isolated lab and first scope/network tests.
3. Select storage driver and signing library, recording constraints.
4. Before accepting substantial external contributions for commercial relicensing, establish an explicit agreement; before collecting sensitive reports, enable a private channel.
5. Follow the [roadmap](DOCS/ROADMAP.md), updating both languages.

Open questions: minimum macOS/Debian versions, accessible toolkit, measured hardware requirements, provider/model, private contact and pricing terms. Do not invent email addresses, prices or release promises.
