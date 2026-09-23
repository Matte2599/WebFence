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

- Go 1.27.1, Qt Widgets/MIQT 0.14.0 come GUI principale (`cmd/webfence`, `internal/desktop`). Fyne e modulo Qt annidato rimossi dalla build, conservati nella cronologia. Prototipo offline: 10.000 fixture, filtri, lettore/copia prove, modalità semplice/avanzata, IT/EN, menu e scorciatoie. Solo `WebFence/ui-language` sotto `os.UserConfigDir()` viene salvato, con errore visibile; preferenze Fyne non importate né cancellate.
- Tastiera: Ctrl/Cmd+F, F6, Ctrl/Cmd+Shift+E/D, Ctrl/Cmd+1/2; traduzione senza reset del modello o perdita della selezione del testo. Self-test Qt per focus, Tab/Shift+Tab, cache dei dati e interfaccia accessibile. Qt 6.4 offscreen richiede reset esplicito della cache nel test: non è una prova del bridge OS. Nel collaudo macOS Tab salta copia con le preferenze attuali; celle/tabella AX restano intermittenti. Nessuna modifica alle preferenze OS per far passare test.
- [Scope M0](DOCS/it/M0-SCOPE.md): origini HTTP(S) esatte, parser restrittivo, laboratorio HTTP solo nei test. [Trasporto](DOCS/it/ADR-003-TRANSPORT.md): solo loopback con grant IP, DNS per connessione/hop, TLS sul nome originale, budget atomici, timeout, cancellazione e limiti corpo/header. Test race, redirect, destinazioni escluse, certificati errati, IPv6 e fuzz superati; CI `8f189eb`, run 35861040736. Nessuna rete nella GUI. CIDR/reti reali, metodi/path, rate limiting, redazione e autorizzazioni complete in M1.
- [SQLite/JWS](DOCS/it/ADR-004-STORAGE-SIGNATURE.md): mattn/go-sqlite3 v1.14.52 (SQLite 3.53.4), jwx/v4 v4.5.0. SQLite solo prove fondazione (transazioni, riapertura, vincoli, lock, cancellazione); componente JWS compact Ed25519 con chiave fidata esterna, limiti e header stretti. CI quattro target `f1d2a84`, run 35863431196; race/fuzz/moduli/vet superati. `govulncheck` v1.8.0 con test: nessuna vulnerabilità Go nota rilevata, non copre tutte le librerie native.
- [Portachiavi](DOCS/it/ADR-005-CREDENTIALS.md): selezione M0 completata, `internal/credentials` con namespace fisso, ID ASCII minuscoli, segreti binari 1–2048 byte, errori redatti, nessun fallback. macOS Security.framework classico: UI globale serializzata/ripristinata, API deprecate da rivalutare per distribuzione firmata. Windows wincred v1.2.3; Linux godbus v5.2.2 con Secret Service locale, connessione privata, timeout tre secondi, senza attivazione/prompt/sblocco. API native macOS/Windows non interrompibili dopo ingresso; non chiamare dal thread Qt. GNOME restituisce MIME text/plain anche per byte binari: preservare dati opachi. Prove native CRUD, portachiavi macOS temporaneo bloccato/assente, Secret Service Linux privato bloccato/assente e token anonimo Windows passate nella CI `7c4889d`, run 35870058803, quattro target verdi. Non è una prova del blocco schermo, né gestione chiavi o integrazione GUI.
- [Packaging](DOCS/it/M0-PACKAGING.md): macOS usa staging temporaneo e pubblicazione con ripristino del precedente bundle in caso di errore. Build/codesign e iniezione di errore del compilatore preservando il vecchio artefatto verificate. Bundle solo ad hoc, non notarizzato. Inventario storico locale: 28 Mach-O, 19 con minimo macOS 26 e nove con minimo 14; SBOM upstream Qt/Homebrew individuate, non SBOM prodotto. Nessun supporto macOS precedente dimostrato, nessun pacchetto Windows/Linux pronto. Licenze/notices/sorgenti e prova su macchina pulita restano aperti.
- [Correzione Qt Cocoa](DOCS/it/ADR-006-QT-COCOA.md): SIGSEGV riprodotto con Qt 6.11.2 originale dopo selezione/filtro/cambio lingua e lettura AX. Due guardie della proposta upstream 765434 patch set 1, revisione c7fd3f34b997bb363be15650647665b3b6b8a5f4, ancora NEW. Plugin ricompilato isolatamente da sorgente ufficiale verificato per SHA-256; Qt installato invariato. Versioni diverse e hash errati rifiutati. Applicazione dopo macdeployqt, collegamenti al bundle, firma ad hoc. Fonti, attribuzione e LGPL separate dalla licenza WebFence. Sequenza critica passata nelle copie 150%/200% e nel bundle standard; non risolve da sola le celle AX intermittenti né certifica VoiceOver. Binari non confezionati usano ancora Qt originale.
- Layout corretto: margini ridotti, minimo due righe complete con spazio scrollbar, splitter non collassabile. Prova visiva 200% IT/EN, ultima colonna raggiunta da tastiera; self-test finestra logica 756×430 superato a scale 1/1,5/2 e vet desktop superato. I valori Qt per copia con Delete esplicita richiedono disattivazione preventiva del finalizer; una doppia liberazione durante lo sviluppo è stata intercettata e corretta prima del commit.
- Ultimi commit: `9655816` correzione Cocoa; CI run 35872886041 fallita solo nella compilazione macOS per header MoltenVK assenti dal runner. `953ed2d` aggiunge prerequisiti MoltenVK/Vulkan e layout compatto; build locale completa riuscita, [CI run 35873709410](https://github.com/Matte2599/WebFence/actions/runs/35873709410) completata con successo sui quattro target, inclusi nuovo self-test e bundle Cocoa. CMake/Ninja e header sono prerequisiti della ricompilazione; il packaging può riusare `WEBFENCE_QT_SOURCE_ARCHIVE`, sempre controllato per hash. Bundle contiene solo Cocoa: non forzare offscreen né plugin Homebrew (mancanza plugin/doppia copia Qt); self-test offscreen sul binario non confezionato.
- Canale GitHub Private vulnerability reporting abilitato e verificato il 2026-09-23; nessuna segnalazione inviata. [Dossier legale](DOCS/it/LEGAL-REVIEW.md) e specifica CLA preparati; autore conferma revisione professionale ancora da organizzare. Nessun CLA approvato, firma o modifica LICENSE.

- Packaging Debian/Windows in implementazione: [ADR-007](DOCS/it/ADR-007-PACKAGING.md). `.deb` nativo Debian 12 con dipendenze ELF derivate, runtime separato senza toolchain e purge; ZIP UCRT64 con chiusura import DLL, avvisi e test PATH Windows. Prime prove XCB richiedono window manager e attesa attivazione nel self-test; normale GUI invariata. Prova Debian ARM64 locale superata (installazione, offscreen/XCB/Openbox, purge). CI `ea2896a` run 35876057107: Debian amd64/arm64 e build native macOS/Linux superati; Windows build/self-test superati, raccolta notices fermata sul percorso ICU, ora corretta e da riverificare. Installed-Size aggiunto; purge con preferenza creata da utente non privilegiato.

### Prossimo lavoro e confini

1. Completare M0 secondo la [matrice](DOCS/it/M0-VALIDATION.md), con prove e collaudo complessivo finale, docs IT/EN, commit/push e CI verificata. Non chiudere per i soli test automatizzati.
2. Risolvere/verificare albero assistivo Qt con lettori reali, focus copia, più monitor e sessione misurata di almeno 30 minuti. Prove visuali già svolte non sostituiscono questi gate.
3. Completare packaging Windows/Linux, dipendenze/notices/sorgenti, minimi OS e macchine pulite. Domande già inviate: disponibilità Windows/NVDA e Debian/Orca; minimo macOS 13/15/26. Nessuna risposta registrata, non inferire consenso.
4. Revisione legale e accordo esplicito prima dei contributi sostanziali da rilicenziare commercialmente. Canale privato già operativo.
5. Nessuno scanner, storage progetti, trasporto di produzione, CVE, report firmati nella GUI o AI implementato; fondazioni non sono integrazione. Nessuna release supportata o equivalenza a scanner commerciali. Maturità MIQT, obblighi Qt e Windows 10 oltre Qt 6.12 restano rischi da gestire.

Questioni aperte: minimo macOS/Debian, collaudo assistivo, hardware misurato, provider/modelli, termini commerciali. Non inventare recapiti, prezzi, benchmark o date di rilascio.

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

- Go 1.27.1, Qt Widgets/MIQT 0.14.0 main GUI (`cmd/webfence`, `internal/desktop`). Fyne and nested Qt module removed from the build, retained in history. Offline prototype: 10,000 fixtures, filters, evidence reader/copy, simple/advanced modes, IT/EN, menus and shortcuts. Only `WebFence/ui-language` under `os.UserConfigDir()` persists, with visible error handling; Fyne preferences are neither imported nor deleted.
- Keyboard: Ctrl/Cmd+F, F6, Ctrl/Cmd+Shift+E/D, Ctrl/Cmd+1/2; translation without model reset or loss of text selection. Qt self-test covers focus, Tab/Shift+Tab, data cache and accessible interface. Qt 6.4 offscreen needs an explicit cache reset in the test: not an OS bridge trial. Native macOS Tab skips copy with current preferences; AX cells/table remain intermittent. No OS preference changes to make tests pass.
- [M0 scope](DOCS/en/M0-SCOPE.md): exact HTTP(S) origins, strict parser, HTTP lab only in tests. [Transport](DOCS/en/ADR-003-TRANSPORT.md): loopback only with IP grants, per-connection/hop DNS, TLS for the original name, atomic budgets, timeouts, cancellation and body/header limits. Race, redirect, excluded destinations, invalid certificates, IPv6 and fuzz checks passed; CI `8f189eb`, run 35861040736. No GUI networking. CIDR/real networks, methods/paths, rate limiting, redaction and complete authorization remain M1.
- [SQLite/JWS](DOCS/en/ADR-004-STORAGE-SIGNATURE.md): mattn/go-sqlite3 v1.14.52 (SQLite 3.53.4), jwx/v4 v4.5.0. SQLite foundation tests only (transactions, reopen, constraints, locks, cancellation); compact Ed25519 JWS component with externally trusted key, bounds and strict headers. Four-target CI `f1d2a84`, run 35863431196; race/fuzz/modules/vet passed. `govulncheck` v1.8.0 with tests found no known Go vulnerabilities; does not cover all native libraries.
- [Credentials](DOCS/en/ADR-005-CREDENTIALS.md): M0 selection complete, `internal/credentials` with fixed namespace, lowercase ASCII IDs, 1–2048 byte binary secrets, redacted errors, no fallback. macOS classic Security.framework: serialized/restored global UI state, deprecated APIs to reassess for signed distribution. Windows wincred v1.2.3; Linux godbus v5.2.2 with local Secret Service, private connection, three-second timeout, no activation/prompts/unlocking. Native macOS/Windows APIs cannot be interrupted after entry; do not call on the Qt thread. GNOME returns text/plain MIME even for binary data: preserve opaque bytes. Native CRUD, locked/absent temporary macOS keychain, locked/absent private Linux Secret Service and Windows anonymous-token trials passed in CI `7c4889d`, run 35870058803, all four targets green. Not proof of screen-lock denial, key lifecycle or GUI integration.
- [Packaging](DOCS/en/M0-PACKAGING.md): macOS uses temporary staging and publication restoring the previous bundle on failure. Build/codesign and injected compiler failure preserving the old artifact verified. Ad hoc only, not notarized. Historical local inventory: 28 Mach-O files, 19 minimum macOS 26 and nine minimum 14; upstream Qt/Homebrew SBOMs located, not a product SBOM. Earlier macOS support not established; no ready Windows/Linux packages. Licenses/notices/sources and clean-machine tests remain open.
- [Qt Cocoa correction](DOCS/en/ADR-006-QT-COCOA.md): original Qt 6.11.2 reproduced SIGSEGV after selection/filter/language change and AX reading. Two guards from upstream proposal 765434 patch set 1, revision c7fd3f34b997bb363be15650647665b3b6b8a5f4, still NEW. Plugin rebuilt in isolation from SHA-256-verified official source; installed Qt unchanged. Different versions and wrong hashes rejected. Applied after macdeployqt, links target bundled frameworks, ad hoc signing. Sources, attribution and LGPL separate from WebFence licensing. Critical sequence passed in 150%/200% copies and standard bundle; does not itself fix intermittent AX cells or certify VoiceOver. Unpackaged binaries still use original Qt.
- Layout corrected: reduced margins, minimum two full rows plus scrollbar space, non-collapsible splitter. Visual 200% IT/EN trial, last column reached by keyboard; logical 756×430 window self-test passed at scales 1/1.5/2 and desktop vet passed. By-value Qt objects with explicit Delete need their finalizer disabled first; a double release during development was caught and fixed before commit.
- Latest commits: `9655816` Cocoa correction; CI run 35872886041 failed only during macOS compilation due to MoltenVK headers missing from the runner. `953ed2d` adds MoltenVK/Vulkan prerequisites and compact layout; complete local build succeeded, [CI run 35873709410](https://github.com/Matte2599/WebFence/actions/runs/35873709410) completed successfully on all four targets, including the new self-test and Cocoa bundle. CMake/Ninja and headers are rebuild prerequisites; packaging can reuse `WEBFENCE_QT_SOURCE_ARCHIVE`, always hash-checked. Bundle includes only Cocoa: do not force offscreen or Homebrew plugins (missing plugin/duplicate Qt); run offscreen self-tests on the unpackaged binary.
- GitHub Private vulnerability reporting enabled and verified on 2026-09-23; no report sent. [Legal dossier](DOCS/en/LEGAL-REVIEW.md) and CLA specification prepared; author confirms professional review still needs arranging. No approved CLA, signature or LICENSE change.

- Debian/Windows packaging under implementation: [ADR-007](DOCS/en/ADR-007-PACKAGING.md). Native Debian 12 `.deb` with derived ELF dependencies, separate runtime without toolchain and purge; UCRT64 ZIP with DLL import closure, notices and Windows-only PATH test. Initial XCB trials require a window manager and activation wait in the self-test; normal GUI unchanged. Local Debian ARM64 trial passed (install, offscreen/XCB/Openbox, purge). CI `ea2896a` run 35876057107: Debian amd64/arm64 and native macOS/Linux passed; Windows build/self-test passed, notices collection stopped on the ICU path, now corrected and awaiting verification. Installed-Size added; purge preference created by an unprivileged user.

### Next work and boundaries

1. Complete M0 according to the [matrix](DOCS/en/M0-VALIDATION.md), with evidence and final comprehensive trial, IT/EN docs, commit/push and verified CI. Do not close based only on automated tests.
2. Resolve/verify Qt assistive tree with actual readers, copy focus, multiple monitors and a measured session of at least 30 minutes. Existing visual checks do not replace these gates.
3. Complete Windows/Linux packaging, dependency notices/sources, OS minima and clean machines. Questions already sent: Windows/NVDA and Debian/Orca availability; macOS minimum 13/15/26. No recorded answer; do not infer consent.
4. Legal review and explicit agreement before substantial contributions intended for commercial relicensing. Private channel is already operational.
5. No scanner, project store, production transport, CVE, GUI report signing or AI implementation; foundations are not integration. No supported release or commercial-scanner equivalence. MIQT maturity, Qt obligations and Windows 10 beyond Qt 6.12 remain risks to manage.

Open issues: macOS/Debian minimum, assistive trials, measured hardware, providers/models, commercial terms. Do not invent contacts, prices, benchmarks or release dates.
