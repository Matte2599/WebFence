# WebFence

**Analisi della sicurezza web, evidenze verificabili e convalida dei fix.**

[English](README.en.md) · [Documentazione](DOCS/README.md) · [Roadmap](DOCS/ROADMAP.md) · [Licenza](LICENSE)

**Stato: M0 in corso — primo prototipo desktop eseguibile, solo con esempi sintetici.** La GUI Go/Qt Widgets permette di caricare 10.000 righe, filtrarle e leggere le evidenze in IT/EN. Il motore di scansione non è implementato; non esistono release supportate o benchmark di sicurezza. Vedi il [resoconto M0](DOCS/it/QT-DESKTOP.md).

**Qt è stato scelto dall’autore** dopo il [confronto pratico](DOCS/it/GUI-COMPARISON.md). Il comando principale usa Qt Widgets tramite MIQT; [ADR accettato](DOCS/it/ADR-002-GUI.md). Accessibilità completa e distribuzione restano gate aperti.

Il core comprende un primo [controllo delle origini autorizzate](DOCS/it/M0-SCOPE.md), con laboratorio HTTP solo nei test su loopback. È stato aggiunto anche un [trasporto di laboratorio](DOCS/it/ADR-003-TRANSPORT.md) con DNS/IP verificati, TLS, budget e cancellazione, limitato a loopback. La GUI resta offline; nessuno scanner di produzione.

Le fondazioni includono anche prove SQLite e un componente JWS Ed25519 limitato: [scelte e verifiche M0](DOCS/it/ADR-004-STORAGE-SIGNATURE.md). Persistenza dei progetti e report firmati nella GUI non sono ancora disponibili. Un [adattatore portachiavi nativo](DOCS/it/ADR-005-CREDENTIALS.md) separato è verificato nativamente sui quattro target CI, comprese condizioni di indisponibilità ([run](https://github.com/Matte2599/WebFence/actions/runs/35870058803)).

Il bundle macOS include una [correzione temporanea Qt Cocoa](DOCS/it/ADR-006-QT-COCOA.md) per il crash assistivo riprodotto; layout compatto verificato al 200% e [CI verde sui quattro target](https://github.com/Matte2599/WebFence/actions/runs/35873709410). Restano aperti collaudi con lettori reali e distribuzione.

Il bundle macOS raccoglie ora [provenienza e attribuzioni disponibili](DOCS/it/M0-PACKAGING.md), segnalando esplicitamente i materiali ancora mancanti per la distribuzione.

Disponibili anche procedure di [packaging Debian e Windows](DOCS/it/ADR-007-PACKAGING.md). Verificati `.deb` amd64/arm64 in runtime Debian separati e ZIP Windows con PATH di solo sistema; [CI `32655b0`, sei job verdi](https://github.com/Matte2599/WebFence/actions/runs/35877356234). Pacchetti di sviluppo: collaudo desktop reale e revisione completa dei materiali di distribuzione ancora aperti.

La [procedura di stabilità prolungata](DOCS/it/M0-STABILITY.md) ripete i flussi GUI e registra memoria Go/RSS. Sessione Cocoa aggiornata completata: oltre 30 minuti senza crash o avvisi AX e senza crescita RSS sostenuta osservata. Cadenza irregolare e limiti documentati; collaudo con lettori reali ancora aperto.


## Perché WebFence

Gli strumenti AI permettono anche a sviluppatori singoli di realizzare applicazioni articolate. La velocità di sviluppo richiede verifiche di sicurezza altrettanto accessibili e rigorose. Questa è la motivazione del progetto, non una dimostrazione che il codice generato con AI sia sempre meno sicuro.

WebFence nasce per analizzare siti e API autorizzati, associare ogni problema alle prove raccolte e verificare le correzioni nel tempo. Combinerà controlli tradizionali, informazioni sulle vulnerabilità note e analisi AI facoltativa. La copertura professionale di piattaforme come Invicti/Acunetix è un riferimento di lungo periodo, non una capacità già dimostrata né un'affiliazione.

## Funzionalità previste

| Area | Obiettivo |
| --- | --- |
| Scansione | Esplorazione di siti, API e, in una fase successiva, applicazioni JavaScript e percorsi autenticati. |
| Profili | Budget di richieste, concorrenza, profondità, durata e invasività configurabili separatamente. |
| Motore tradizionale | Regole versionate, controlli riproducibili, evidenze e indicazione della copertura effettiva. |
| Motore AI | Provider remoti con modelli frontier, runtime locali e pacchetti di modelli opzionali, soggetti a risorse e licenze. |
| Vulnerability intelligence | Cache aggiornata di CVE e arricchimenti; distinzione tra corrispondenza di versione e vulnerabilità verificata. |
| Report | Risultati IT/EN con prove, priorità, limiti, rimedi e firma crittografica verificabile. |
| Memoria dei progetti | Inventario e cronologia persistenti fino alla cancellazione; credenziali e prove grezze con conservazione distinta. |
| Convalida fix | Nuovo controllo mirato, confronto con la baseline e ricerca di regressioni nel perimetro riesaminato. |

Tutte queste capacità sono **pianificate**. Le milestone e i criteri di accettazione sono nella [roadmap](DOCS/ROADMAP.md).

## Direzione tecnica

**Go è la scelta raccomandata per il core e l'applicazione desktop nativa**, con Qt Widgets/MIQT scelto per la GUI. La prima installazione sarà pensata per un singolo operatore. SQLite è la proposta per lo storage locale. Browser di scansione e inferenza AI saranno componenti separati, attivati quando necessari; una CLI potrà riutilizzare il core in seguito.

Rust rimane un'opzione per componenti circoscritti se misure reali ne giustificheranno l'introduzione. Il confronto e le condizioni per rivedere la scelta sono nell'[ADR sul linguaggio](DOCS/it/ADR-001-LANGUAGE.md). La modalità desktop è confermata; il toolkit deve ancora superare prove di accessibilità, prestazioni e distribuzione.

## Principi del prodotto

- Scansioni soltanto su sistemi propri o esplicitamente autorizzati, con perimetro e limiti verificati dal motore.
- AI facoltativa: il percorso tradizionale deve funzionare senza LLM; nessun invio a provider esterni per impostazione predefinita.
- Evidenze, ipotesi AI e corrispondenze CVE devono restare distinguibili.
- Un controllo fallito, bloccato o incompleto non può diventare un risultato «sicuro».
- La firma dimostra integrità e provenienza rispetto a una chiave fidata; non certifica l'assenza di vulnerabilità.

## Come orientarsi oggi

1. Leggi la [visione e i requisiti](DOCS/it/PRODUCT.md).
2. Consulta [architettura](DOCS/it/ARCHITECTURE.md), [modello delle minacce](DOCS/it/THREAT-MODEL.md) e [roadmap](DOCS/ROADMAP.md).
3. Per contribuire, parti da [CONTRIBUTING.md](CONTRIBUTING.md). La [guida di sviluppo](DOCS/it/DEVELOPMENT.md) contiene i comandi del prototipo.
4. Le decisioni persistenti sono in [MEMORY.md](MEMORY.md); le istruzioni per gli agenti sono in [AGENTS.md](AGENTS.md).

## Avviare il prototipo

Con Go 1.27.1 e le dipendenze native indicate nella [guida di sviluppo](DOCS/it/DEVELOPMENT.md):

```sh
CGO_CXXFLAGS='-O2 -g -std=c++17' go run ./cmd/webfence
```

Su macOS Apple Silicon si può creare un bundle locale con `sh scripts/package-macos.sh`. È un laboratorio GUI: gli URL `.invalid` sono testo inerte. La build di prova non è un installer firmato/notarizzato.

Piattaforme richieste: macOS Apple Silicon, Windows 10/11 x86-64, Debian e derivati x86-64/ARM64. Il supporto effettivo dipende dalle verifiche della [matrice M0](DOCS/it/QT-DESKTOP.md). La [direzione UX](DOCS/it/UX.md) prevede un desktop tradizionale e sobrio, percorso guidato e strumenti avanzati progressivi.

Materiali delle dipendenze: [raccoglitore sorgenti e avvisi](DOCS/it/M0-SOURCE-MATERIALS.md) verificato su 15 archivi upstream; inclusione facoltativa di 258 avvisi nel bundle macOS prima della firma. Ogni nuovo bundle macOS dichiara inoltre il [minimo OS ricavato dai propri binari](DOCS/it/M0-PACKAGING.md#versione-minima-dichiarata-dallartefatto). La revisione della distribuzione resta aperta.

Raccolti anche [21 pacchetti sorgente Windows](DOCS/it/M0-WINDOWS-SOURCES.md), con ricette collegate ai binari e controlli di integrità documentati; inclusione facoltativa degli archivi originali nello ZIP. La [CI Windows](https://github.com/Matte2599/WebFence/actions/runs/35933571589) verifica la corrispondenza di 74 notices selezionati con gli archivi binari MSYS2; completezza e revisione legale restano aperte.

Un [registro dei checksum pubblicati](DOCS/evidence/windows-binary-checksum-lock-2026-09-24.md) vincola ora le versioni esatte dei 22 pacchetti binari Windows: le modifiche alle dipendenze richiedono riesame esplicito. La [CI `d753f01`](https://github.com/Matte2599/WebFence/actions/runs/35937923381) ha superato sei job al primo tentativo; il registro non prova firme PGP o conformità legale.

La [selezione dei sidecar Windows](DOCS/evidence/windows-attribution-sidecars-2026-09-24.md) include ora 136 file di avviso e attribuzione collegati agli archivi binari, con controllo delle omissioni durante il packaging. La [CI `6e13c95`](https://github.com/Matte2599/WebFence/actions/runs/35939387168) ha superato sei job al primo tentativo; completezza delle licenze e revisione legale restano aperte.

Disponibile anche l’inclusione nel bundle macOS dei [supplementi Homebrew GLib/libb2](DOCS/it/M0-SOURCE-MATERIALS.md#supplementi-homebrew-nel-bundle-macos), vincolati agli hash delle ricette installate.

Verificata localmente e in [CI su macOS 15](https://github.com/Matte2599/WebFence/actions/runs/35916839178) la [ricompilazione e sostituzione del plugin Cocoa](DOCS/it/M0-QT-REPLACEMENT.md) dai materiali inclusi nel bundle, mantenendo la stessa build Go.

Disponibile l’assemblaggio di un [pacchetto sorgenti nativi macOS](DOCS/it/M0-SOURCE-MATERIALS.md#pacchetto-sorgenti-macos-affiancabile-al-bundle) separato, con archivi, patch e hash collegati al bundle.

La documentazione IT/EN è inclusa nei pacchetti di sviluppo; il packaging controlla i collegamenti ai file locali.

## Licenza e autore

Autore e responsabile del progetto: **Matteo Luigi Feroldi** — [Matte2599](https://github.com/Matte2599).

Il progetto adotta la **WebFence Community License 1.0**, una licenza source-available personalizzata. Sono consentiti gratuitamente studio, modifica, fork, uso personale non commerciale e uso da parte di professionisti indipendenti, anche per analisi retribuite e report consegnati ai clienti. La condivisione gratuita segue i limiti di [LICENSE](LICENSE).

L'uso da parte di aziende, la distribuzione del programma alle aziende, la rivendita del software e l'offerta di accesso al programma come servizio richiedono preventiva autorizzazione scritta di Matteo Luigi Feroldi, che può prevedere un corrispettivo o royalties concordate. La consegna di un report professionale a un'azienda non equivale a distribuirle il programma.

Queste restrizioni non sono compatibili con la [Open Source Definition](https://opensource.org/osd). Per questo WebFence non è presentato come software open source OSI. La [documentazione sulle licenze](DOCS/it/LICENSING.md) spiega la scelta e le alternative. Il testo personalizzato richiede revisione legale prima di essere usato come base per contratti commerciali.

Per richieste di licenza, apri una issue con oggetto **Licensing inquiry**, senza dati riservati; l'autore potrà indicare un canale privato. Per vulnerabilità di WebFence, segui [SECURITY.md](SECURITY.md).
