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
- Desktop a operatore singolo, core separato dalla GUI, driver SQLite selezionato (ADR-004), store del prodotto pianificato, browser e runtime AI isolati. CLI successiva e opzionale.
- Licenza scelta: **WebFence Community License 1.0**, personalizzata source-available, `LicenseRef-WebFence-Community-1.0`. Non chiamarla open source OSI.
- Il file [LICENSE](LICENSE) contiene testo inglese prevalente e traduzione italiana. Revisione legale non effettuata; CLA e contratto commerciale non sono ancora pronti.
- Nessuna scansione remota o trasmissione a provider è autorizzata dalla sola esistenza di questo progetto.
- La firma è dell'operatore/chiave configurata, non automaticamente dell'autore; non certifica la sicurezza del target.
- Un retest inconcludente non significa fix; una nuova osservazione non dimostra da sola una regressione.

### Stato reale

Qt Widgets/MIQT 0.14.0 è ora la GUI principale (`cmd/webfence`, `internal/desktop`), Go 1.27.1. Fyne e il modulo Qt annidato rimossi dalla build, conservati nella cronologia. Il prototipo resta offline: 10.000 fixture, filtri, prove/copia, vista semplice/avanzata, IT/EN e menu/scorciatoie lingua. Si salva soltanto `WebFence/ui-language` sotto `os.UserConfigDir()`, con gestione visibile degli errori. Le vecchie preferenze Fyne non sono importate né cancellate. Fixture, cataloghi e preferenze sono pacchetti Go indipendenti da Qt.

Build, vet, verifica moduli, test puri con race detector e self-test Qt aggiornato superati localmente. Bundle Qt macOS verificato; scorciatoie IT/EN e ripristino lingua dopo riapertura provati nella GUI reale. CI finale sul codice `fa8bd32`: quattro job superati nella run 35853356437, macOS ARM64, Windows Server x86-64, Ubuntu x86-64/ARM64; build, test, vet, self-test e bundle macOS. Packaging riusa i flag della build; CI macOS con wrapper C++ `-O0 -g0`, default locale `-O2 -g`. La compilazione a cache fredda resta costosa (~11 minuti osservati nella run finale), non è una misura di prestazioni runtime. [Stato Qt](DOCS/it/QT-DESKTOP.md), [ADR-002 accettato](DOCS/it/ADR-002-GUI.md). Il confronto storico (`90c7c7b`) aveva CI Linux Qt superata e limiti nell’albero/accesso assistivo: non dichiararli risolti dalla migrazione. M0 aperta; nessuno scanner, storage progetti, trasporto di produzione, CVE, report firmati nella GUI, AI o keychain. Nessuna release supportata. Valutare maturità MIQT, obblighi Qt LGPL/commerciali e mantenimento Windows 10 oltre Qt 6.12. Nessun acquisto. Commit/push sempre autorizzati; verificarne l’esito, non dedurlo dal remoto configurato.

Task tastiera/accessibilità: menu File/Visualizza/Lingua, filtri con etichette, Ctrl/Cmd+F, F6, Ctrl/Cmd+Shift+E/D, focus restituito al controllo avanzato quando i dettagli vengono nascosti. Traduzione senza reset del modello e senza perdere il testo selezionato. Self-test esteso con focus, selezione e QAccessibleTableInterface; attivazione assistiva esplicita soltanto nel test offscreen. Tab/Shift+Tab verificati con eventi Qt: la tabella non trattiene più Tab, il riepilogo è raggiungibile e i dettagli nascosti vengono saltati. Qt 6.4 offscreen non attiva il bridge: il self-test registra il limite e resetta esplicitamente la cache per verificare i dati, senza simulare una prova di notifiche OS. Verifiche locali e CI finale `6268ae3` superate sui quattro target (run 35857456908, bundle macOS incluso). Prova GUI macOS: nuove scorciatoie e menu verificati, selezione testo conservata al cambio lingua, Tab esce dalla tabella; il ciclo nativo salta il pulsante copia nella configurazione corrente, senza modifica di preferenze OS. Nodo tabella AX intermittente anche dopo la modifica, ancora da verificare con VoiceOver reale; non confondere l’interfaccia Qt con il bridge OS. Vedi [stato Qt](DOCS/it/QT-DESKTOP.md).

Task scope M0: `internal/scope` aggiunge confronto immutabile delle origini HTTP(S), parsing restrittivo, errori a codice senza URL, path/query preservati. Test con race detector e fuzzing locale di 20 secondi superati; laboratorio solo in `lab_test.go`, due server loopback e blocco indipendente delle destinazioni. CI `ed71bec` superata sui quattro target, run 35859069506, incluso fuzzing Linux x86-64 e regressioni Qt. Nessuna rete nella GUI. Questa policy rimane soltanto il livello URL; il task successivo aggiunge le prove di trasporto loopback descritte sotto. [Contratto e limiti](DOCS/it/M0-SCOPE.md).

Task trasporto M0: `internal/transport.NewLab` accetta solo grant di origini con IP loopback espliciti. DNS controllato per ogni passaggio, rifiuto delle risposte miste, connessione all’IP verificato e controllo del peer; TLS verificato sul nome originale. GET senza cookie/header del chiamante, proxy disabilitati, HTTP/1 senza pooling/retry; budget atomico condiviso, concorrenza, timeout Fetch/run, limiti body/header, compressione respinta, cancellazione/Close. Test race e vet locali superati, inclusi IPv6 loopback e certificati errati. CI `8f189eb` verde sui quattro target (run 35861040736), comprese regressioni Qt e bundle macOS. Nessun collegamento alla GUI o traffico esterno. [ADR-003](DOCS/it/ADR-003-TRANSPORT.md) chiude il gate dei primi test di laboratorio M0, non il trasporto di produzione: CIDR/reti reali, autorizzazione, metodi/percorsi, rate limit e redazione restano da fare.

Task SQLite/JWS M0: mattn/go-sqlite3 v1.14.52 (SQLite 3.53.4 incorporato), jwx/v4 v4.5.0 selezionati in [ADR-004](DOCS/it/ADR-004-STORAGE-SIGNATURE.md). `internal/foundation` contiene solo test SQLite (transazioni/riapertura, vincoli, lock, cancellazione); `internal/signature` firma/verifica byte con profilo compact Ed25519, chiave fidata esterna, limiti e header stretti. Test race, Go complessivi, vet/moduli e fuzz JWS locale 20 secondi superati (2.507.150 esecuzioni osservate). CI `f1d2a84` superata sui quattro target, [run 35863431196](https://github.com/Matte2599/WebFence/actions/runs/35863431196), comprese regressioni Qt, fuzz JWS Linux x86-64 e bundle macOS. `govulncheck` v1.8.0 con test non ha rilevato vulnerabilità Go note; non copre tutte le dipendenze native. Nessuna persistenza progetti, JCS, bundle report o gestione chiavi; GUI invariata. Portachiavi ancora da selezionare. Ricognizione preliminare: go-keyring v0.2.8 usa il comando security su macOS e attese di prompt D-Bus senza context su Linux; 99designs/keyring v1.2.2 controlla risultati vuoti prima dell’errore in Get macOS. Valutare backend nativi isolati (keybase/go-keychain v0.0.1, wincred, Secret Service con context), senza assumere compatibilità o test già eseguiti; nessuna di queste dipendenze aggiunta al modulo. Obiettivo corrente: completare tutta M0, verificare ogni gate con evidenze e concludere con collaudo complessivo, documentazione e commit/push/CI; non segnare chiusa per i soli test automatizzati.

Registro di chiusura M0: [matrice e procedura desktop](DOCS/it/M0-VALIDATION.md), con gate distinti e nessuna percentuale fittizia. Self-test Qt offscreen aggiuntivi a scala 1.5 e 2 superati localmente, senza prova visiva. Bundle macOS locale esistente: verifica codesign --deep --strict superata; otool del binario principale punta ai framework interni e librerie di sistema. Il bundle include anche numerose dylib terze (ICU, glib, freetype, ecc.): inventario licenze/sorgenti e macchina pulita ancora da completare, non pubblicare il bundle come release pronta. Il canale GitHub Private vulnerability reporting era disabilitato (GET pubblico: enabled=false); abilitato nelle impostazioni con conferma e riepilogo Security Enabled il 2026-09-23. SECURITY aggiornato; nessuna segnalazione inviata. Disponibilità di macchine Windows/NVDA e Debian/Orca richiesta all’autore, risposta non ancora registrata. [Dossier legale](DOCS/it/LEGAL-REVIEW.md) e specifica CLA predisposti; l’autore ha confermato che la revisione professionale è ancora da organizzare. Nessun CLA approvato, nessuna modifica di LICENSE o contratto sottoscritto.

Task portachiavi M0: aggiunto `internal/credentials`, namespace fisso, ID minuscoli, segreti binari 1–2048 byte, errori redatti e nessun fallback. macOS: adattatore Security.framework classico, controllo UI globale serializzato/ripristinato, API deprecate da rivalutare per app firmata. Windows: wincred v1.2.3; Linux: godbus v5.2.2, Secret Service locale senza prompt, connessione privata e timeout 3 secondi. [ADR-005](DOCS/it/ADR-005-CREDENTIALS.md) registra limiti, inclusa cancellazione non interrompibile dentro le API macOS/Windows. Test macOS con portachiavi temporaneo separato superati, inclusi blocco e assenza; suite Go/vet/moduli superate. CI e launcher Linux isolato predisposti; nessuna prova runtime Windows/Linux ancora dichiarata. GUI invariata.

Prima CI portachiavi `aa722e3`: prove native macOS superate; Linux x86-64/ARM64 ha scoperto il MIME `text/plain` restituito da GNOME Keyring anche per segreti binari. Corretto il controllo: dati sempre opachi, restano controlli sessione/parametri/dimensione; launcher senza attivazione concorrente del daemon. Verifica successiva in attesa.

Ricognizione packaging: [dossier](DOCS/it/M0-PACKAGING.md). Bundle locale preesistente c4ce9cab con vcs.modified=true, non HEAD: 28 Mach-O, 19 richiedono macOS 26 e 9 macOS 14; nessuna dipendenza assoluta esterna dopo esclusione LC_ID_DYLIB. Questo artefatto non è compatibile macOS 13/15 per semplice cambio flag. Individuate SBOM Homebrew/Qt installate e famiglie di dylib, non ancora inventario completo del prodotto. Chiesto all’autore minimo 13/15/26, risposta pendente.

Prova aggiuntiva Windows predisposta: sottoprocesso e thread con token anonimo, ripristino del token, Get/Set/Delete devono negare accesso e voce sintetica originale deve restare integra. Cross-compilazione/vet superati; prova nativa in attesa. Non blocca la sessione OS personale.

CI credenziali `a916fab` superata sui quattro target, run 35867740266 (verifica finale API: success). La prova Windows con token anonimo del commit successivo resta da eseguire. Packaging macOS aggiornato a staging temporaneo e pubblicazione con rollback; build/codesign locali e guasto compilatore con artefatto precedente preservato superati. Collaudo GUI: layout standard/150% ispezionati ma crash SIGSEGV riprodotto al reset filtro con riga selezionata e successivo cambio lingua via AX. Copia diagnostica temporanea con stderr; causa nativa ancora da isolare. M0 resta aperta.

### Prossimo lavoro

1. Qt scelto: approfondire menu/focus, albero della tabella, lettori di schermo e stabilità; completare i gate M0 sul toolkit adottato.
2. Completare verifiche OS/versioni minime, packaging e keychain; mantenere il broker loopback separato dal futuro trasporto M1 per target reali.
3. Completare scelta/prove del portachiavi; SQLite/JWS confermati in CI; storage progetti e bundle report restano M1/M2.
4. Prima di accettare contributi sostanziali esterni per rilicenza commerciale, definire un accordo esplicito e completare la revisione legale. Canale privato GitHub ora attivo.
5. Proseguire secondo [roadmap](DOCS/ROADMAP.md), aggiornando entrambe le lingue.

Domande ancora aperte: versioni minime macOS/Debian, verifica assistiva Qt, soglie hardware misurate, provider/modello e termini economici. Non inventare email, prezzi o promesse di rilascio.

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
- Single-operator desktop, GUI-independent core, selected SQLite driver (ADR-004), planned product store, isolated browser and AI runtime. Later optional CLI.
- Selected license: custom source-available **WebFence Community License 1.0**, `LicenseRef-WebFence-Community-1.0`. Do not call it OSI open source.
- [LICENSE](LICENSE) contains governing English text and Italian translation. Legal review has not occurred; CLA and commercial contract are not ready.
- The existence of this project does not authorize remote scans or provider uploads.
- The configured operator/key signs reports, not automatically the author; signing does not certify target security.
- Inconclusive retesting does not mean fixed; a new observation alone does not prove a regression.

### Actual state

Qt Widgets/MIQT 0.14.0 is now the main GUI (`cmd/webfence`, `internal/desktop`), Go 1.27.1. Fyne and the nested Qt module removed from the build, retained in history. The prototype remains offline: 10,000 fixtures, filters, evidence/copy, simple/advanced view, IT/EN and language menu/shortcuts. Only `WebFence/ui-language` under `os.UserConfigDir()` is saved, with visible error handling. Old Fyne preferences are neither imported nor deleted. Fixtures, catalogs and preferences are Qt-independent Go packages.

Build, vet, module verification, race-enabled pure tests and updated Qt self-test passed locally. macOS Qt bundle verified; IT/EN shortcuts and language restoration after reopening tested in the actual GUI. Final CI for code `fa8bd32`: four jobs passed in run 35853356437, macOS ARM64, Windows Server x86-64, Ubuntu x86-64/ARM64; build, tests, vet, self-test and macOS bundle. Packaging reuses build flags; macOS CI uses `-O0 -g0` C++ wrappers, local default `-O2 -g`. Cold compilation remains costly (~11 minutes observed in the final run), not a runtime performance measurement. [Qt status](DOCS/en/QT-DESKTOP.md), [accepted ADR-002](DOCS/en/ADR-002-GUI.md). The historical comparison (`90c7c7b`) passed Linux Qt CI but had assistive tree/action limitations: migration does not resolve those by itself. M0 open; no scanner, project storage, production transport, CVE, GUI report signing, AI or keychain. No supported release. Assess MIQT maturity, Qt LGPL/commercial obligations and Windows 10 maintenance beyond Qt 6.12. No purchase. Commit/push remain authorized; verify success rather than inferring it from remote configuration.

Keyboard/accessibility task: File/View/Language menus, labeled filters, Ctrl/Cmd+F, F6, Ctrl/Cmd+Shift+E/D, focus returned to the advanced control when hiding details. Translation without resetting the model or losing text selection. Self-test extended with focus, selection and QAccessibleTableInterface; explicit accessibility activation only in the offscreen test. Tab/Shift+Tab checked with Qt events: the table no longer retains Tab, the summary is reachable and hidden details are skipped. Qt 6.4 offscreen does not activate the bridge: the self-test logs the limitation and explicitly resets the cache to check data, without claiming an OS notification test. Local checks and final CI for `6268ae3` passed on all four targets (run 35857456908, macOS bundle included). macOS GUI trial: new shortcuts and menu verified, text selection survives translation, Tab leaves the table; the native cycle skips the copy button in the current configuration, without changing OS preferences. The macOS AX table node remains intermittent after this change and still requires actual VoiceOver testing; do not equate the Qt interface with the OS bridge. See [Qt status](DOCS/en/QT-DESKTOP.md).

M0 scope task: `internal/scope` adds immutable HTTP(S) origin comparison, strict parsing, coded errors without URLs and preserved paths/queries. Race-enabled tests and local 20-second fuzzing passed; lab exists only in `lab_test.go`, with two loopback servers and an independent destination fence. CI for `ed71bec` passed on all four targets, run 35859069506, including Linux x86-64 fuzzing and Qt regressions. No GUI networking. This policy remains the URL layer only; the subsequent task adds the loopback transport checks below. [Contract and limitations](DOCS/en/M0-SCOPE.md).

M0 transport task: `internal/transport.NewLab` accepts only origin grants with explicit loopback IPs. Controlled DNS per hop, rejection of mixed answers, connection to the validated IP and peer checks; TLS verifies the original name. GET without caller cookies/headers, disabled proxies, HTTP/1 without pooling/retries; shared atomic attempt budget, concurrency, Fetch/run timeouts, body/header limits, rejected compression, cancellation/Close. Local race tests and vet passed, including IPv6 loopback and invalid certificates. CI for `8f189eb` passed on all four targets (run 35861040736), including Qt regressions and macOS bundle. No GUI integration or external traffic. [ADR-003](DOCS/en/ADR-003-TRANSPORT.md) closes the initial M0 lab test gate, not production transport: CIDR/real networks, authorization, methods/paths, rate limiting and redaction remain pending.

M0 SQLite/JWS task: mattn/go-sqlite3 v1.14.52 (embedded SQLite 3.53.4), jwx/v4 v4.5.0 selected in [ADR-004](DOCS/en/ADR-004-STORAGE-SIGNATURE.md). `internal/foundation` contains SQLite tests only (transactions/reopen, constraints, locks, cancellation); `internal/signature` signs/verifies bytes with a compact Ed25519 profile, externally trusted key, limits and strict headers. Race tests, complete Go tests, vet/module checks and local 20-second JWS fuzzing passed (2,507,150 observed executions). CI for `f1d2a84` passed on all four targets, [run 35863431196](https://github.com/Matte2599/WebFence/actions/runs/35863431196), including Qt regressions, Linux x86-64 JWS fuzzing and macOS bundle. `govulncheck` v1.8.0 with tests found no known Go vulnerabilities; it does not cover all native dependencies. No project persistence, JCS, report bundle or key management; GUI unchanged. Keychain remains unselected. Preliminary inspection: go-keyring v0.2.8 uses the security command on macOS and context-free D-Bus prompt waits on Linux; 99designs/keyring v1.2.2 checks empty results before errors in macOS Get. Assess isolated native backends (keybase/go-keychain v0.0.1, wincred, context-aware Secret Service), without assuming compatibility or completed tests; none of these dependencies was added to the module. Current objective: complete all M0, verify every gate with evidence and conclude with comprehensive validation, documentation and commit/push/CI; do not close based solely on automated tests.

M0 closure register: [matrix and desktop procedure](DOCS/en/M0-VALIDATION.md), separate gates without invented percentages. Additional local Qt offscreen self-tests at scale 1.5 and 2 passed, without visual validation. Existing local macOS bundle: codesign --deep --strict verification passed; main executable otool points to bundled frameworks and system libraries. The bundle also contains numerous third-party dylibs (ICU, glib, freetype, etc.): license/source inventory and clean-machine verification remain pending; do not publish it as a ready release. GitHub Private vulnerability reporting was disabled (public GET: enabled=false); enabled in settings with confirmation and Security overview Enabled on 2026-09-23. SECURITY updated; no report submitted. Author asked about Windows/NVDA and Debian/Orca machine availability; no answer recorded yet. [Legal dossier](DOCS/en/LEGAL-REVIEW.md) and CLA specification prepared; the author confirmed professional review still needs to be arranged. No approved CLA, LICENSE change or signed agreement.

M0 credential task: added `internal/credentials`, fixed namespace, lowercase IDs, 1–2048 byte binary secrets, redacted errors and no fallback. macOS: classic Security.framework adapter, serialized/restored global UI control, deprecated APIs to reassess for signed apps. Windows: wincred v1.2.3; Linux: godbus v5.2.2, local Secret Service without prompts, private connection and three-second timeout. [ADR-005](DOCS/en/ADR-005-CREDENTIALS.md) records limits, including noninterruptible macOS/Windows API calls. Separate temporary macOS keychain tests passed, including locked/absent states; Go/vet/module checks passed. CI and isolated Linux launcher prepared; no Windows/Linux runtime result claimed yet. GUI unchanged.

First credential CI `aa722e3`: native macOS tests passed; Linux x86-64/ARM64 exposed GNOME Keyring returning `text/plain` for binary secrets. Corrected the check: data stays opaque; session/parameter/size checks remain. Launcher no longer activates a concurrent daemon. Follow-up verification pending.

Packaging investigation: [dossier](DOCS/en/M0-PACKAGING.md). Existing local c4ce9cab bundle with vcs.modified=true, not HEAD: 28 Mach-O files, 19 require macOS 26 and 9 macOS 14; no external absolute dependency after excluding LC_ID_DYLIB. This artifact cannot support macOS 13/15 through a flag change alone. Installed Homebrew/Qt SBOMs and dylib families identified, not yet a full product inventory. Asked author for minimum 13/15/26; answer pending.

Additional Windows trial prepared: subprocess and thread with anonymous token, token restoration, Get/Set/Delete must deny access and original synthetic item must stay intact. Cross-compilation/vet passed; native test pending. Does not lock the personal OS session.

Credential CI `a916fab` passed all four targets, run 35867740266 (final API verification: success). The subsequent Windows anonymous-token trial still needs execution. macOS packaging now stages temporarily and publishes with rollback; local build/codesign and injected compiler failure preserving the previous artifact passed. GUI trial: standard/150% layouts inspected, but SIGSEGV reproduced after filter reset with a selected row and subsequent language change through AX. Temporary diagnostic copy captures stderr; native cause still to isolate. M0 remains open.

### Next work

1. Qt selected: investigate menus/focus, table tree, screen readers and stability; complete M0 gates for the adopted toolkit.
2. Complete OS/minimum-version, packaging and keychain checks; keep the loopback broker separate from the future M1 real-target transport.
3. Complete keychain selection/tests; SQLite/JWS confirmed in CI; project storage and report bundles remain M1/M2.
4. Before accepting substantial external contributions for commercial relicensing, establish an explicit agreement and complete legal review. GitHub private reporting is now active.
5. Follow the [roadmap](DOCS/ROADMAP.md), updating both languages.

Open questions: minimum macOS/Debian versions, Qt assistive verification, measured hardware requirements, provider/model and pricing terms. Do not invent email addresses, prices or release promises.
