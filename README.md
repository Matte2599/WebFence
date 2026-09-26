# WebFence

**Analisi della sicurezza web, evidenze verificabili e convalida dei fix.**

[English](README.en.md) · [Documentazione](DOCS/README.md) · [Roadmap](DOCS/ROADMAP.md) · [Licenza](LICENSE)

WebFence è una **alpha desktop Go/Qt Widgets**. Offre un laboratorio di 10.000 esempi sintetici e un primo flusso M1 per progetti autorizzati: visite HTTP(S) GET limitate, discovery HTML, un controllo osservativo di header e risultati redatti salvati localmente. **M1 è chiusa come milestone tecnica** secondo la [matrice M1](DOCS/it/M1-VALIDATION.md); prove su persone e hardware reale non sono state dichiarate superate. Non esistono release supportate, benchmark di sicurezza o equivalenza dimostrata con scanner commerciali. Lo stato M0 resta nella [matrice M0](DOCS/it/M0-VALIDATION.md).

Il [primo blocco M1](DOCS/it/M1-PROJECT-AUTHORIZATION.md) aggiunge un modello di progetto e uno snapshot delle origini dichiarate dall'operatore, ancora in memoria e senza rete.

Il [laboratorio M1](DOCS/it/M1-AUTHORIZED-LAB.md) applica quello snapshot al trasporto HTTP solo loopback, anche dopo i redirect e alla scadenza.

Lo [store progetti M1](DOCS/it/M1-PROJECT-STORE.md) conserva dichiarazioni versionate, run e osservazioni redatte in SQLite v4; gestisce riavvio, quota, backup manuale ed eliminazione logica. La finestra M1 lo usa per progetti e risultati.

Le [run gestite M1](DOCS/it/M1-MANAGED-RUNS.md) collegano la revisione corrente alla revoca locale e fermano il laboratorio HTTP quando l'autorizzazione cambia.

Il [primo controllo HTTP M1](DOCS/it/M1-HEADER-LAB.md) usa run gestite e seed espliciti solo loopback per osservare un header in risposte HTML, con esiti redatti e conteggio dei seed. Non è una scansione di produzione.

La [discovery HTML M1](DOCS/it/M1-DISCOVERY-LAB.md) osserva link e form nelle risposte dei seed con limiti e verifica dello scope; non visita i candidati e non invia form. Il report conserva solo conteggi redatti.

Le [visite HTTP controllate M1](DOCS/it/M1-CONTROLLED-CRAWL.md) aggiungono policy per metodi, percorsi ed esclusioni, link seguiti solo con opt-in, rate per origine e grant di IP pubblici fissati. La GUI espone un'origine e un seed per scansione; i form non vengono inviati. Usa solo target propri o esplicitamente autorizzati.

La [alpha M2](DOCS/it/M2-VALIDATION.md) aggiunge cache CVE/NVD aggiornata su richiesta, correlazione conservativa con prodotto/versione dichiarati dall'operatore e bundle di report IT/EN verificabili offline. Il [desktop](DOCS/it/M2-GUI-EXTENSION.md) espone identità CPE, attestazioni esplicite e gestione locale delle chiavi per la run selezionata; la [simulazione](DOCS/it/M2-ACCURACY-SIMULATION.md) non misura accuratezza su CVE reali e non è stato eseguito sync live delle fonti.

**Qt è stato scelto dall’autore** dopo il [confronto pratico](DOCS/it/GUI-COMPARISON.md). Il comando principale usa Qt Widgets tramite MIQT; [ADR accettato](DOCS/it/ADR-002-GUI.md). Le scelte tecniche e i limiti del prototipo sono nella [guida di sviluppo](DOCS/it/DEVELOPMENT.md).

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

La tabella descrive obiettivi di prodotto, **non funzionalità completate**; le capacità attuali e i limiti sono nelle matrici [M1](DOCS/it/M1-VALIDATION.md) e [M2](DOCS/it/M2-VALIDATION.md). Le milestone e i criteri di accettazione sono nella [roadmap](DOCS/ROADMAP.md).

## Direzione tecnica

**Go è la scelta per il core e l'applicazione desktop nativa**, con Qt Widgets/MIQT scelto per la GUI. La alpha è pensata per un singolo operatore e salva progetti e osservazioni redatte in SQLite. Browser di scansione e inferenza AI saranno componenti separati, attivati quando necessari; una CLI potrà riutilizzare il core in seguito.

Rust rimane un'opzione per componenti circoscritti se misure reali ne giustificheranno l'introduzione. Il confronto e le condizioni per rivedere la scelta sono nell'[ADR sul linguaggio](DOCS/it/ADR-001-LANGUAGE.md). La modalità desktop è confermata; per le verifiche del toolkit consultare la [matrice M0](DOCS/it/M0-VALIDATION.md).

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

Su macOS Apple Silicon si può creare un bundle locale con `sh scripts/package-macos.sh`. Gli URL `.invalid` degli esempi M0 sono testo inerte; **Scansione M1** apre invece rete solo per i target configurati e autorizzati dall'operatore. La build di prova non è un installer firmato/notarizzato.

Piattaforme target M1: macOS 26+ Apple Silicon, Windows 10 1809+/11 x86-64 e Ubuntu 24.04 LTS x86-64/ARM64; altre Debian/derivate restano un obiettivo di compatibilità. La [policy di supporto](DOCS/it/M1-SUPPORT-POLICY.md) distingue target e verifiche. La [direzione UX](DOCS/it/UX.md) prevede un desktop tradizionale e sobrio, percorso guidato e strumenti avanzati progressivi.

## Licenza e autore

Autore e responsabile del progetto: **Matteo Luigi Feroldi** — [Matte2599](https://github.com/Matte2599).

Il progetto adotta la **WebFence Community License 1.0**, una licenza source-available personalizzata. Sono consentiti gratuitamente studio, modifica, fork, uso personale non commerciale e uso da parte di professionisti indipendenti, anche per analisi retribuite e report consegnati ai clienti. La condivisione gratuita segue i limiti di [LICENSE](LICENSE).

L'uso da parte di aziende, la distribuzione del programma alle aziende, la rivendita del software e l'offerta di accesso al programma come servizio richiedono preventiva autorizzazione scritta di Matteo Luigi Feroldi, che può prevedere un corrispettivo o royalties concordate. La consegna di un report professionale a un'azienda non equivale a distribuirle il programma.

Queste restrizioni non sono compatibili con la [Open Source Definition](https://opensource.org/osd). Per questo WebFence non è presentato come software open source OSI. La [documentazione sulle licenze](DOCS/it/LICENSING.md) spiega la scelta e le alternative; un'[analisi interna](DOCS/it/M1-LEGAL-ASSESSMENT.md) valuta il testo attuale senza sostituire il parere professionale necessario per contratti e distribuzioni pertinenti.

Per richieste di licenza, apri una issue con oggetto **Licensing inquiry**, senza dati riservati; l'autore potrà indicare un canale privato. Per vulnerabilità di WebFence, segui [SECURITY.md](SECURITY.md).
