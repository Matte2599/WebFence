# Memoria del progetto / Project memory

Aggiornato / Updated: 2026-09-25. Questa memoria conserva decisioni e problemi aperti; prove e stato sono nelle matrici [M0 IT](DOCS/it/M0-VALIDATION.md) / [EN](DOCS/en/M0-VALIDATION.md) e [M1 IT](DOCS/it/M1-VALIDATION.md) / [EN](DOCS/en/M1-VALIDATION.md).

## Italiano

### Requisiti confermati dall'autore

- Autore: **Matteo Luigi Feroldi**; repository [Matte2599/WebFence](https://github.com/Matte2599/WebFence). Applicazione desktop nativa e documentazione IT/EN; futuri report IT/EN.
- Sistemi richiesti: macOS solo Apple Silicon; Windows 10/11 x86-64; Debian e derivate x86-64/ARM64. macOS **26** è il minimo temporaneo scelto dall'autore; minimi Windows/Debian non ancora deliberati. UX sobria e familiare come strumenti Windows 2015–2020, semplice per principianti e dettagliata per esperti.
- Go e Qt Widgets/MIQT sono scelte dell'autore dopo confronto con Rust/Fyne ([ADR-001 IT](DOCS/it/ADR-001-LANGUAGE.md), [ADR-002 IT](DOCS/it/ADR-002-GUI.md)). Cambi architetturali richiedono un ADR.
- **WebFence Community License 1.0** è source-available, non open source OSI: persone e professionisti indipendenti possono usarla gratuitamente anche per analisi retribuite; aziende richiedono autorizzazione anche per uso interno. Rivendita/distribuzione commerciale del software richiede accordo dell'autore, eventualmente con royalties. Revisione legale ancora da organizzare.
- Obiettivo di lungo periodo: analisi tradizionale e AI opzionale, CVE aggiornate, report firmati, progetti persistenti, convalida fix e regressioni. Invicti/Acunetix sono riferimenti di profondità desiderata, non equivalenza dimostrata.
- Flusso di lavoro richiesto: branch dedicato per blocco, CI verde, squash merge su `main`, verifica CI del commit finale. Non creare commit solo per registrare CI. Aggiornare questa memoria alla fine del blocco; le istruzioni correnti dell'autore prevalgono.

### Decisioni tecniche da conservare

- Il laboratorio M0 di 10.000 fixture resta separato dalla scansione. La [alpha tecnica M1](DOCS/it/M1-VALIDATION.md) collega progetto dichiarato/autorizzato, run gestita, broker HTTP(S) controllato, discovery HTML osservativa, controllo `HTTP-XCTO-001`, ledger redatto e GUI IT/EN. È un primo controllo di header, non un audit generale. Nessun LLM è richiesto. La GUI M1 supporta una origine e un seed per scansione; il core ne supporta più entro limiti espliciti.
- Richiedere consenso dichiarato, origine esatta, policy GET/percorso, grant IP esatti, budget e cadenza; seguire i link solo con opt-in, mai inviare form. La modalità pubblica richiede IP pubblici fissati manualmente ([ADR-008 IT](DOCS/it/ADR-008-PINNED-PUBLIC-TRANSPORT.md)); i test non hanno contattato target esterni. Trattare pagine, DNS e futuri output AI come dati non fidati. La dichiarazione dell'operatore non prova la sua autorizzazione.
- SQLite **v4** salva dichiarazioni/revisioni e solo codici/conteggi di run/visite, senza URL visitati, query, header, body o credenziali ([store IT](DOCS/it/M1-PROJECT-STORE.md)). Lock OS esclusivo sul file, una run crawler persistente per istanza, marcatura `interrupted` al riavvio, limite di 64 MiB sul file principale, 100 run per progetto e 256 visite per run, backup privato in file nuovo, eliminazione logica a cascata. WAL/backup esterni non sono cancellati in modo forense. Il riferimento di autorizzazione e le origini restano metadati in chiaro; non inserirvi segreti.
- JWS Ed25519, portachiavi nativi e packaging di sviluppo esistono come fondazioni separate, non come report firmati o release supportata. Il bundle macOS applica la correzione Qt Cocoa ([ADR-006 IT](DOCS/it/ADR-006-QT-COCOA.md)); Windows CI usa il lock dei pacchetti MSYS2 ([guida IT](DOCS/it/DEVELOPMENT.md)).
- Non commettere segreti, chiavi private, dati clienti, pesi, database CVE o report reali. Non eseguire scansioni esterne senza target e attività esplicitamente autorizzati nella richiesta.

### Problemi aperti

- La **chiusura formale M1** richiede i gate umani [M1-H01…H08](DOCS/it/M1-PREREQUISITES.md): VoiceOver/NVDA/Orca ascoltati da persone, monitor con scale diverse, workstation Windows 10/11 e Debian x86-64/ARM64 reali, versioni minime, revisione legale e accordo contributori. L'autore può provare un PC Windows 10; build/versione ed esito non sono ancora disponibili. Non risultano ascoltatori per i lettori di schermo. Il Mac locale è Apple Silicon/macOS 26.6.2, non la versione minima 26.0, e ha un solo display.
- Il broker pubblico non è stato provato su rete pubblica reale. Archivi separati non condividono un limite globale; manca feedback sul carico del target e un firewall egress indipendente. La policy non interpreta query né garantisce che GET sia senza effetti. La GUI non offre ancora rinnovo/revoca, backup o ripristino guidato.
- CVE, export/report firmati, browser/autenticazione, retest e AI appartengono alle milestone successive; non presentarli come implementati. Contributi esterni non concedono automaticamente diritti di rilicenza ([CONTRIBUTING](CONTRIBUTING.md)).

## English

### Author-confirmed requirements

- Author: **Matteo Luigi Feroldi**; repository [Matte2599/WebFence](https://github.com/Matte2599/WebFence). Native desktop application and IT/EN documentation; future IT/EN reports.
- Requested systems: Apple Silicon macOS only; Windows 10/11 x86-64; Debian and derivatives x86-64/ARM64. macOS **26** is the author's temporary minimum; Windows/Debian minima remain undecided. Restrained, familiar 2015–2020 Windows-tool UX, accessible to novices with detail for experts.
- Go and Qt Widgets/MIQT are author-selected after Rust/Fyne comparison ([ADR-001 EN](DOCS/en/ADR-001-LANGUAGE.md), [ADR-002 EN](DOCS/en/ADR-002-GUI.md)). Architecture changes require an ADR.
- **WebFence Community License 1.0** is source-available, not OSI open source: individuals and independent professionals may use it free of charge, including for paid assessments; companies need authorization even for internal use. Software resale/commercial distribution needs the author's agreement, potentially with royalties. Legal review has not been arranged.
- Long-term goal: traditional and optional AI analysis, updated CVEs, signed reports, persistent projects, fix validation and regressions. Invicti/Acunetix are desired depth references, not established equivalence.
- Requested workflow: dedicated branch per block, green CI, squash merge into `main`, verify CI for the final commit. Do not make CI-status-only commits. Update this memory at block end; current author instructions prevail.

### Technical decisions to preserve

- The 10,000-fixture M0 lab stays separate from scanning. The [technical M1 alpha](DOCS/en/M1-VALIDATION.md) connects an operator-declared/authorized project, managed run, controlled HTTP(S) broker, observational HTML discovery, `HTTP-XCTO-001` check, redacted ledger and IT/EN GUI. This is one initial header check, not a general audit. No LLM is required. The M1 GUI supports one origin and seed per scan; the core supports more within explicit bounds.
- Require declared consent, exact origin, GET/path policy, exact IP grants, budgets and pacing; follow links only with opt-in and never submit forms. Public mode needs manually pinned public IPs ([ADR-008 EN](DOCS/en/ADR-008-PINNED-PUBLIC-TRANSPORT.md)); tests contacted no external target. Treat pages, DNS and future AI output as untrusted. An operator declaration does not prove authorization.
- SQLite **v4** stores declarations/revisions and only run/visit codes and counts, without visited URLs, queries, headers, bodies or credentials ([store EN](DOCS/en/M1-PROJECT-STORE.md)). Exclusive OS file lock, one persistent crawler run per instance, `interrupted` marking at restart, 64 MiB main-file cap, 100 runs per project and 256 visits per run, private backup to a new file and cascading logical deletion. WAL/separate backups are not forensically erased. Authorization references and origins remain plaintext metadata; do not enter secrets.
- Ed25519 JWS, native keychains and development packaging exist as separate foundations, not signed reports or supported releases. The macOS bundle applies the Qt Cocoa correction ([ADR-006 EN](DOCS/en/ADR-006-QT-COCOA.md)); Windows CI uses the MSYS2 package lock ([EN guide](DOCS/en/DEVELOPMENT.md)).
- Do not commit secrets, private keys, client data, weights, CVE databases or real reports. Do not scan external systems without targets and activities explicitly authorized in the request.

### Open problems

- **Formal M1 closure** requires human [M1-H01…H08 gates](DOCS/en/M1-PREREQUISITES.md): people listening to VoiceOver/NVDA/Orca, mixed-scale monitors, real Windows 10/11 and Debian x86-64/ARM64 workstations, minimum OS versions, legal review and contributor agreement. The author can test a Windows 10 PC; build/version and outcome are not yet available. No screen-reader listeners are available. The local Mac is Apple Silicon/macOS 26.6.2, not minimum 26.0, and has one display.
- Public transport has not been tested on a real public network. Separate stores do not share an overall limit; target load feedback and independent egress firewall are missing. Policy does not interpret queries or guarantee that GET has no effects. The GUI does not yet offer authorization renewal/revocation, backup or guided restore.
- CVE, signed export/reports, browser/authentication, retests and AI belong to later milestones; do not present them as implemented. External contributions do not automatically grant relicensing rights ([CONTRIBUTING](CONTRIBUTING.md)).
