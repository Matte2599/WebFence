# Memoria del progetto / Project memory

Aggiornato / Updated: 2026-09-24. Memoria delle decisioni persistenti e dei problemi aperti; lo stato di M0 è riportato **solo** nelle matrici [IT](DOCS/it/M0-VALIDATION.md) / [EN](DOCS/en/M0-VALIDATION.md). Le prove storiche dettagliate restano nei documenti collegati e in `DOCS/evidence/`.

## Italiano

### Decisioni confermate dall'autore

- WebFence è di **Matteo Luigi Feroldi**; repository [Matte2599/WebFence](https://github.com/Matte2599/WebFence). È un'applicazione **desktop nativa**, con UI, documentazione e futuri report in italiano e inglese.
- Piattaforme richieste: macOS solo Apple Silicon; Windows 10/11 x86-64; Debian e derivate x86-64/ARM64. UX sobria e familiare come strumenti desktop Windows 2015–2020, con percorso guidato e controlli avanzati.
- Go per core e desktop; l'autore ha scelto **Qt Widgets/MIQT** dopo il confronto con Fyne. [ADR-001](DOCS/it/ADR-001-LANGUAGE.md) e [ADR-002](DOCS/it/ADR-002-GUI.md) registrano motivi e criteri di revisione. Architettura diversa richiede un nuovo ADR.
- Licenza **WebFence Community License 1.0**, personalizzata source-available: persone e professionisti indipendenti possono usare e modificare nei termini del testo, anche per analisi retribuite; le aziende richiedono autorizzazione, anche per uso interno. Distribuzione a scopo di lucro e alle aziende richiede autorizzazione di Matteo Luigi Feroldi; royalties/accordi sono negoziati. Non chiamarla licenza open source OSI né sostituirla senza richiesta dell'autore.
- Obiettivi di prodotto: analisi tradizionale e AI opzionale (modelli remoti/locali), CVE aggiornate, report firmati, progetti persistenti fino alla cancellazione, retest dei fix e regressioni. Invicti/Acunetix sono riferimenti di profondità desiderata, non equivalenza dimostrata. Una firma non certifica che un target sia sicuro.
- **Nuova regola di lavoro:** un branch per blocco; CI verde sul branch, poi squash merge su `main` e verifica CI del commit finale. Nessun commit solo per registrare CI. Stato M0 solo nelle due matrici; README e roadmap vi rimandano. Aggiornare questa memoria solo a fine blocco, con decisioni e problemi aperti. Le istruzioni attuali dell'autore prevalgono.
- **Criterio M0 definito dall'autore:** richiede il completamento di M0-03/04/05 e la chiusura automatizzabile di M0-01/02/06. I gate che richiedono persone o hardware reale vanno indicati **«richiede intervento umano — prerequisito M1»** nelle matrici. L'inventario attuale di sorgenti e notices è sufficiente per M0, senza ulteriori livelli di verifica. Questo non è un parere legale né approvazione alla distribuzione.

### Decisioni tecniche e limiti da conservare

- Il prototipo è offline e usa fixture sintetiche. La GUI Qt carica 10.000 righe, filtra, legge/copia prove, offre IT/EN e salva solo la preferenza di lingua; non esegue scansioni di produzione. Il trasporto di laboratorio è limitato a loopback e soggetto a scope, DNS/IP, TLS, budget, timeout e cancellazione ([ADR-003](DOCS/it/ADR-003-TRANSPORT.md)). Non eseguire scansioni esterne senza target e attività autorizzati.
- SQLite è la scelta per lo store locale futuro; esistono prove di fondazione, non ancora persistenza dei progetti. Il componente JWS Ed25519 richiede una chiave fidata esterna; i report firmati nella GUI restano futuri ([ADR-004](DOCS/it/ADR-004-STORAGE-SIGNATURE.md)). Credenziali native selezionate per macOS/Windows/Secret Service; non archiviare segreti in fallback testuale ([ADR-005](DOCS/it/ADR-005-CREDENTIALS.md)).
- Il bundle macOS applica una patch Qt Cocoa documentata in [ADR-006](DOCS/it/ADR-006-QT-COCOA.md); la verifica AX automatizzata sul codice corrente è distinta dall'uso con VoiceOver reale. Il packaging di sviluppo comprende bundle macOS, ZIP Windows e `.deb` Debian ([ADR-007](DOCS/it/ADR-007-PACKAGING.md)). Sorgenti/notices sono inventariati, ma non equivalgono a un giudizio legale o a un installer supportato.
- Primo blocco tecnico M1: `internal/project` registra in memoria una dichiarazione confermata dall'operatore con proprietario indicato, riferimento, scadenza e origini HTTP(S) esatte. Produce uno snapshot immutabile per run e ricontrolla la scadenza per ogni URL. [Contratto e limiti](DOCS/it/M1-PROJECT-AUTHORIZATION.md). Non prova l'autorizzazione legale, non persiste e non apre rete.
- Rete verso target, browser e AI futuri devono rimanere sotto policy e budget deterministici. Trattare pagine e output AI come dati non fidati. Non inserire nel repository segreti, chiavi private, dati clienti, pesi, database CVE o report reali.

### Problemi aperti per M1

- Collaudi con persone e hardware: annunci e uso con VoiceOver, NVDA e Orca; passaggio tra monitor con scale diverse; installazione e uso su Windows 10/11 e Debian reali; prove sulle versioni minime dei sistemi. Motivi e procedura sono nelle [matrici M0](DOCS/it/M0-VALIDATION.md).
- [Revisione legale](DOCS/it/LEGAL-REVIEW.md) qualificata ancora da organizzare. Accordo contributori specificato ma non approvato: contributi esterni non conferiscono automaticamente diritti di rilicenza. Definire termini e processo prima di aprire la raccolta pertinente.
- Scanner reale, store di progetto, report di prodotto, integrazione credenziali/chiavi, AI e retest sono lavoro delle milestone successive. Distinguere sempre capacità implementate, proposte e verificate.
- Per completare M1 servono ancora persistenza e migrazioni del progetto, versione/rinnovo della dichiarazione, policy di metodi/percorsi/IP, broker di produzione con controlli per hop e budget, UI IT/EN, discovery e prove redatte. Non collegare lo snapshot di sole origini direttamente a traffico reale.

## English

### Author-confirmed decisions

- WebFence is by **Matteo Luigi Feroldi**; repository [Matte2599/WebFence](https://github.com/Matte2599/WebFence). It is a **native desktop application**, with UI, documentation and future reports in Italian and English.
- Requested platforms: Apple Silicon macOS only; Windows 10/11 x86-64; Debian and derivatives x86-64/ARM64. UX should be restrained and familiar like 2015–2020 Windows desktop tools, with guided and advanced paths.
- Go for core and desktop; the author selected **Qt Widgets/MIQT** after comparing Fyne. [ADR-001](DOCS/en/ADR-001-LANGUAGE.md) and [ADR-002](DOCS/en/ADR-002-GUI.md) record reasons and review criteria. A different architecture requires a new ADR.
- **WebFence Community License 1.0** is a custom source-available license: individuals and independent professionals may use and modify under its terms, including paid assessments; companies need authorization, including internal use. For-profit distribution and distribution to companies require Matteo Luigi Feroldi's authorization; royalties/terms are negotiated. Do not call it OSI open source or replace it without the author's request.
- Product goals: traditional and optional AI analysis (remote/local models), updated CVEs, signed reports, projects retained until deletion, fix retests and regressions. Invicti/Acunetix are desired depth references, not established equivalence. A signature does not certify that a target is secure.
- **New work rule:** one branch per block; green branch CI, then squash merge into `main` and verify final-commit CI. No status-only CI commits. M0 status belongs only in the two matrices; README and roadmap link there. Update this memory only at block end with decisions and open problems. Current author instructions take precedence.
- **Author-defined M0 criterion:** requires completion of M0-03/04/05 and automatable closure of M0-01/02/06. Gates needing people or real hardware must be marked **“requires human intervention — prerequisite for M1”** in the matrices. The present source and notice inventory suffices for M0, without additional verification layers. This is neither legal advice nor distribution approval.

### Technical decisions and limits to preserve

- The prototype is offline and uses synthetic fixtures. The Qt GUI loads 10,000 rows, filters, reads/copies evidence, offers IT/EN and persists only the language preference; it does not perform production scans. Lab transport is loopback-only and controlled by scope, DNS/IP, TLS, budgets, timeouts and cancellation ([ADR-003](DOCS/en/ADR-003-TRANSPORT.md)). Do not scan external systems without authorized targets and activities.
- SQLite is selected for a future local store; foundation tests exist, but project persistence does not. The Ed25519 JWS component requires an external trusted key; signed reports in the GUI are future work ([ADR-004](DOCS/en/ADR-004-STORAGE-SIGNATURE.md)). Native credentials are selected for macOS/Windows/Secret Service; do not fall back to plaintext storage ([ADR-005](DOCS/en/ADR-005-CREDENTIALS.md)).
- The macOS bundle applies a Qt Cocoa patch described in [ADR-006](DOCS/en/ADR-006-QT-COCOA.md); automated AX verification of current code differs from actual VoiceOver use. Development packaging includes a macOS bundle, Windows ZIP and Debian `.deb` files ([ADR-007](DOCS/en/ADR-007-PACKAGING.md)). Sources/notices are inventoried, but that is not a legal opinion or a supported installer.
- First technical M1 block: `internal/project` records an in-memory operator-confirmed assertion with claimed owner, reference, expiry and exact HTTP(S) origins. It creates an immutable run snapshot and rechecks expiry for every URL. [Contract and limits](DOCS/en/M1-PROJECT-AUTHORIZATION.md). It does not prove legal authorization, persist data or open network connections.
- Future target networking, browsers and AI must remain under deterministic policies and budgets. Treat pages and AI outputs as untrusted data. Do not commit secrets, private keys, customer data, model weights, CVE databases or real reports.

### Open problems for M1

- Trials with people and hardware: announcements and use with VoiceOver, NVDA and Orca; moving between monitors at different scales; installation and use on real Windows 10/11 and Debian desktops; tests on minimum OS versions. Reasons and procedure are in the [M0 matrices](DOCS/en/M0-VALIDATION.md).
- Qualified [legal review](DOCS/en/LEGAL-REVIEW.md) is yet to be arranged. A contributor agreement is specified but not approved: external contributions do not automatically grant relicensing rights. Set terms and process before opening the corresponding contribution path.
- A real scanner, project store, product reports, credential/key integration, AI and retests belong to later milestones. Always distinguish implemented, proposed and verified capabilities.
- Completing M1 still requires project persistence/migrations, declaration versioning/renewal, method/path/IP policy, a production broker with per-hop checks and budgets, IT/EN UI, discovery and redacted evidence. Do not connect the origin-only snapshot directly to real target traffic.
