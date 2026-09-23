# Roadmap WebFence

[Indice / Index](README.md) · [Italiano](#italiano) · [English](#english)

Aggiornamento / Updated: 2026-09-24. Responsabile / Owner: Matteo Luigi Feroldi.

## Italiano

Questa roadmap è basata su dipendenze e criteri di uscita, senza date di rilascio promesse. Esistono la fondazione documentale e il primo prototipo desktop M0 offline; M0 è ancora aperta. Lo sviluppo di un prodotto paragonabile per profondità a scanner commerciali richiede iterazioni e benchmark; non è una singola milestone.

### Fondazione documentale — completata

- [x] Visione, requisiti WF-01–WF-10 e scelta desktop nativo.
- [x] Raccomandazione Go, confronto Rust e prototipo Fyne previsto.
- [x] Licenza source-available con professionisti indipendenti ammessi e uso aziendale riservato.
- [x] Architettura, minacce, firme, AI, CVE, conservazione e retest descritti in IT/EN.
- [x] Memoria di progetto e istruzioni per agenti/contributori.

Queste spunte attestano documenti creati, non codice funzionante o revisione legale conclusa.

### M0 — Fattibilità desktop e fondazioni del codice

Dipende dalla fondazione documentale. **In corso.** Primo task: prototipo offline e fondazioni Go; [risultati e limiti](it/M0-DESKTOP.md).

- [x] Modulo Go/Fyne fissato, cataloghi IT/EN, fixture offline e test GUI con race detector.
- [x] Finestra, 10.000 righe, filtri, dettaglio/copiatore evidenze e preferenza lingua persistente.
- [x] Build e bundle locale di sviluppo su macOS Apple Silicon; CI verificata con test e quattro build native superate ([run](https://github.com/Matte2599/WebFence/actions/runs/35845953673)).
- [x] Piattaforme richieste e direzione UX confermate dall’autore: [specifica UX](it/UX.md).
- [x] Confronto locale Fyne/Qt su fixture comuni, self-test Qt e prova assistiva preliminare; [risultati e limiti](it/GUI-COMPARISON.md), [ADR-002 accettato](it/ADR-002-GUI.md).
- [x] Esperimento Qt: compilazione, vet e self-test offscreen superati su Ubuntu 24.04 x86-64/ARM64, Qt 6.4.2 ([run](https://github.com/Matte2599/WebFence/actions/runs/35849176371)); non prova il desktop assistivo Linux.
- [x] Scelta autore: Qt Widgets/MIQT + Go; [ADR-002 accettato](it/ADR-002-GUI.md), integrazione principale e preferenza lingua indipendente dal toolkit; CI Qt con cache coerente fra build e packaging.
- [x] CI della GUI Qt principale superata sui quattro target, inclusi self-test e bundle macOS ([run](https://github.com/Matte2599/WebFence/actions/runs/35853356437)); lingua persistente verificata anche con riavvio reale su macOS.
- [x] Navigazione Qt: menu Visualizza, etichette filtri, scorciatoie di focus e cambio lingua senza reset; self-test con Tab/Shift+Tab e interfaccia accessibile Qt (limiti offscreen documentati). CI superata sui quattro target ([run](https://github.com/Matte2599/WebFence/actions/runs/35857456908)); la verifica con lettori reali resta aperta.
- [x] Primo livello scope: origini HTTP(S) esatte, parser restrittivo, laboratorio HTTP loopback con redirect e destinazione esclusa, test concorrenti e fuzzing; [limiti](it/M0-SCOPE.md). CI sui quattro target superata ([run](https://github.com/Matte2599/WebFence/actions/runs/35859069506)). Non è il broker di rete completo.
- [ ] Completare verifiche Qt di menu/focus, lettori reali, stabilità e distribuzione; [stato corrente](it/QT-DESKTOP.md).
- [x] Registrato Qt come alternativa a Fyne dopo confronto pratico; gate assistivo Qt ancora aperto.

Le spunte precedenti chiudono il primo task, non tutta M0. I gate complessivi sono riportati nella [matrice di verifica e collaudo](it/M0-VALIDATION.md):

- [ ] Desktop Go/Qt: finestra, tabella ampia, lettura prove, tastiera, DPI e tecnologie assistive. [Correzione Cocoa](it/ADR-006-QT-COCOA.md) verificata sulla sequenza del crash; layout 200% corretto e provato, [CI `953ed2d` verde](https://github.com/Matte2599/WebFence/actions/runs/35873709410); Tab/Copia nativo corretto e verificato; lettori e più monitor restano aperti. [Collaudo prolungato](it/M0-STABILITY.md) implementato e [CI sei job superata](https://github.com/Matte2599/WebFence/actions/runs/35879464771); sessione Cocoa di 30 minuti completata (600 cicli), senza crash o crescita RSS sostenuta osservata; avvisi AX della prima patch registrati; sostituzione 772484 senza avvisi in 60 s, da collaudare ancora con lettori. CI Cocoa aggiunta: difetto dimensioni su macOS 15 isolato (59/60 punti), correzione reattiva e regressione prima/dopo locali; [CI `8d3053f` superata, sei job](https://github.com/Matte2599/WebFence/actions/runs/35885936443). Nuova sessione con patch 772484 completata: 584 cicli in oltre 30 minuti, stderr vuoto, RSS senza crescita sostenuta osservata; cadenza irregolare documentata.
- [ ] Build e packaging di prova sui sistemi candidati; matrice OS/architetture reale. [Ricognizione](it/M0-PACKAGING.md): il bundle locale richiede macOS 26; minimo desiderato da fissare e verificare. [Pacchetti Debian/Windows](it/ADR-007-PACKAGING.md) e runtime separati verificati: [CI `32655b0`, sei job verdi](https://github.com/Matte2599/WebFence/actions/runs/35877356234), inclusi Debian amd64/arm64 e ZIP Windows fuori toolchain. Inventario automatico macOS verificato in CI; [raccoglitore sorgenti](it/M0-SOURCE-MATERIALS.md) con 15 archivi upstream verificati e 247 file di avvisi. Inclusione facoltativa degli avvisi nel bundle prima della firma implementata e verificata localmente; [CI `56840c5`, sei job superati](https://github.com/Matte2599/WebFence/actions/runs/35898848286). Raccoglitore comune aggiunto per includere DOCS/IT-EN e correggere i link locali nei pacchetti Windows/Debian; [CI `f99b3e7`, sei job superati](https://github.com/Matte2599/WebFence/actions/runs/35900777988). Confronto 43 DLL Windows con 22 archivi MSYS2 e hash delle ricette verificato nella [CI `054bb2d`, sei job superati](https://github.com/Matte2599/WebFence/actions/runs/35904059667). [Raccolta Windows](it/M0-WINDOWS-SOURCES.md): 21 archivi/316.641.108 byte, 103 checksum e supplemento Git verificati; firme distaccate non verificate. [CI `37da7cf`](https://github.com/Matte2599/WebFence/actions/runs/35907192055): sei job superati, 20 regressioni Python sui quattro target. Inclusione facoltativa dei 21 archivi sorgente nello ZIP implementata, con ricette rigenerate e vincolo all’inventario corrente; 25 regressioni locali passate. CI [`69f1dcc`](https://github.com/Matte2599/WebFence/actions/runs/35908880543) completata: sei job superati, 25 regressioni Python sui quattro target; su Windows raccolti/inclusi 21 archivi, verificati gli hash nello ZIP estratto, preservato lo ZIP su errore e superati self-test/soak offscreen e nativi con PATH di solo sistema. Supplementi Homebrew GLib/libb2 integrati prima della firma: quattro file/1.093.639 byte e due ricette vincolati agli hash revisionati, senza esecuzione. Piano con varianti libb2 Tahoe/Sequoia verificate; il manifest registra la variante effettiva. Il runner richiede `brew update` e GLib esplicito: verificato passaggio 2.88.3 → 2.90.0. Bundle locale con 247 avvisi e supplementi: firma/Cocoa e preservazione su piano errato superati, lingua personale invariata. [CI `5521c34`](https://github.com/Matte2599/WebFence/actions/runs/35912249024): sei job superati, 31 regressioni Python sui quattro target, supplementi firmati verificati su macOS 15. Completezza sorgenti e revisione distribuzione restano aperte; dettagli in M0-SOURCE-MATERIALS. Minimo macOS per artefatto: `macho_minos.py` legge tutti i load command di deployment e imposta LSMinimumSystemVersion prima della firma, preservando un valore preesistente più alto. Inventario con minimo per file/massimo e file determinanti; requisiti incorporati nei binari invariati. Regressione: vecchio plist senza chiave pur con minimo 26. Nuovo bundle locale 26.0.0 (19 file minimo 26, nove minimo 14), verifica indipendente vtool su 28 file, firma e self-test Cocoa passati; 36 regressioni Python locali. [CI `3593dcc`](https://github.com/Matte2599/WebFence/actions/runs/35914285406): sei job superati, 36 regressioni Python sui quattro target. Bundle del runner macOS 15: minimo dichiarato/Mach-O 15.0.0 e 28 file ricontrollati con vtool, firma e Cocoa superati. Non definisce il minimo ufficiale né prova avvio sotto requisito. Patch/risorse, mappatura, assemblaggio completo, desktop reali e minimi OS restano aperti.
- [x] Selezione motivata di GUI, SQLite, JWS e portachiavi: [ADR-004](it/ADR-004-STORAGE-SIGNATURE.md), [ADR-005](it/ADR-005-CREDENTIALS.md); prove native completate sui quattro target, inclusa indisponibilità Windows con token anonimo ([CI](https://github.com/Matte2599/WebFence/actions/runs/35870058803)).
- [x] Versioni fissate, scheletro minimo, cataloghi IT/EN, CI per il codice introdotto.
- [x] Laboratorio sintetico e primi test scope/rete: origini, DNS/IP per connessione, TLS, redirect, budget condivisi e cancellazione verificati su loopback; [ADR-003 e limiti](it/ADR-003-TRANSPORT.md), [CI verde sui quattro target](https://github.com/Matte2599/WebFence/actions/runs/35861040736). Trasporto per target reali e policy complete restano in M1.
- [ ] Revisione legale del testo, canale privato di sicurezza e accordo contributori prima delle rispettive aperture. Canale GitHub privato abilitato e verificato il 2026-09-23; revisione e accordo ancora aperti.

Uscita: prototipo distribuibile sulle piattaforme inizialmente dichiarate, lettura delle prove accessibile, rischi di packaging documentati e ADR aggiornato. Nessuna capacità di scansione professionale dichiarata.

Progresso aggiuntivo M0-02: Ricompilazione/sostituzione Cocoa: nuovo collaudo nativo usa script/patch/CMake inclusi nel bundle e una modifica diagnostica solo nella copia privata. Prova locale macOS 26.6.2 superata: stesso Go build ID, firma ad hoc, self-test e quattro cicli/12,003 s, marker caricato due volte. Bundle originale completo, plugin installato e lingua preservati; archivio invalido e output esistente rifiutati. Primo tentativo intercetta risoluzione opt→Cellar errata nel collaudo; prefisso ora preservato. Evidenza distingue bundle f64f534+dirty dal nuovo script; procedura IT/EN M0-QT-REPLACEMENT. Collaudo incluso nei materiali confezionati e CI macOS; [CI `3f68e6a`](https://github.com/Matte2599/WebFence/actions/runs/35916839178) completata: sei job superati. Su macOS 15.7.9 la procedura inclusa nel bundle ricompila/carica la variante Cocoa, mantiene il Go build ID e supera firma, self-test e soak; originali preservati. Le 36 regressioni Python e i controlli dei pacchetti Windows/Debian sono verdi. La run precedente `51673d9` (35916784359) è stata annullata dal push successivo, che codifica lo spazio di contesto della patch senza cambiarne i byte prodotti. Prova del solo plugin, nessuna chiusura generale M0/distribuzione/legale.

Avvisi Qt: individuati e recuperati 11 testi LicenseFile/LicenseFiles omessi dai nomi generici. Raccoglitore a due letture, riferimenti confinati alla radice, limiti e rigenerazione del registro in inclusione. Raccolta reale 15 archivi/258 avvisi (247 precedenti invariati), tutti gli hash ricontrollati e inclusione separata superata; output /tmp/webfence-native-sources-3598b5f-qt-references. Vecchie raccolte Qt rifiutate: rigenerarle riusando gli archivi. Regressione riprodotta sul codice precedente; test sintetici aggiunti. Bundle locale ricostruito da `3598b5f` con modifiche dichiarate: 258 avvisi ricontrollati, legame con l’inventario corrente, supplementi Homebrew presenti, firma ad hoc e self-test Cocoa superati; preferenza lingua preservata. 41 test Python locali superati; [CI `98478f2`](https://github.com/Matte2599/WebFence/actions/runs/35918855381) completata: sei job superati e 41 regressioni Python sui quattro target nativi. Bundle/Cocoa/sostituzione plugin macOS, ZIP Windows con sorgenti e prova di preservazione su errore, pacchetti Debian amd64/arm64 superati. I 258 avvisi reali sono verificati nel bundle locale; la CI del bundle macOS non acquisisce ancora l’intera raccolta facoltativa. Restano mappatura dei componenti distribuiti, assemblaggio completo e gate esterni M0.

Pacchetto separato dei materiali nativi macOS: `scripts/package-macos-sources.py` verifica la firma dell’app, rigenera avvisi/supplementi, include archivi originali e registra hash dopo la firma per abbinare il bundle. Nessun download o modifica dell’app; ZIP riletto e pubblicato senza sovrascritture. Prova reale: 173.236.303 byte, 15 archivi, 258 avvisi, quattro supplementi, 464 file materiali e 28 binari associati; unzip macOS e preservazione ZIP esistente superati. 46 test Python locali superati. Output locale dist/WebFence-native-sources-local.zip; evidenza DOCS/evidence/macos-source-package-2026-09-23.json. [CI `f46768d`](https://github.com/Matte2599/WebFence/actions/runs/35920813028) completata: sei job superati e 46 test Python sui quattro target nativi. Il runner macOS acquisisce realmente tutti i 15 archivi, rigenera 258 avvisi e quattro supplementi, crea uno ZIP di 173.243.954 byte associato a 28 binari firmati e supera unzip; Cocoa e sostituzione plugin restano verdi. Windows e Debian amd64/arm64 superano i rispettivi collaudi. Materiali disponibili riuniti, non completezza dei sorgenti corrispondenti o approvazione legale; Risorse ulteriori, mappatura e gate esterni ancora aperti.

Revisione tecnica delle 15 ricette Homebrew incluse nel pacchetto nativo: nessun altro download esplicito di patch/risorse per la build stabile macOS individuato oltre ai supplementi GLib/libb2 già raccolti; patch D-Bus inline presente. Font Graphite2/HarfBuzz e JPEG sono fixture dei test; PNG libpng aggiuntivo solo Linux. Hash e copie delle 15 ricette, builder/patch Cocoa ricontrollati contro lo ZIP. Registro bilingue in DOCS/evidence/homebrew-recipe-review-2026-09-23.md; dossier legale aggiornato ai materiali effettivamente disponibili. Non prova ricostruzione dell’intero ambiente, mappatura degli incorporati o compatibilità legale. Codice invariato rispetto a f46768d, CI run 35920813028 già superata; M0 aperta.

[Evidenza Qt](evidence/qt-component-review-2026-09-23.md). Mappatura Qt del bundle macOS: nove binari associati per percorso alle entità della SBOM conservata; Cocoa trattato come baseline upstream più patch WebFence. 62 pacchetti raggiungibili tramite DEPENDS_ON, di cui 32 attribuzioni, non prova di inclusione effettiva. Verificati 27 metadati e 22 testi contro archivio Qt, bundle e ZIP; tutti i LicenseId coincidono. Hash dopo firma dei nove binari e firma bundle ricontrollati. Cinque degli otto checksum SHA-1 upstream differiscono dal keg: nessuna identità binaria dedotta dalla SBOM. Esame tecnico parziale, nessuna selezione/approvazione legale; dettaglio e limiti nel registro bilingue.

[CI `f345a1f`](https://github.com/Matte2599/WebFence/actions/runs/35923685155) conclusa con sei job verdi, tentativo 2: il primo job macOS era fallito per timeout DNS scaricando D-Bus, dopo test GUI/Cocoa e sostituzione plugin riusciti. Ritentato soltanto macOS, senza modifiche al codice; raccolta finale 15 archivi/258 avvisi/quattro supplementi/28 binari associati superata. Gli altri cinque job erano già verdi; 46 regressioni Python sui quattro target nativi e pacchetti Windows/Debian superati. Documentazione della build: 76 Markdown/629 collegamenti locali. Matrice aggiornata per richiamare il collaudo Cocoa già completato di oltre 30 minuti, distinto dalla prova con lettore reale. M0 resta aperta.

[Registro non Qt](evidence/macos-nonqt-component-review-2026-09-23.md). Verificata la provenienza tecnica dei 19 Mach-O non Qt del bundle macOS locale: 18 librerie di 14 pacchetti Homebrew più l’eseguibile Go. Con i nove file Qt già mappati, tutti i 28 file inventariati hanno un’associazione tecnica per questa build. Hash di librerie/keg, 14 archivi originali, ricette/SBOM e 90 avvisi sorgente confrontati con app e ZIP; firma ad hoc valida. Le espressioni SBOM riguardano archivi sorgente, non assegnano automaticamente licenze ai singoli binari. Completezza dei sorgenti, codice incorporato e revisione legale restano aperti.

[Controllo Mach-O](evidence/macos-linkage-closure-2026-09-24.md). Controllo di chiusura Mach-O integrato nel packaging macOS: sei LC_RPATH esterni eliminati nello staging prima di inventario/firma, quindi rifiuto di dipendenze non Apple fuori dal bundle. Prova su copia privata: 28 binari, 194 riferimenti (148 Apple, 46 interni), firma ad hoc, self-test Cocoa e soak 10 s superati; originali preservati. 53 regressioni Python locali verdi, di cui sette nuove. La [CI sul codice integrato](https://github.com/Matte2599/WebFence/actions/runs/35927157090) ha superato sei job; la prova non copre dlopen o Mac puliti. [Evidenza](evidence/macos-linkage-closure-2026-09-24.md).

[CI `c47f1ea`](https://github.com/Matte2599/WebFence/actions/runs/35927157090) conclusa al primo tentativo: sei job verdi. Sui quattro target nativi passano 53 regressioni Python; macOS 15 rimuove sei RPATH esterni e verifica 28 binari/194 riferimenti (148 Apple, 46 interni) nel bundle firmato, poi supera Cocoa, sostituzione plugin e pacchetto sorgenti 15/258/4/28. Windows passa anche da percorso Unicode/spazi senza toolchain nel PATH; Debian amd64/arm64 passa runtime separato e purge. Questo chiude la verifica del controllo di collegamento, non i gate M0 su lettori, macchine pulite, minimi OS o revisione legale.

[Prova runtime macOS isolata](evidence/macos-isolated-runtime-2026-09-24.md): su copia privata firmata, ambiente figlio con PATH di sistema, self-test Cocoa con due plugin caricati dal bundle e soak tracciato con 24 immagini app/1.127 totali, nessuna esterna a bundle o sistema Apple. [CI `bc217a2`: sei job verdi](https://github.com/Matte2599/WebFence/actions/runs/35929219514), 57 regressioni Python sui target nativi; sul runner macOS 15 due plugin, 24 immagini app/1.033 totali e quattro cicli in 12,102 s. Il self-test con tracciamento dyld simultaneo ha avuto cinque asserzioni Tab fallite; le prove separate non sostituiscono Mac pulito, lettori o revisione legale.

[Difetto AX dopo reset dei filtri](evidence/macos-ax-table-reset-2026-09-24.md): su una copia privata del bundle, la sequenza 10.000 → 1 → 0 → 10.000 righe lascia `AXRows` con 10.000 riferimenti ma la prima riga non valida (`kAXErrorInvalidUIElement`). Confermato a stato fermo da un nuovo client AX; self-test Qt interno e soak non lo rilevano. Due varianti private del plugin non hanno corretto il difetto; la terza è inconcludente. Un controllo UI successivo ha segnalato Mac bloccato, senza stabilire da quando: ricontrollare a schermo sbloccato, poi correggere se confermato e provare un lettore reale. Bundle originale e preferenza personale preservati; M0-01 aperto.

[Metadati licenza Windows](evidence/windows-license-metadata-2026-09-24.md): hash di 21 archivi sorgente e 42 ricette/metadati ricontrollati; campi `license` delle ricette indicizzati per i 22 pacchetti binari, con distinzione `libiconv`. Nessuna conclusione sulla licenza effettiva delle 43 DLL o sui componenti incorporati; notices, firme PGP SKIP, sostituzione e revisione legale ancora aperti. M0-02/06 invariati.

### M1 — Alpha tradizionale controllata

Dipende da M0. Copre WF-01, WF-02, WF-03, WF-07, WF-09.

- [ ] Progetti, scope/autorizzazione, trasporto controllato, budget e cancellazione.
- [ ] Discovery HTTP(S), primi controlli a basso impatto e prove redatte.
- [ ] Persistenza, ripristino dopo crash, limiti disco e cancellazione base del progetto.
- [ ] Desktop IT/EN con avanzamento, risultati e copertura incompleta esplicita.

Uscita: flusso progetto → scansione lab → risultati → riavvio verificato; suite di scope senza violazioni, casi vulnerabili/corretti per ogni regola, nessun LLM richiesto. Alpha utilizzabile soltanto nei limiti documentati.

### M2 — Intelligence e report verificabili

Dipende da M1. Copre WF-04 e WF-06.

- [ ] Adattatori CVE/NVD, cache incrementale, provenienza e stato di aggiornamento.
- [ ] Matching con confidenza, versioni/backport e distinzione tra candidato e verificato.
- [ ] Export JSON/HTML IT/EN, manifest, firma, gestione chiavi e verificatore offline.
- [ ] Prove di manomissione, chiave non fidata, feed indisponibile e sync interrotta.

Uscita: bundle originale verificabile e alterazioni respinte; ultima cache valida recuperabile; report con controlli non eseguiti visibili. Nessuna dichiarazione di firma qualificata o copertura CVE totale.

### M3 — Applicazioni dinamiche e autenticate

Dipende da M2. Estende WF-03 e realizza WF-10.

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

This roadmap uses dependencies and exit criteria, without promised release dates. The documentation foundation and first offline M0 desktop prototype exist; M0 remains open. Building coverage comparable in depth to commercial scanners requires iterations and benchmarks, not one milestone.

### Documentation foundation — complete

- [x] Vision, WF-01–WF-10 requirements and native desktop decision.
- [x] Go recommendation, Rust comparison and planned Fyne prototype.
- [x] Source-available license allowing independent professionals and reserving company use.
- [x] IT/EN architecture, threats, signatures, AI, CVE, retention and retest specifications.
- [x] Project memory and agent/contributor instructions.

These checks represent created documents, not working code or completed legal review.

### M0 — Desktop feasibility and code foundations

Depends on the documentation foundation. **In progress.** First task: offline prototype and Go foundations; [results and limitations](en/M0-DESKTOP.md).

- [x] Pinned Go/Fyne module, IT/EN catalogs, offline fixtures and GUI tests with race detector.
- [x] Window, 10,000 rows, filters, evidence reader/copy and persistent language preference.
- [x] Local development build and bundle on Apple Silicon macOS; verified CI with tests and four passing native builds ([run](https://github.com/Matte2599/WebFence/actions/runs/35845953673)).
- [x] Author-confirmed requested platforms and UX direction: [UX specification](en/UX.md).
- [x] Local Fyne/Qt comparison with shared fixtures, Qt self-test and preliminary assistive trial; [results and limitations](en/GUI-COMPARISON.md), [accepted ADR-002](en/ADR-002-GUI.md).
- [x] Qt experiment: build, vet and offscreen self-test passed on Ubuntu 24.04 x86-64/ARM64, Qt 6.4.2 ([run](https://github.com/Matte2599/WebFence/actions/runs/35849176371)); does not establish Linux assistive desktop behavior.
- [x] Author choice: Qt Widgets/MIQT + Go; [accepted ADR-002](en/ADR-002-GUI.md), main integration and toolkit-independent language preference; Qt CI with consistent build/packaging cache flags.
- [x] Main Qt GUI CI passed on all four targets, including self-tests and macOS bundle ([run](https://github.com/Matte2599/WebFence/actions/runs/35853356437)); persistent language also checked through actual macOS restart.
- [x] Qt navigation: View menu, filter labels, focus shortcuts and language change without reset; self-test covers Tab/Shift+Tab and the Qt accessible interface (offscreen limitations documented). CI passed on all four targets ([run](https://github.com/Matte2599/WebFence/actions/runs/35857456908)); actual screen-reader verification remains open.
- [x] First scope layer: exact HTTP(S) origins, strict parser, loopback HTTP lab with redirects and an excluded destination, concurrent tests and fuzzing; [limitations](en/M0-SCOPE.md). CI passed on all four targets ([run](https://github.com/Matte2599/WebFence/actions/runs/35859069506)). Not the complete network broker.
- [ ] Complete Qt menus/focus, actual readers, stability and distribution checks; [current state](en/QT-DESKTOP.md).
- [x] Recorded Qt as the Fyne alternative after practical comparison; Qt assistive gate remains open.

These checks close the first task, not all of M0. Overall gates are tracked in the [verification and trial matrix](en/M0-VALIDATION.md):

- [ ] Go/Qt desktop: window, large table, evidence reading, keyboard, DPI and assistive technologies. [Cocoa correction](en/ADR-006-QT-COCOA.md) verified against the crash sequence; 200% layout corrected and tested, [CI `953ed2d` passed](https://github.com/Matte2599/WebFence/actions/runs/35873709410); native Tab/Copy corrected and verified; readers and multi-monitor remain open. [Prolonged trial](en/M0-STABILITY.md) implemented and [six CI jobs passed](https://github.com/Matte2599/WebFence/actions/runs/35879464771); 30-minute Cocoa session completed (600 cycles), without crashes or observed sustained RSS growth; first-patch AX warnings recorded; replacement 772484 without warnings in 60 s, actual reader trials still needed. Cocoa CI added: macOS 15 sizing defect isolated (59/60 points), reactive correction and local before/after regression; [CI `8d3053f` passed, six jobs](https://github.com/Matte2599/WebFence/actions/runs/35885936443). New session with patch 772484 completed: 584 cycles in over 30 minutes, empty stderr, no observed sustained RSS growth; uneven cadence documented.
- [ ] Trial builds and packaging on candidate systems; actual OS/architecture matrix. [Investigation](en/M0-PACKAGING.md): local bundle requires macOS 26; intended minimum must be fixed and verified. [Debian/Windows packages](en/ADR-007-PACKAGING.md) and separate runtimes verified: [CI `32655b0`, six passing jobs](https://github.com/Matte2599/WebFence/actions/runs/35877356234), including Debian amd64/arm64 and Windows ZIP outside the toolchain. Automated macOS inventory verified in CI; [source collector](en/M0-SOURCE-MATERIALS.md) with 15 verified upstream archives and 247 notice files. Optional notice attachment before bundle signing implemented and verified locally; [CI `56840c5`, six passing jobs](https://github.com/Matte2599/WebFence/actions/runs/35898848286). Shared collector added to include DOCS/IT-EN and fix local links in Windows/Debian packages; [CI `f99b3e7`, six passing jobs](https://github.com/Matte2599/WebFence/actions/runs/35900777988). 43 Windows DLLs matched against 22 MSYS2 archives and recipe hashes in [CI `054bb2d`, six passing jobs](https://github.com/Matte2599/WebFence/actions/runs/35904059667). [Windows collection](en/M0-WINDOWS-SOURCES.md): 21 archives/316,641,108 bytes, 103 checksums and Git supplement verified; detached signatures unverified. [CI `37da7cf`](https://github.com/Matte2599/WebFence/actions/runs/35907192055): six passing jobs, 20 Python regressions on four targets. Optional attachment of 21 source archives in the ZIP implemented, with regenerated recipes and current-inventory binding; 25 local regressions passed. CI [`69f1dcc`](https://github.com/Matte2599/WebFence/actions/runs/35908880543) completed: six passing jobs, 25 Python regressions on four targets; Windows collected/attached 21 archives, verified extracted-ZIP hashes, preserved the ZIP on failure and passed offscreen/native self-tests and soaks with system-only PATH. Homebrew GLib/libb2 supplements attached before signing: four files/1,093,639 bytes and two recipes bound to reviewed hashes, without execution. Reviewed Tahoe/Sequoia libb2 variants; manifest records the actual variant. Runner requires `brew update` and explicit GLib: 2.88.3 → 2.90.0 verified. Local bundle with 247 notices and supplements: signature/Cocoa and preservation on invalid plan passed, personal language unchanged. [CI `5521c34`](https://github.com/Matte2599/WebFence/actions/runs/35912249024): six passing jobs, 31 Python regressions on four targets, signed supplements verified on macOS 15. Source completeness and distribution review remain open; details in M0-SOURCE-MATERIALS. Per-artifact macOS minimum: `macho_minos.py` reads all deployment load commands and sets LSMinimumSystemVersion before signing, preserving a higher existing value. Inventory records per-file/maximum requirements and limiting files; embedded deployment targets unchanged. Regression: previous plist lacked the key despite minimum 26. New local bundle 26.0.0 (19 files minimum 26, nine minimum 14), independent vtool checks on 28 files, signature and Cocoa self-test passed; 36 local Python regressions. [CI `3593dcc`](https://github.com/Matte2599/WebFence/actions/runs/35914285406): six passing jobs, 36 Python regressions on four targets. macOS 15 runner bundle: declared/Mach-O minimum 15.0.0 and 28 files rechecked with vtool, signature and Cocoa passed. Does not define official minimum or test launch below the requirement. Patches/resources, mapping, complete assembly, real desktops and OS minima remain open.
- [x] Justified selection of GUI, SQLite, JWS and keychain: [ADR-004](en/ADR-004-STORAGE-SIGNATURE.md), [ADR-005](en/ADR-005-CREDENTIALS.md); native tests completed on all four targets, including Windows unavailability with an anonymous token ([CI](https://github.com/Matte2599/WebFence/actions/runs/35870058803)).
- [x] Pinned versions, minimal skeleton, IT/EN catalogs and CI for introduced code.
- [x] Synthetic lab and initial scope/network tests: origins, per-connection DNS/IP, TLS, redirects, shared budgets and cancellation checked on loopback; [ADR-003 and limitations](en/ADR-003-TRANSPORT.md), [passing CI on all four targets](https://github.com/Matte2599/WebFence/actions/runs/35861040736). Real-target transport and complete policies remain in M1.
- [ ] Legal review, private security channel and contributor agreement before opening the corresponding processes. GitHub private channel enabled and verified on 2026-09-23; review and agreement remain open.

Exit: distributable prototype on initially declared platforms, accessible evidence reading, documented packaging risks and updated ADR. No professional scanning capability claimed.

Additional M0-02 progress: Cocoa rebuild/replacement: new native trial uses scripts/patch/CMake shipped in the bundle and a diagnostic modification only in the private copy. Local macOS 26.6.2 trial passed: same Go build ID, ad hoc signature, self-test and four cycles/12.003 s, marker loaded twice. Entire original bundle, installed plugin and language preserved; invalid archive and existing output rejected. First attempt caught incorrect opt→Cellar resolution in the trial; prefix now preserved. Evidence distinguishes f64f534+dirty bundle from new script; IT/EN procedure M0-QT-REPLACEMENT. Trial included in packaged materials and macOS CI; [CI `3f68e6a`](https://github.com/Matte2599/WebFence/actions/runs/35916839178) completed: six passing jobs. On macOS 15.7.9 the procedure shipped in the bundle rebuilds/loads the Cocoa variant, retains the Go build ID and passes signing, self-test and soak; originals preserved. The 36 Python regressions and Windows/Debian package checks passed. The previous `51673d9` run (35916784359) was cancelled by the next push, which encodes patch-context whitespace without changing generated patch bytes. Plugin-only proof, no general M0/distribution/legal closure.

Qt notices: found and recovered 11 LicenseFile/LicenseFiles texts missed by generic names. Two-pass collector, references confined to archive root, budgets and regenerated ledger during attachment. Actual collection: 15 archives/258 notices (previous 247 unchanged), all hashes rechecked and separate attachment passed; output /tmp/webfence-native-sources-3598b5f-qt-references. Older Qt collections rejected: regenerate using reusable archives. Regression reproduced with previous code; synthetic tests added. Local bundle rebuilt from `3598b5f` with declared modifications: 258 notices rechecked, current-inventory binding, Homebrew supplements present, ad hoc signature and Cocoa self-test passed; personal language preserved. 41 local Python tests passed; [CI `98478f2`](https://github.com/Matte2599/WebFence/actions/runs/35918855381) completed: six passing jobs and 41 Python regressions on four native targets. macOS bundle/Cocoa/plugin replacement, Windows ZIP with sources and preservation on failure, and Debian amd64/arm64 packages passed. The actual 258 notices are verified in the local bundle; macOS bundle CI still does not acquire the full optional collection. Shipped-component mapping, complete assembly and external M0 gates remain open.

Separate macOS native-material package: `scripts/package-macos-sources.py` verifies the app signature, regenerates notices/supplements, includes original archives and records post-signing hashes to associate the bundle. No downloads or app modification; ZIP read back and published without overwriting. Actual trial: 173,236,303 bytes, 15 archives, 258 notices, four supplements, 464 material files and 28 associated binaries; macOS unzip and existing-ZIP preservation passed. 46 local Python tests passed. Local output dist/WebFence-native-sources-local.zip; evidence DOCS/evidence/macos-source-package-2026-09-23.json. [CI `f46768d`](https://github.com/Matte2599/WebFence/actions/runs/35920813028) completed: six passing jobs and 46 Python tests on four native targets. The macOS runner actually downloads all 15 archives, regenerates 258 notices and four supplements, creates a 173,243,954-byte ZIP associated with 28 signed binaries and passes unzip; Cocoa and plugin replacement remain green. Windows and Debian amd64/arm64 pass their respective trials. Available materials assembled, not complete corresponding sources or legal approval; Further resources, mapping and external gates remain open.

Technical review of the 15 Homebrew recipes included in the native package: no other explicit patch/resource download identified for the stable macOS build beyond already collected GLib/libb2 supplements; inline D-Bus patch present. Graphite2/HarfBuzz fonts and JPEG are test fixtures; additional libpng PNG is Linux-only. Hashes and copies of all 15 recipes plus Cocoa builder/patch rechecked against the ZIP. Bilingual ledger in DOCS/evidence/homebrew-recipe-review-2026-09-23.md; legal dossier updated to actually available materials. Does not prove whole-environment rebuilding, embedded mapping or legal compatibility. Code unchanged from f46768d, CI run 35920813028 already passed; M0 remains open.

[Qt evidence](evidence/qt-component-review-2026-09-23.md). Qt mapping for the macOS bundle: nine binaries associated by path with retained SBOM entities; Cocoa treated as upstream baseline plus WebFence patch. 62 packages reachable through DEPENDS_ON, including 32 attributions, not proof of actual inclusion. Verified 27 metadata files and 22 texts against Qt archive, bundle and ZIP; all LicenseId values match. Rechecked nine post-signing binary hashes and bundle signature. Five of eight upstream SHA-1 checksums differ from the keg: no binary identity inferred from the SBOM. Partial technical review, no legal selection/approval; details and limits in the bilingual ledger.

[CI `f345a1f`](https://github.com/Matte2599/WebFence/actions/runs/35923685155) completed with six passing jobs, attempt 2: the first macOS job failed on a DNS timeout while downloading D-Bus, after passing GUI/Cocoa and plugin replacement trials. Retried macOS only, without code changes; final collection of 15 archives/258 notices/four supplements/28 associated binaries passed. The other five jobs had already passed; 46 Python regressions on four native targets and Windows/Debian packages passed. Built documentation: 76 Markdown files/629 local links. Matrix updated to reference the already completed Cocoa trial exceeding 30 minutes, separately from actual reader testing. M0 remains open.

[Non-Qt ledger](evidence/macos-nonqt-component-review-2026-09-23.md). Verified technical provenance for the 19 non-Qt Mach-O files in the local macOS bundle: 18 libraries from 14 Homebrew packages plus the Go executable. Together with nine mapped Qt files, all 28 inventoried files have a technical association for this build. Library/keg hashes, 14 original archives, recipes/SBOMs and 90 source notices were compared with the app and ZIP; ad hoc signature valid. SBOM expressions cover source archives and do not automatically assign licenses to individual binaries. Source completeness, embedded code and legal review remain open.

[Mach-O check](evidence/macos-linkage-closure-2026-09-24.md). Mach-O closure check integrated into macOS packaging: six external LC_RPATH entries removed in staging before inventory/signing, then rejection of non-Apple dependencies outside the bundle. Private-copy trial: 28 binaries, 194 references (148 Apple, 46 internal), ad hoc signature, Cocoa self-test and 10-second soak passed; originals preserved. 53 local Python regressions passed, including seven new ones. [CI for the integrated code](https://github.com/Matte2599/WebFence/actions/runs/35927157090) passed six jobs; the trial does not cover dlopen or clean Macs. [Evidence](evidence/macos-linkage-closure-2026-09-24.md).

[CI `c47f1ea`](https://github.com/Matte2599/WebFence/actions/runs/35927157090) completed on the first attempt: six passing jobs. All four native targets pass 53 Python regressions; macOS 15 removes six external rpaths and checks 28 binaries/194 references (148 Apple, 46 internal) in the signed bundle, then passes Cocoa, plugin replacement and the 15/258/4/28 native-source package. Windows also passes from a Unicode/spaced path without the toolchain in PATH; Debian amd64/arm64 passes separate-runtime and purge checks. This closes verification of the linkage check, not M0 gates for readers, clean machines, minimum OS versions or legal review.

[Isolated macOS runtime trial](evidence/macos-isolated-runtime-2026-09-24.md): a signed private copy, child environment with a system-only PATH, Cocoa self-test with two plugins loaded from the bundle, and a traced soak with 24 app images/1,127 total, none outside the bundle or Apple system. [CI `bc217a2`: six passing jobs](https://github.com/Matte2599/WebFence/actions/runs/35929219514), 57 Python regressions on native targets; macOS 15 runner observed two plugins, 24 app images/1,033 total and four cycles in 12.102 s. A self-test with simultaneous dyld tracing failed five Tab assertions; the separate checks do not replace a clean Mac, readers or legal review.

[AX defect after filter reset](evidence/macos-ax-table-reset-2026-09-24.md): on a private bundle copy, the 10,000 → 1 → 0 → 10,000 row sequence leaves `AXRows` reporting 10,000 references while the first row is invalid (`kAXErrorInvalidUIElement`). Confirmed in an idle state from a fresh AX client; the internal Qt self-test and soak miss it. Two private plugin variants did not fix it; the third result is inconclusive. A later UI check reported the Mac locked without establishing when: retest with the screen unlocked, then fix if confirmed and try a real reader. Original bundle and personal preference preserved; M0-01 open.

[Windows license metadata](evidence/windows-license-metadata-2026-09-24.md): hashes of 21 source archives and 42 recipe/metadata files rechecked; recipe `license` fields indexed for the 22 binary packages, distinguishing `libiconv`. No conclusion about effective licenses of the 43 DLLs or embedded components; notices, SKIP PGP signatures, replacement and legal review remain open. M0-02/06 unchanged.

### M1 — Controlled traditional alpha

Depends on M0. Covers WF-01, WF-02, WF-03, WF-07, WF-09.

- [ ] Projects, scope/authorization, controlled transport, budgets and cancellation.
- [ ] HTTP(S) discovery, initial low-impact checks and redacted evidence.
- [ ] Persistence, crash recovery, disk limits and basic project deletion.
- [ ] IT/EN desktop with progress, results and explicit incomplete coverage.

Exit: verified project → lab scan → results → restart workflow; scope suite without violations, vulnerable/fixed cases per rule, no LLM required. Alpha usable only within documented limits.

### M2 — Intelligence and verifiable reports

Depends on M1. Covers WF-04 and WF-06.

- [ ] CVE/NVD adapters, incremental cache, provenance and freshness status.
- [ ] Confidence-aware matching, versions/backports and candidate/verified distinction.
- [ ] IT/EN JSON/HTML export, manifests, signing, key management and offline verifier.
- [ ] Tampering, untrusted-key, unavailable-feed and interrupted-sync tests.

Exit: original bundle verifies and alterations are rejected; last valid cache is recoverable; reports show unexecuted checks. No qualified-signature or complete-CVE-coverage claims.

### M3 — Dynamic and authenticated applications

Depends on M2. Extends WF-03 and implements WF-10.

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
