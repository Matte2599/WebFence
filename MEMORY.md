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

- Core e desktop in Go; **Qt Widgets/MIQT scelto dall’autore**, sostituisce Fyne (ADR-002). [ADR-001](DOCS/it/ADR-001-LANGUAGE.md) contiene confronto con Rust e criteri di revisione.
- Desktop a operatore singolo, core separato dalla GUI, SQLite proposto, browser e runtime AI isolati. CLI successiva e opzionale.
- Licenza scelta: **WebFence Community License 1.0**, personalizzata source-available, `LicenseRef-WebFence-Community-1.0`. Non chiamarla open source OSI.
- Il file [LICENSE](LICENSE) contiene testo inglese prevalente e traduzione italiana. Revisione legale non effettuata; CLA e contratto commerciale non sono ancora pronti.
- Nessuna scansione remota o trasmissione a provider è autorizzata dalla sola esistenza di questo progetto.
- La firma è dell'operatore/chiave configurata, non automaticamente dell'autore; non certifica la sicurezza del target.
- Un retest inconcludente non significa fix; una nuova osservazione non dimostra da sola una regressione.

### Stato reale

Qt Widgets/MIQT 0.14.0 è ora la GUI principale (`cmd/webfence`, `internal/desktop`), Go 1.27.1. Fyne e il modulo Qt annidato rimossi dalla build, conservati nella cronologia. Il prototipo resta offline: 10.000 fixture, filtri, prove/copia, vista semplice/avanzata, IT/EN e menu/scorciatoie lingua. Si salva soltanto `WebFence/ui-language` sotto `os.UserConfigDir()`, con gestione visibile degli errori. Le vecchie preferenze Fyne non sono importate né cancellate. Fixture, cataloghi e preferenze sono pacchetti Go indipendenti da Qt.

Build, vet, verifica moduli, test puri con race detector e self-test Qt aggiornato superati localmente. Bundle Qt macOS verificato; scorciatoie IT/EN e ripristino lingua dopo riapertura provati nella GUI reale. CI finale sul codice `fa8bd32`: quattro job superati nella run 35853356437, macOS ARM64, Windows Server x86-64, Ubuntu x86-64/ARM64; build, test, vet, self-test e bundle macOS. Packaging riusa i flag della build; CI macOS con wrapper C++ `-O0 -g0`, default locale `-O2 -g`. La compilazione a cache fredda resta costosa (~11 minuti osservati nella run finale), non è una misura di prestazioni runtime. [Stato Qt](DOCS/it/QT-DESKTOP.md), [ADR-002 accettato](DOCS/it/ADR-002-GUI.md). Il confronto storico (`90c7c7b`) aveva CI Linux Qt superata e limiti nell’albero/accesso assistivo: non dichiararli risolti dalla migrazione. M0 aperta; nessuno scanner, storage progetti, trasporto di produzione, CVE, firma, AI o keychain. Nessuna release supportata. Valutare maturità MIQT, obblighi Qt LGPL/commerciali e mantenimento Windows 10 oltre Qt 6.12. Nessun acquisto. Commit/push sempre autorizzati; verificarne l’esito, non dedurlo dal remoto configurato.

Task tastiera/accessibilità: menu File/Visualizza/Lingua, filtri con etichette, Ctrl/Cmd+F, F6, Ctrl/Cmd+Shift+E/D, focus restituito al controllo avanzato quando i dettagli vengono nascosti. Traduzione senza reset del modello e senza perdere il testo selezionato. Self-test esteso con focus, selezione e QAccessibleTableInterface; attivazione assistiva esplicita soltanto nel test offscreen. Tab/Shift+Tab verificati con eventi Qt: la tabella non trattiene più Tab, il riepilogo è raggiungibile e i dettagli nascosti vengono saltati. Qt 6.4 offscreen non attiva il bridge: il self-test registra il limite e resetta esplicitamente la cache per verificare i dati, senza simulare una prova di notifiche OS. Verifiche locali e CI finale `6268ae3` superate sui quattro target (run 35857456908, bundle macOS incluso). Prova GUI macOS: nuove scorciatoie e menu verificati, selezione testo conservata al cambio lingua, Tab esce dalla tabella; il ciclo nativo salta il pulsante copia nella configurazione corrente, senza modifica di preferenze OS. Nodo tabella AX intermittente anche dopo la modifica, ancora da verificare con VoiceOver reale; non confondere l’interfaccia Qt con il bridge OS. Vedi [stato Qt](DOCS/it/QT-DESKTOP.md).

Task scope M0: `internal/scope` aggiunge confronto immutabile delle origini HTTP(S), parsing restrittivo, errori a codice senza URL, path/query preservati. Test con race detector e fuzzing locale di 20 secondi superati; laboratorio solo in `lab_test.go`, due server loopback e blocco indipendente delle destinazioni. CI `ed71bec` superata sui quattro target, run 35859069506, incluso fuzzing Linux x86-64 e regressioni Qt. Nessuna rete nella GUI. Non trattare questa policy come broker completo: IP/DNS, reti private, TLS, metodi/percorsi, scadenza e budget restano da implementare. [Contratto e limiti](DOCS/it/M0-SCOPE.md).

### Prossimo lavoro

1. Qt scelto: approfondire menu/focus, albero della tabella, lettori di schermo e stabilità; completare i gate M0 sul toolkit adottato.
2. Completare verifiche OS/versioni minime, packaging e keychain; estendere il laboratorio scope a IP/DNS, budget e cancellazione del futuro trasporto.
3. Selezionare driver storage e libreria firma, documentando vincoli.
4. Prima di accettare contributi sostanziali esterni per rilicenza commerciale, definire un accordo esplicito; prima della raccolta di segnalazioni sensibili, attivare un canale privato.
5. Proseguire secondo [roadmap](DOCS/ROADMAP.md), aggiornando entrambe le lingue.

Domande ancora aperte: versioni minime macOS/Debian, verifica assistiva Qt, soglie hardware misurate, provider/modello, canale privato e termini economici. Non inventare email, prezzi o promesse di rilascio.

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

- Go core and desktop; **Qt Widgets/MIQT selected by the author**, replacing Fyne (ADR-002). [ADR-001](DOCS/en/ADR-001-LANGUAGE.md) compares Rust and records reconsideration criteria.
- Single-operator desktop, GUI-independent core, proposed SQLite store, isolated browser and AI runtime. Later optional CLI.
- Selected license: custom source-available **WebFence Community License 1.0**, `LicenseRef-WebFence-Community-1.0`. Do not call it OSI open source.
- [LICENSE](LICENSE) contains governing English text and Italian translation. Legal review has not occurred; CLA and commercial contract are not ready.
- The existence of this project does not authorize remote scans or provider uploads.
- The configured operator/key signs reports, not automatically the author; signing does not certify target security.
- Inconclusive retesting does not mean fixed; a new observation alone does not prove a regression.

### Actual state

Qt Widgets/MIQT 0.14.0 is now the main GUI (`cmd/webfence`, `internal/desktop`), Go 1.27.1. Fyne and the nested Qt module removed from the build, retained in history. The prototype remains offline: 10,000 fixtures, filters, evidence/copy, simple/advanced view, IT/EN and language menu/shortcuts. Only `WebFence/ui-language` under `os.UserConfigDir()` is saved, with visible error handling. Old Fyne preferences are neither imported nor deleted. Fixtures, catalogs and preferences are Qt-independent Go packages.

Build, vet, module verification, race-enabled pure tests and updated Qt self-test passed locally. macOS Qt bundle verified; IT/EN shortcuts and language restoration after reopening tested in the actual GUI. Final CI for code `fa8bd32`: four jobs passed in run 35853356437, macOS ARM64, Windows Server x86-64, Ubuntu x86-64/ARM64; build, tests, vet, self-test and macOS bundle. Packaging reuses build flags; macOS CI uses `-O0 -g0` C++ wrappers, local default `-O2 -g`. Cold compilation remains costly (~11 minutes observed in the final run), not a runtime performance measurement. [Qt status](DOCS/en/QT-DESKTOP.md), [accepted ADR-002](DOCS/en/ADR-002-GUI.md). The historical comparison (`90c7c7b`) passed Linux Qt CI but had assistive tree/action limitations: migration does not resolve those by itself. M0 open; no scanner, project storage, production transport, CVE, signing, AI or keychain. No supported release. Assess MIQT maturity, Qt LGPL/commercial obligations and Windows 10 maintenance beyond Qt 6.12. No purchase. Commit/push remain authorized; verify success rather than inferring it from remote configuration.

Keyboard/accessibility task: File/View/Language menus, labeled filters, Ctrl/Cmd+F, F6, Ctrl/Cmd+Shift+E/D, focus returned to the advanced control when hiding details. Translation without resetting the model or losing text selection. Self-test extended with focus, selection and QAccessibleTableInterface; explicit accessibility activation only in the offscreen test. Tab/Shift+Tab checked with Qt events: the table no longer retains Tab, the summary is reachable and hidden details are skipped. Qt 6.4 offscreen does not activate the bridge: the self-test logs the limitation and explicitly resets the cache to check data, without claiming an OS notification test. Local checks and final CI for `6268ae3` passed on all four targets (run 35857456908, macOS bundle included). macOS GUI trial: new shortcuts and menu verified, text selection survives translation, Tab leaves the table; the native cycle skips the copy button in the current configuration, without changing OS preferences. The macOS AX table node remains intermittent after this change and still requires actual VoiceOver testing; do not equate the Qt interface with the OS bridge. See [Qt status](DOCS/en/QT-DESKTOP.md).

M0 scope task: `internal/scope` adds immutable HTTP(S) origin comparison, strict parsing, coded errors without URLs and preserved paths/queries. Race-enabled tests and local 20-second fuzzing passed; lab exists only in `lab_test.go`, with two loopback servers and an independent destination fence. CI for `ed71bec` passed on all four targets, run 35859069506, including Linux x86-64 fuzzing and Qt regressions. No GUI networking. Do not treat this policy as a complete broker: IP/DNS, private networks, TLS, methods/paths, expiry and budgets remain unimplemented. [Contract and limitations](DOCS/en/M0-SCOPE.md).

### Next work

1. Qt selected: investigate menus/focus, table tree, screen readers and stability; complete M0 gates for the adopted toolkit.
2. Complete OS/minimum-version, packaging and keychain checks; extend the scope lab to IP/DNS, budgets and cancellation for the future transport.
3. Select storage driver and signing library, recording constraints.
4. Before accepting substantial external contributions for commercial relicensing, establish an explicit agreement; before collecting sensitive reports, enable a private channel.
5. Follow the [roadmap](DOCS/ROADMAP.md), updating both languages.

Open questions: minimum macOS/Debian versions, Qt assistive verification, measured hardware requirements, provider/model, private contact and pricing terms. Do not invent email addresses, prices or release promises.
