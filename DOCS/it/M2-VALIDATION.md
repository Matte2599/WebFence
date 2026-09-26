# M2 — Verifica dell'alpha intelligence e report

[English](../en/M2-VALIDATION.md) · [Roadmap](../ROADMAP.md) · [Sviluppo](DEVELOPMENT.md)

Stato tecnico: i quattro criteri M2 sono implementati nel core e nel desktop Qt. La chiusura richiede anche la CI verde del branch e del commit integrato su `main`; gli esiti vanno letti nei run GitHub del commit pertinente, senza dedurli da test locali. Le prove qui descritte sono sintetiche e non stabiliscono qualità di matching su un corpus reale, supporto per tutte le versioni minime, accessibilità certificata o prontezza di distribuzione.

| Criterio | Prova tecnica e limite |
| --- | --- |
| Adattatori CVE/NVD e cache | `internal/intelligence` sincronizza finestre NVD esplicite e CVE per ID in SQLite separato, con provenienza, hash, watermark e rollback transazionale. La GUI apre rete verso le sole fonti ufficiali su azione esplicita. Nessun catalogo è incluso o scaricato all'avvio; lo stato `fresh/stale` usa una soglia locale di 24 ore, non promette completezza. |
| Matching prudente | `Assess` richiede vendor/prodotto/versione e riferimenti forniti dall'operatore; `candidate`, `applicable`, `verified`, `not_applicable`, `unknown` restano distinti. La GUI offre inventario/manuale/banner e confidenza, ma non trasforma un header M1 in CVE né espone ancora i form per attestazioni backport/verifica. Le correlazioni della GUI vivono nella sessione per la run selezionata fino all'export. |
| JSON/HTML, firma e verifica | Il bundle `.wfr` include artefatti IT/EN statici, manifest JCS con hash e JWS Ed25519; la chiave privata usa il portachiavi nativo. La GUI crea una chiave operatore, esporta signed/unsigned e verifica con il registro di fiducia locale. Il CLI tecnico permette anche rotazione, revoca, export/import di descrittori pubblici con impronta confermata separatamente. Il verificatore offline non consulta revoche correnti né timestamp fidati. |
| Regressioni di sicurezza | Fixture locali provano feed indisponibile e sync interrotta senza perdere l'ultimo snapshot valido; versioni ambigue restano `unknown`. Bundle alterati, file inattesi, firme errate e chiavi sconosciute/revocate vengono respinti. L'autotest Qt percorre una run loopback, un record CVE sintetico recuperato da un server locale, matching esplicito ed export bilingue non firmato. |

Il report conserva la revisione di autorizzazione della run e solo il ledger redatto. La copertura del sito non è attestata: mostra il minimo noto di seed non eseguiti e dichiara sconosciute le altre aree. NVD e CVE hanno stati separati nel report, con modalità offline distinta dalla freschezza al momento dell'export. L'export CLI senza snapshot cache li indica `unavailable`, senza dichiarare zero CVE.

## Test completo M2

Il controllo locale dell'intera milestone usa `go mod verify`, `go test -race` su tutti i pacchetti core, `go test ./...`, `go vet ./...`, build dei tre comandi, autotest Qt offscreen, soak test, test Python di supporto, coerenza IT/EN, link relativi e `git diff --check`. La CI aggiunge build, autotest e packaging di sviluppo su macOS, Windows e Ubuntu ARM64/x86-64. L'autotest non sostituisce VoiceOver, NVDA, Orca o prove hardware reali.

**Prova locale del 26 settembre 2026, Apple Silicon/macOS 26.6.2:** i controlli sopra sono passati; anche `go test -race ./internal/desktop` con i flag C++17, il self-test Qt con la correlazione CVE sintetica e il soak di 10 secondi (quattro cicli) sono passati. La suite Python ha eseguito 79 test, con quattro salti legati all'ambiente. Il risultato del branch e del merge finale è distinto da questa prova locale e si verifica nei run CI corrispondenti.

Non sono stati eseguiti sync reali NVD/CVE né scansioni su target esterni per chiudere M2. La fonte e il sito analizzato sono confini di rete distinti: il feed non riceve inventario del progetto. L'accuratezza CVE, la disponibilità live degli endpoint, il feedback sul carico dei target, i limiti assistivi M0/M1 e la revisione legale professionale restano rischi per supporto e distribuzione, non esiti implicitamente superati da M2.
