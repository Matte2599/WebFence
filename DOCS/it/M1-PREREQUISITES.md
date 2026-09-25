# M1 — Piano dei prerequisiti con intervento umano

[English](../en/M1-PREREQUISITES.md) · [Matrice M0](M0-VALIDATION.md) · [Roadmap](../ROADMAP.md)

Questo piano rende eseguibili i gate trasferiti da M0 a M1. Lo stato di **M0** rimane esclusivamente nella [matrice M0](M0-VALIDATION.md). Una build CI, un test AX/AT-SPI o un container non sostituiscono un giudizio di una persona sul desktop richiesto. Il [preflight locale del 2026-09-24](../evidence/m1-prerequisite-preflight-2026-09-24.md) registra soltanto le verifiche effettivamente eseguite.

## Matrice di esecuzione

| ID | Prerequisito | Ambiente e responsabile della prova | Criterio per chiudere il gate |
| --- | --- | --- | --- |
| M1-H01 | VoiceOver | Mac Apple Silicon reale; persona che ascolta e usa VoiceOver | Scheda IT ed EN con annunci effettivi, navigazione e lettura delle evidenze; difetti bloccanti risolti o limite esplicitamente accettato. |
| M1-H02 | NVDA | Windows 10/11 x86-64 reale; persona che ascolta e usa NVDA | Stessa prova IT/EN su ZIP estratto, avviato fuori da MSYS2; versione di Windows e NVDA registrate. |
| M1-H03 | Orca | Desktop Debian/derivato reale, x86-64 o ARM64; persona che ascolta e usa Orca | Stessa prova IT/EN su `.deb` installato; sessione X11/Wayland e versione di Orca registrate. |
| M1-H04 | Monitor misti | Desktop reale con due monitor attivi, uno con scala diversa; operatore | Trasferimento della finestra tra gli schermi in entrambe le direzioni, con focus, tabella, menu, popup e testo leggibili in IT/EN; scale e risoluzioni registrate. |
| M1-H05 | Workstation supportate | Windows 10 x86-64, Windows 11 x86-64, Debian/derivato x86-64 e ARM64 reali, anche tramite operatori diversi | Per ciascuna combinazione: pacchetto ottenuto e verificato, installazione/estrazione, avvio fuori dalla toolchain, percorso sintetico IT/EN, chiusura e riavvio. Registrare separatamente successi e problemi. |
| M1-H06 | Versioni minime | Mac, Windows e Debian sulle versioni minime **dichiarate** dopo decisione sul supporto | Versione minima per ogni piattaforma/architettura fissata, binari e dipendenze compatibili, esecuzione del percorso sintetico su quel sistema reale. Una versione più recente non basta. |
| M1-H07 | Revisione legale | Consulente legale qualificato incaricato dall'autore | Parere su versioni/hash precisi di LICENSE, notices/pacchetti e [dossier](LEGAL-REVIEW.md); correzioni concordate applicate; approvazione documentata dall'autore prima della distribuzione pertinente. |
| M1-H08 | Accordo contributori | Autore e consulente; poi processo di accettazione | Testo per titolari individuali e aziendali revisionato/approvato, versione e accettazione verificabili prima di integrare contributi sostanziali che richiedono rilicenza. |

**Stato iniziale:** procedure preparate, nessun M1-H01…H08 superato per effetto di questo documento. H01 è praticabile sull'attuale Mac solo con un ascoltatore; H04 richiede un secondo monitor. H02/H03/H05 richiedono workstation e operatori reali. Per H06 l'autore ha scelto **macOS 26** come obiettivo minimo temporaneo su Apple Silicon; il bundle corrente dichiara 26.0.0, ma resta da provarlo su 26.0 reale. I minimi Windows/Debian vanno ancora decisi e provati. H07/H08 richiedono un professionista e l'approvazione dell'autore.

## Preparazione comune

1. Annotare commit sorgente, nome e SHA-256 dell'artefatto **effettivamente provato**, architettura, versione completa di OS/Qt/lettore e tipo di sessione. Per macOS registrare gli hash dell'eseguibile e del plugin Cocoa del bundle; per Windows dello ZIP; per Debian del `.deb`. Non riutilizzare l'hash di una build precedente.
2. Usare una workstation reale indipendente dal runner CI, un account di prova senza dati cliente e un pacchetto di sviluppo. Prima di dichiarare «installazione su host pulito», registrare eventuali Qt, MSYS2, Go o altre dipendenze già installate: se incidono sull'avvio, l'esito è limitato.
3. Avviare WebFence dall'artefatto installato/estratto, non da `go run` o da un eseguibile sciolto. Usare le 10.000 fixture sintetiche incluse e, per la nuova finestra M1, soltanto un server loopback posseduto dall'operatore se disponibile; non configurare target esterni.
4. Salvare privatamente note, eventuali registrazioni e schermate. Nel repository pubblico inserire solo una sintesi redatta con hash/versioni, esito e difetti riproducibili; niente nomi di tester, voci, dati personali o preferenze dell'account.

### Handoff per il PC Windows 10 disponibile

L'autore può eseguire una prova guidata su Windows 10. Prima occorrono **build e numero di versione Windows** (`winver` oppure PowerShell: `(Get-CimInstance Win32_OperatingSystem) | Select-Object Caption,Version,BuildNumber`) e uno ZIP di sviluppo costruito dal commit da verificare; la CI attuale testa lo ZIP ma non lo pubblica come artefatto scaricabile. Prepararlo con la [toolchain UCRT64 e la procedura Windows](DEVELOPMENT.md#avvio-e-controlli) / [comandi di packaging](ADR-007-PACKAGING.md#comandi-presenti), oppure trasferire privatamente un ZIP di sviluppo con hash concordato. Non trattare il solo `.exe` come pacchetto. Dopo l'estrazione, in PowerShell annotare `(Get-FileHash -Algorithm SHA256 .\webfence-windows-amd64.zip).Hash`, avviare `WebFence\webfence.exe`, completare il percorso IT/EN qui sotto e riavviare. Per verificare il pacchetto automaticamente, PowerShell 7 può eseguire `./scripts/test-windows-package.ps1 -Archive ./dist/webfence-windows-amd64.zip` dal checkout corrispondente; non sostituisce l'uso desktop. Non installare NVDA o modificare impostazioni assistive solo per produrre un falso esito H02: quel gate resta aperto finché una persona può ascoltarlo.

## Percorso desktop IT/EN

Ripetere **per ogni lingua**. Annotare l'annuncio reale del lettore, non solo l'etichetta attesa; distinguere «non annunciato», «annunciato ma ambiguo» e «non raggiungibile».

1. Aprire da tastiera menu e controlli; verificare ordine di focus, indicatore visivo e possibilità di tornare alla tabella senza mouse.
2. Caricare 10.000 esempi. Raggiungere intestazioni, prima e ultima riga, quattro celle, selezione e pannello evidenze; leggere ID, severità, titolo e prova. Confermare che il lettore non usa riferimenti di riga/cella invalidi dopo l'azione.
3. Cercare `DEMO-10000` (una riga), poi `no-such-fixture` (zero), poi svuotare la ricerca (10.000). Ripetere lettura e selezione dopo ogni passaggio; registrare conteggio visualizzato e annunci. Questa sequenza copre il difetto AX storico.
4. Copiare **solo** la prova sintetica esplicitamente e verificare il risultato. Aprire e chiudere dettagli avanzati; verificare testo lungo e assenza di contenuto di una selezione precedente quando il risultato è vuoto.
5. Cambiare lingua tramite menu/scorciatoia, chiudere e riaprire. Verificare traduzione, preferenza salvata e identità invariata degli ID. Ripristinare la preferenza originale dell'operatore dopo la prova.
6. Per M1-H04 spostare la finestra, con risultati e popup aperti, dal monitor A a B e viceversa; ripetere focus, righe e menu a ciascuna scala. Registrare se il cambio impone riavvio o produce clipping, sfocatura o perdita di focus.
7. Aprire **Scansione M1** da tastiera. Verificare etichette, conferma autorizzazione non preselezionata, campi e pulsanti, avviso di copertura, risultato redatto e testo IT/EN. Una prova di scansione richiede un server loopback di cui l'operatore controlla l'avvio; non usare siti esterni per questo gate. Dopo riavvio, verificare il risultato salvato e poi eliminare il progetto di prova confermando la finestra di dialogo.

Un gate assistivo passa solo se una persona ha realmente **ascoltato** gli annunci e completato il percorso con il lettore attivo. Un test automatico senza audio resta supporto diagnostico. Se emerge un difetto, annotare riproduzione, ambiente, gravità e retest sul pacchetto corretto; non segnare «superato» perché il test è stato eseguito.

## Scheda di prova da copiare per ogni ambiente

Conservare la scheda operativa fuori dal repository se contiene dati personali; pubblicare una versione redatta. Ogni riga `passa` richiede osservazione diretta.

```text
Gate ID:                    Stato: da eseguire | passa | non passa | bloccato
Data (UTC):                 Commit sorgente:
Artefatto / SHA-256:        Qt / MIQT:
Macchina/architettura:      OS e build:
Sessione/display/scale:     Lettore e versione (o nessuno):
Operatore: persona, identificativo privato; non pubblicare il nome
IT: avvio / tastiera / 10000→1→0→10000 / evidenze / copia / lingua / riavvio:
EN: avvio / tastiera / 10000→1→0→10000 / evidenze / copia / lingua / riavvio:
Annunci osservati (H01-H03):
Monitor A→B→A (H04):
Dipendenze già presenti / limiti dell'host:
Difetti e riferimenti al retest:
Motivo del blocco o criterio di accettazione:
Esito attestato dall'operatore (conservato privatamente):
```

Per H05 produrre quattro schede distinte, una per ogni combinazione indicata. H06 richiede una scheda ulteriore per ogni versione minima deliberata, anche se coincide con una prova H05: collegare le due, senza duplicare evidenze. macOS 26 è l'obiettivo deliberato, ancora da provare sulla versione 26.0; i minimi Windows 10 e Debian restano decisioni di prodotto. Non dedurre una prova reale dalla matrice upstream Qt o dai metadata del bundle.

## Tracciato legale e contributi

Per H07 consegnare al consulente il [dossier](LEGAL-REVIEW.md), LICENSE IT/EN, CONTRIBUTING e un inventario degli artefatti realmente previsti per la distribuzione. Registrare privatamente incarico, giurisdizioni/usi esaminati, parere e modifiche; nel registro pubblico solo versione/hash dei testi, data, decisione dell'autore e questioni aperte. H08 usa la [specifica nel dossier](LEGAL-REVIEW.md#specifica-dellaccordo-contributori-da-redigere); un modello o una casella PR non costituiscono accettazione. Finché H07/H08 non sono chiusi, non dichiarare approvati distribuzione e rilicenza né integrare contributi sostanziali destinati a rilicenza.

Questi gate non impediscono di continuare sviluppo e test **locali** di M1 con fixture sintetiche; sono condizioni prima delle dichiarazioni e attività indicate, e rimangono visibili finché completati.
