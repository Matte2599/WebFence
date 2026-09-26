# Roadmap WebFence

[Indice / Index](README.md) · [Italiano](#italiano) · [English](#english)

Aggiornamento / Updated: 2026-09-26. Responsabile / Owner: Matteo Luigi Feroldi.

## Italiano

Questa roadmap è basata su dipendenze e criteri di uscita, senza date di rilascio promesse. Esistono la fondazione documentale, il prototipo desktop M0 e le alpha tecniche M1/M2; stato e limiti sono nelle matrici [M0](it/M0-VALIDATION.md), [M1](it/M1-VALIDATION.md) e [M2](it/M2-VALIDATION.md). Lo sviluppo di un prodotto paragonabile per profondità a scanner commerciali richiede iterazioni e benchmark; non è una singola milestone.

### Fondazione documentale — completata

- [x] Visione, requisiti WF-01–WF-10 e scelta desktop nativo.
- [x] Raccomandazione Go, confronto Rust e scelta Qt Widgets/MIQT dopo la prova Fyne.
- [x] Licenza source-available con professionisti indipendenti ammessi e uso aziendale riservato.
- [x] Architettura, minacce, firme, AI, CVE, conservazione e retest descritti in IT/EN.
- [x] Memoria di progetto e istruzioni per agenti/contributori.

Queste spunte attestano documenti creati, non codice funzionante o revisione legale conclusa.

### M0 — Fattibilità desktop e fondazioni del codice

Dipende dalla fondazione documentale. Ambito: prototipo desktop nativo offline e bilingue, packaging di sviluppo sui target richiesti, scelte Qt/SQLite/JWS/portachiavi, cataloghi e CI, laboratorio sintetico di scope/rete, canale privato di segnalazione. Criteri, prove, chiusura automatizzabile e prerequisiti umani per M1 sono documentati esclusivamente nella [matrice M0 IT](it/M0-VALIDATION.md) e nella [matrice M0 EN](en/M0-VALIDATION.md).

### M1 — Alpha tradizionale controllata

Dipende da M0. Copre WF-01, WF-02, WF-03, WF-07, WF-09.

I gate con intervento umano trasferiti dalla [matrice M0](it/M0-VALIDATION.md) hanno un [piano di collaudo](it/M1-PREREQUISITES.md) e un [preflight locale](evidence/m1-prerequisite-preflight-2026-09-24.md). Per decisione successiva dell'autore, non bloccano più la chiusura tecnica di M1: rimangono prove da svolgere prima delle pertinenti dichiarazioni di supporto, accessibilità e distribuzione.

Primo blocco tecnico: [modello progetto e dichiarazione di autorizzazione](it/M1-PROJECT-AUTHORIZATION.md) in memoria, senza rete. Non completa i criteri M1 né i prerequisiti umani della [matrice M0](it/M0-VALIDATION.md).

Secondo blocco tecnico: [snapshot applicato al laboratorio HTTP](it/M1-AUTHORIZED-LAB.md) su ogni hop e alla scadenza, ancora solo loopback. Policy complete e trasporto di produzione restano da realizzare.

Terzo blocco tecnico: [store SQLite v1 dei progetti](it/M1-PROJECT-STORE.md), con creazione, riapertura, elenco e cancellazione dei soli metadati. Quota disco, backup e integrazione desktop restano aperti.

Quarto blocco tecnico: [revisioni e rinnovo dell'autorizzazione](it/M1-PROJECT-STORE.md) con migrazione SQLite v1→v2, cronologia immutabile e controllo di versione concorrente.

Quinto blocco tecnico: [run gestite e revoca locale](it/M1-MANAGED-RUNS.md) su rinnovo, revoca esplicita persistente, cancellazione, chiusura e scadenza; migrazione SQLite v2→v3. Il laboratorio HTTP interrompe le richieste in corso. Coordinamento tra processi e integrazione desktop restano aperti.

Sesto blocco tecnico: [primo controllo HTTP su seed espliciti](it/M1-HEADER-LAB.md), collegando progetto salvato, run gestita e broker solo loopback; osserva un header senza conservare URL/body/header grezzi nei risultati. Non realizza ancora discovery, persistenza degli esiti o UI.

Settimo blocco tecnico: [discovery HTML osservativa](it/M1-DISCOVERY-LAB.md) sulle risposte dei seed; estrae link e azioni di form sotto scope e limiti, senza nuove richieste. Il report espone conteggi e copertura parziale.

Ottavo blocco tecnico: [visite HTTP controllate](it/M1-CONTROLLED-CRAWL.md) con policy GET/HEAD, prefissi ed esclusioni, coda BFS opt-in, rate per origine e [IP pubblici fissati](it/ADR-008-PINNED-PUBLIC-TRANSPORT.md). Verifica sintetica; nessun target esterno reale.

**M1 chiusa come alpha tecnica controllata** secondo la [matrice di validazione](it/M1-VALIDATION.md). I quattro criteri sotto sono implementati e verificati; le [piattaforme target](it/M1-SUPPORT-POLICY.md) sono state fissate e la [revisione legale interna](it/M1-LEGAL-ASSESSMENT.md) è documentata. Le prove M1-H01…H08 non eseguite non sono state trasformate in esiti positivi: passano a condizioni di qualità/rilascio o contributi per le attività pertinenti.

- [x] Progetti, scope/autorizzazione, trasporto controllato, budget e cancellazione.
- [x] Discovery HTTP(S), primo controllo a basso impatto e prove redatte.
- [x] Persistenza, ripristino di run interrotte, limiti disco e cancellazione base del progetto.
- [x] Desktop IT/EN con avanzamento, risultati e copertura incompleta esplicita.

Uscita: flusso progetto → scansione lab → risultati → riavvio verificato; suite di scope senza violazioni, casi vulnerabili/corretti per ogni regola, nessun LLM richiesto. Alpha utilizzabile soltanto nei limiti documentati.

### M2 — Intelligence e report verificabili

Dipende da M1. Copre WF-04 e WF-06.

Primo blocco core: [cache CVE/NVD](it/M2-INTELLIGENCE-CACHE.md) con sync transazionale, provenienza e stato.

Secondo blocco core: [matching conservativo](it/M2-MATCHING.md) su segnali espliciti, con confidenza, intervalli di versione e backport attestati.

Terzo blocco core: [report verificabili](it/M2-REPORTS.md) da run salvate, con JSON/HTML IT/EN, manifest JCS, firma JWS, gestione locale delle chiavi e verifica offline.

Quarto blocco: [flusso desktop M2 e verifica complessiva](it/M2-VALIDATION.md), con sync manuale, segnale prodotto/versione dichiarato dall'operatore, export dalla run selezionata e regressioni native.

Verifica extra successiva alla chiusura M2: [simulazione locale del matching](it/M2-ACCURACY-SIMULATION.md) su un corpus sintetico etichettato e rafforzamento delle condizioni che richiedono `unknown`. La misura non equivale a un benchmark su CVE reali.

Estensione desktop successiva: [funzioni avanzate M2 nella GUI](it/M2-GUI-EXTENSION.md) per identità CPE, attestazioni e gestione delle chiavi pubbliche, con regressioni del flusso e ispezione grafica Cocoa/offscreen. Restano aperte accuratezza reale, persistenza delle correlazioni e prove assistive/hardware pertinenti.

- [x] Adattatori CVE/NVD, cache incrementale, provenienza e stato di aggiornamento.
- [x] Matching con confidenza, versioni/backport e distinzione tra candidato e verificato.
- [x] Export JSON/HTML IT/EN, manifest, firma, gestione chiavi e verificatore offline.
- [x] Prove di manomissione, chiave non fidata, feed indisponibile e sync interrotta.

Uscita: bundle originale verificabile e alterazioni respinte; ultima cache valida recuperabile; report con controlli non eseguiti visibili. Nessuna dichiarazione di firma qualificata o copertura CVE totale.

### M3 — Applicazioni dinamiche e autenticate

Dipende da M2. Estende WF-03 e realizza WF-10.

Primo blocco core: [gate delle richieste browser](it/M3-BROWSER-GATE.md) per scope, policy, budget e revoca, senza runtime browser o traffico target. I criteri sotto restano aperti.

- [ ] Browser isolato, scope anche su subresource/redirect e limiti di processo.
- [ ] Sessioni e flussi di login per account di test, verifica validità e separazione identità.
- [ ] Import OpenAPI, crawling dinamico, controlli contestuali e accessi tra ruoli.
- [ ] Espansione delle famiglie di regole soltanto con fixture e misure pubblicate.

Uscita: SPA e API del corpus coperte nelle parti dichiarate, traffico fuori scope bloccato, credenziali non inoltrate a terzi, nessun errore di login presentato come esito negativo del test.

### M4 — Convalida fix e regressioni: MVP funzionale

Dipende da M3. Completa WF-07 e WF-08.

- [ ] Baseline immutabili, fingerprint versionati e confronto della copertura.
- [ ] Retest mirati con prerequisiti verificati e stati `fixed_verified` / `inconclusive` distinti.
- [ ] Ricerca di regressioni nell'area riesaminata, separata dai problemi osservati per la prima volta.
- [ ] Cancellazione completa dei dati gestiti dall'app e prova di ripristino backup.

Uscita: fixture vulnerabile → fix → retest → regressione riconosciuta; nessun falso «risolto» nei casi ambigui del corpus. Questo è il primo MVP del ciclo completo, ancora senza AI obbligatoria.

### M5 — AI opzionale remota e locale

Dipende da M4 e dalla baseline di qualità. Realizza WF-05.

- [ ] Contratto provider, output strutturato, redazione e consenso remoto per progetto.
- [ ] Almeno un adattatore remoto e uno locale scelti con benchmark, non per il solo nome del modello.
- [ ] Pacchetti di modelli opzionali, diritti verificati, hash, preflight risorse e budget.
- [ ] Corpus di prompt injection, citazioni, errori, OOM e confronto AI attiva/disattiva.

Uscita: benefici misurati e costi dichiarati; AI disabilitabile; nessun fallback cloud implicito o ampliamento di scope. I requisiti hardware pubblicati derivano da prove effettive.

### M6 — Beta pubblica e rilascio stabile

Dipende da M1–M5 e dai gate di [qualità](it/QUALITY.md).

- [ ] PDF con QA visiva, localizzazione completa e documentazione di installazione verificata.
- [ ] Installer firmati dove supportato, SBOM, licenze terze parti e aggiornamenti verificati.
- [ ] Prove end-to-end su tutte le piattaforme supportate, accessibilità e recupero dati.
- [ ] Canale di sicurezza operativo, gestione issue, note di rilascio e contratti commerciali pronti per gli usi offerti.
- [ ] Report pubblico dei benchmark con limiti e nessuna metrica inventata.

Uscita: tutti i gate applicabili superati e rischi residui documentati. «Stabile» non significa assenza di vulnerabilità.

### Successivamente

CLI/CI, import SBOM, arricchimenti KEV/EPSS, verifiche su specifici framework, sandbox per plugin e modalità team/server: priorità da assegnare in base a utenti e misure. SaaS multi-tenant, SAST completo e correzione automatica non sono impegni della prima roadmap.

## English

This roadmap uses dependencies and exit criteria, without promised release dates. The documentation foundation, M0 desktop prototype and technical M1/M2 alphas exist; status and limits are in the [M0](en/M0-VALIDATION.md), [M1](en/M1-VALIDATION.md) and [M2](en/M2-VALIDATION.md) matrices. Building coverage comparable in depth to commercial scanners requires iterations and benchmarks, not one milestone.

### Documentation foundation — complete

- [x] Vision, WF-01–WF-10 requirements and native desktop decision.
- [x] Go recommendation, Rust comparison and Qt Widgets/MIQT choice after the Fyne trial.
- [x] Source-available license allowing independent professionals and reserving company use.
- [x] IT/EN architecture, threats, signatures, AI, CVE, retention and retest specifications.
- [x] Project memory and agent/contributor instructions.

These checks represent created documents, not working code or completed legal review.

### M0 — Desktop feasibility and code foundations

Depends on the documentation foundation. Scope: bilingual offline native desktop prototype, development packaging for requested targets, Qt/SQLite/JWS/credential choices, catalogs and CI, synthetic scope/network lab, and private vulnerability reporting. Criteria, evidence, automatable closure and human prerequisites for M1 are documented only in the [M0 EN matrix](en/M0-VALIDATION.md) and [M0 IT matrix](it/M0-VALIDATION.md).

### M1 — Controlled traditional alpha

Depends on M0. Covers WF-01, WF-02, WF-03, WF-07, WF-09.

The human-intervention gates carried from the [M0 matrix](en/M0-VALIDATION.md) have a [trial plan](en/M1-PREREQUISITES.md) and [local preflight](evidence/m1-prerequisite-preflight-2026-09-24.md). By a later author decision they no longer block technical M1 closure; they remain to be performed before the relevant support, accessibility and distribution claims.

First technical block: [in-memory project model and authorization declaration](en/M1-PROJECT-AUTHORIZATION.md), without networking. It does not complete the M1 criteria or the human prerequisites in the [M0 matrix](en/M0-VALIDATION.md).

Second technical block: [snapshot enforced in the HTTP lab](en/M1-AUTHORIZED-LAB.md) on every hop and at expiry, still loopback-only. Full policies and production transport remain to be built.

Third technical block: [SQLite v1 project store](en/M1-PROJECT-STORE.md), with creation, reopening, listing and deletion of metadata only. Disk quotas, backup and desktop integration remain open.

Fourth technical block: [authorization revisions and renewal](en/M1-PROJECT-STORE.md) with SQLite v1→v2 migration, immutable history and optimistic revision checks.

Fifth technical block: [managed runs and local revocation](en/M1-MANAGED-RUNS.md) on renewal, persistent explicit revocation, deletion, close and expiry; SQLite v2→v3 migration. The HTTP lab interrupts in-flight requests. Cross-process coordination and desktop integration remain open.

Sixth technical block: [first HTTP check on explicit seeds](en/M1-HEADER-LAB.md), connecting saved project, managed run and loopback-only broker; it observes one header without retaining raw URLs/bodies/headers in results. Discovery, result persistence and UI are still missing.

Seventh technical block: [observational HTML discovery](en/M1-DISCOVERY-LAB.md) on seed responses; it extracts links and form actions under scope and parser limits, without new requests. The report exposes counts and partial coverage.

Eighth technical block: [controlled HTTP visits](en/M1-CONTROLLED-CRAWL.md) with GET/HEAD policy, path prefixes/exclusions, opt-in BFS queue, per-origin pacing and [pinned public IPs](en/ADR-008-PINNED-PUBLIC-TRANSPORT.md). Synthetic verification; no real external target.

**M1 is closed as a controlled technical alpha** according to the [validation matrix](en/M1-VALIDATION.md). The four criteria below are implemented and verified; [target platforms](en/M1-SUPPORT-POLICY.md) are set and the [internal legal assessment](en/M1-LEGAL-ASSESSMENT.md) is recorded. Unperformed M1-H01…H08 trials have not been turned into passing results: they move to quality/release or contribution conditions for relevant future activity.

- [x] Projects, scope/authorization, controlled transport, budgets and cancellation.
- [x] HTTP(S) discovery, first low-impact check and redacted evidence.
- [x] Persistence, interrupted-run recovery, disk limits and basic project deletion.
- [x] IT/EN desktop with progress, results and explicit incomplete coverage.

Exit: verified project → lab scan → results → restart workflow; scope suite without violations, vulnerable/fixed cases per rule, no LLM required. Alpha usable only within documented limits.

### M2 — Intelligence and verifiable reports

Depends on M1. Covers WF-04 and WF-06.

First core block: [CVE/NVD cache](en/M2-INTELLIGENCE-CACHE.md) with transactional sync, provenance and status.

Second core block: [conservative matching](en/M2-MATCHING.md) on explicit signals, with confidence, version ranges and attested backports.

Third core block: [verifiable reports](en/M2-REPORTS.md) from saved runs, with IT/EN JSON/HTML, a JCS manifest, JWS signing, local key management and offline verification.

Fourth block: [M2 desktop flow and full validation](en/M2-VALIDATION.md), with manual sync, operator-declared product/version signal, export from the selected run and native regressions.

Extra review after M2 closure: [local matching simulation](en/M2-ACCURACY-SIMULATION.md) on a labeled synthetic corpus and tighter conditions requiring `unknown`. This measure is not a benchmark on real CVEs.

Later desktop extension: [advanced M2 features in the GUI](en/M2-GUI-EXTENSION.md) for CPE identity, attestations and public key management, with flow regressions and Cocoa/offscreen visual inspection. Real-world accuracy, assessment persistence and relevant assistive/hardware trials remain open.

- [x] CVE/NVD adapters, incremental cache, provenance and freshness status.
- [x] Confidence-aware matching, versions/backports and candidate/verified distinction.
- [x] IT/EN JSON/HTML export, manifests, signing, key management and offline verifier.
- [x] Tampering, untrusted-key, unavailable-feed and interrupted-sync tests.

Exit: original bundle verifies and alterations are rejected; last valid cache is recoverable; reports show unexecuted checks. No qualified-signature or complete-CVE-coverage claims.

### M3 — Dynamic and authenticated applications

Depends on M2. Extends WF-03 and implements WF-10.

First core block: [browser request gate](en/M3-BROWSER-GATE.md) for scope, policy, budgets and revocation, without a browser runtime or target traffic. The criteria below remain open.

- [ ] Isolated browser, scope for subresources/redirects and process limits.
- [ ] Test-account sessions/login flows, validity checks and identity separation.
- [ ] OpenAPI import, dynamic crawling, contextual checks and cross-role access testing.
- [ ] Additional rule families only with fixtures and published measurements.

Exit: declared corpus SPA/API surfaces covered, out-of-scope traffic blocked, no credentials forwarded to third parties, no login error presented as a negative test result.

### M4 — Fix validation and regressions: functional MVP

Depends on M3. Completes WF-07 and WF-08.

- [ ] Immutable baselines, versioned fingerprints and coverage comparison.
- [ ] Targeted retests with verified prerequisites and distinct `fixed_verified` / `inconclusive` states.
- [ ] Regression discovery in reassessed areas, separate from first-observed findings.
- [ ] Complete app-managed data deletion and backup recovery testing.

Exit: vulnerable fixture → fix → retest → detected regression; no false “fixed” results in ambiguous corpus cases. This is the first full-cycle MVP, still without mandatory AI.

### M5 — Optional remote and local AI

Depends on M4 and the quality baseline. Implements WF-05.

- [ ] Provider contract, structured output, redaction and per-project remote consent.
- [ ] At least one remote and one local adapter selected through benchmarks, not model branding alone.
- [ ] Optional model packages, reviewed rights, hashes, resource preflight and budgets.
- [ ] Prompt-injection, citation, error and OOM corpus; AI-enabled/disabled comparison.

Exit: measured benefits and disclosed costs; AI can be disabled; no implicit cloud fallback or scope expansion. Published hardware requirements come from real tests.

### M6 — Public beta and stable release

Depends on M1–M5 and [quality](en/QUALITY.md) gates.

- [ ] PDF with visual QA, complete localization and verified installation documentation.
- [ ] Signed installers where supported, SBOM, third-party licenses and verified updates.
- [ ] End-to-end tests on every supported platform, accessibility and data recovery.
- [ ] Operational security channel, issue handling, release notes and commercial contracts for offered uses.
- [ ] Public benchmark report with limitations and no invented metrics.

Exit: all applicable gates pass and remaining risks are documented. “Stable” does not mean vulnerability-free.

### Later

CLI/CI, SBOM import, KEV/EPSS enrichment, framework-specific checks, plugin sandboxing and team/server mode: prioritize using user needs and measurements. Multi-tenant SaaS, complete SAST and automatic remediation are not commitments in this initial roadmap.
