# M2 — Funzioni avanzate nel desktop

[English](../en/M2-GUI-EXTENSION.md) · [Validazione M2](M2-VALIDATION.md) · [Matching](M2-MATCHING.md) · [Report](M2-REPORTS.md)

Il dialogo **CVE e report** della run selezionata ha tre schede: **Fonti CVE**, **Correlazione**, **Report e chiavi**. L'aggiornamento della cache resta manuale e le attestazioni sono dichiarazioni dell'operatore, non controlli CVE automatici.

## Correlazione

- Il segnale usa produttore, prodotto, versione, metodo, confidenza e riferimento non segreto. Per NVD senza CPE completo va indicato anche il tipo `applicazione`/`sistema operativo`/`hardware`. Un CPE 2.3 completo opzionale sostituisce i campi separati di identità; la versione deve coincidere con quella del CPE.
- Il form del backport richiede conferma esplicita, advisory HTTPS e riferimento valido. Il form di verifica richiede conferma, metodo manuale o controllo sicuro **già eseguito** e riferimento valido. Le due attestazioni non possono essere confermate insieme. Nessun advisory viene aperto dall'app.
- Quando cambiano CVE, fonte, run, identità/versione o qualità del segnale, le attestazioni nel form vengono cancellate. Questo evita che una conferma resti applicata a un altro record. Le correlazioni già salvate per la run nel dialogo restano in memoria solo fino alla chiusura della sessione o al cambio run.
- Una verifica tentata con dati non validi rimane `unknown` nel core, invece di essere presentata come applicabile senza spiegazione. La GUI usa descrizioni IT/EN per stati e motivi, mantenendo codici stabili nel report.

## Report e chiavi

La GUI crea chiavi private nel portachiavi nativo, esporta report firmati o esplicitamente non firmati e verifica bundle offline. Può anche **ruotare** una chiave attiva, **revocare** una chiave, esportare un descrittore pubblico JSON e importarne uno con impronta SHA-256 ottenuta da un canale indipendente. Rotazione e revoca chiedono conferma. Una chiave pubblica importata può verificare report, ma non firma senza la corrispondente chiave privata locale. Il registro di fiducia non è un servizio di revoca aggiornato online.

## Verifica locale aggiuntiva

L'autotest Qt usa una run e due record CVE/NVD sintetici su loopback. Esercita entrambi i modi di identità NVD, attestazioni valide e in conflitto, reset dei campi obsoleti, export bilingue non firmato e firmato, verifica offline, generazione/rotazione/revoca con un portachiavi sintetico isolato, import con impronta errata e corretta e conservazione della chiave selezionata al cambio lingua. `go test -race` sul core, `go test ./...`, `go vet ./...`, test Python e soak restano regressioni generali.

La resa è stata osservata su macOS Cocoa nativo e Qt offscreen in IT/EN: tre schede, inizio/fine di moduli scorrevoli e finestra normale **820×740** e compatta **620×560** punti logici. Le schermate sintetiche locali sono state ispezionate per sovrapposizioni, controlli irraggiungibili e tagli del testo; la politica di crescita e ritorno a capo del form elimina il campo troppo stretto visto inizialmente su Cocoa. Questo controllo non certifica assenza di difetti su Windows/Linux, ogni scala DPI, temi o tecnologie assistive; la CI verifica i flussi nativi sulle piattaforme configurate.

In una ripetizione Cocoa locale, gli assert di focus/tastiera preesistenti della schermata M0 sono risultati intermittenti mentre il primo piano macOS cambiava; una ripetizione senza contesa è passata. Il flusso M2 è rimasto verde. Non si considera questa oscillazione una prova di focus affidabile in uso reale.
