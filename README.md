# WebFence

**Analisi della sicurezza web, evidenze verificabili e convalida dei fix.**

[English](README.en.md) · [Documentazione](DOCS/README.md) · [Roadmap](DOCS/ROADMAP.md) · [Licenza](LICENSE)

Il prototipo desktop Go/Qt Widgets usa solo fixture sintetiche: carica 10.000 righe, le filtra e mostra le evidenze in italiano e inglese. Il motore di scansione di produzione non è implementato e non esistono release supportate o benchmark di sicurezza. Per il resoconto di M0 e i gate trasferiti a M1 vedere la [matrice di validazione](DOCS/it/M0-VALIDATION.md); il [piano dei collaudi umani M1](DOCS/it/M1-PREREQUISITES.md) spiega come eseguirli.

Il [primo blocco M1](DOCS/it/M1-PROJECT-AUTHORIZATION.md) aggiunge un modello di progetto e uno snapshot delle origini dichiarate dall'operatore, ancora in memoria e senza rete.

Il [laboratorio M1](DOCS/it/M1-AUTHORIZED-LAB.md) applica quello snapshot al trasporto HTTP solo loopback, anche dopo i redirect e alla scadenza.

Lo [store progetti M1](DOCS/it/M1-PROJECT-STORE.md) conserva i metadati in SQLite, migra gli schemi v1/v2 alla v3 e registra rinnovi e revoche di autorizzazione versionati; la GUI non lo usa ancora.

Le [run gestite M1](DOCS/it/M1-MANAGED-RUNS.md) collegano la revisione corrente alla revoca locale e fermano il laboratorio HTTP quando l'autorizzazione cambia; restano scollegate dalla GUI.

Il [primo controllo HTTP M1](DOCS/it/M1-HEADER-LAB.md) usa run gestite e seed espliciti solo loopback per osservare un header in risposte HTML, con esiti redatti e conteggio dei seed. Non è una scansione di produzione.

La [discovery HTML M1](DOCS/it/M1-DISCOVERY-LAB.md) osserva link e form nelle risposte dei seed con limiti e verifica dello scope; non visita i candidati e non invia form. Il report conserva solo conteggi redatti.

Le [visite HTTP controllate M1](DOCS/it/M1-CONTROLLED-CRAWL.md) aggiungono policy per metodi, percorsi ed esclusioni, link seguiti solo con consenso esplicito, rate per origine e grant di IP pubblici fissati. Sono API del core testate su fixture sintetiche, non ancora disponibili nel desktop; i form non vengono inviati.

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

La tabella descrive obiettivi di prodotto, **non funzionalità completate**; i soli incrementi M1 implementati sono collegati sopra. Le milestone e i criteri di accettazione sono nella [roadmap](DOCS/ROADMAP.md).

## Direzione tecnica

**Go è la scelta per il core e l'applicazione desktop nativa**, con Qt Widgets/MIQT scelto per la GUI. La prima installazione sarà pensata per un singolo operatore. SQLite è scelto per lo storage locale; lo store dei soli metadati di progetto è il primo componente implementato. Browser di scansione e inferenza AI saranno componenti separati, attivati quando necessari; una CLI potrà riutilizzare il core in seguito.

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

Su macOS Apple Silicon si può creare un bundle locale con `sh scripts/package-macos.sh`. È un laboratorio GUI: gli URL `.invalid` sono testo inerte. La build di prova non è un installer firmato/notarizzato.

Piattaforme richieste: macOS Apple Silicon, Windows 10/11 x86-64, Debian e derivati x86-64/ARM64. La [direzione UX](DOCS/it/UX.md) prevede un desktop tradizionale e sobrio, percorso guidato e strumenti avanzati progressivi. Stato delle verifiche, materiali di distribuzione e prerequisiti umani sono nella [matrice M0](DOCS/it/M0-VALIDATION.md).

## Licenza e autore

Autore e responsabile del progetto: **Matteo Luigi Feroldi** — [Matte2599](https://github.com/Matte2599).

Il progetto adotta la **WebFence Community License 1.0**, una licenza source-available personalizzata. Sono consentiti gratuitamente studio, modifica, fork, uso personale non commerciale e uso da parte di professionisti indipendenti, anche per analisi retribuite e report consegnati ai clienti. La condivisione gratuita segue i limiti di [LICENSE](LICENSE).

L'uso da parte di aziende, la distribuzione del programma alle aziende, la rivendita del software e l'offerta di accesso al programma come servizio richiedono preventiva autorizzazione scritta di Matteo Luigi Feroldi, che può prevedere un corrispettivo o royalties concordate. La consegna di un report professionale a un'azienda non equivale a distribuirle il programma.

Queste restrizioni non sono compatibili con la [Open Source Definition](https://opensource.org/osd). Per questo WebFence non è presentato come software open source OSI. La [documentazione sulle licenze](DOCS/it/LICENSING.md) spiega la scelta e le alternative. Il testo personalizzato richiede revisione legale prima di essere usato come base per contratti commerciali.

Per richieste di licenza, apri una issue con oggetto **Licensing inquiry**, senza dati riservati; l'autore potrà indicare un canale privato. Per vulnerabilità di WebFence, segui [SECURITY.md](SECURITY.md).
