# M0 — Primo prototipo desktop

[English](../en/M0-DESKTOP.md) · [Indice](../README.md)

Data: 2026-09-23. Primo task M0: fondazioni Go e laboratorio GUI offline. **M0 non è conclusa.**

## Funzioni implementate

- Finestra Go/Fyne con tabella virtualizzata di 10.000 esempi, caricata esplicitamente dall'utente.
- Filtri combinabili per ID/URL e severità sintetica; dettaglio del record selezionato, precedente/successivo, copia delle prove.
- Prove mostrate come testo semplice: anche il frammento `<script>` della fixture rimane inerte. URL riservati `example.invalid`, mai contattati.
- Cataloghi IT/EN incorporati, selezione iniziale dalla lingua del sistema con fallback inglese e preferenza salvata. Cambiare lingua non modifica ID, filtro canonico o prove.
- Stato vuoto e nessuna corrispondenza distinti; svuotare o filtrare fuori la selezione elimina anche le prove visualizzate, evitando dettagli obsoleti.
- Modulo Go e dipendenze fissate, test senza display, workflow CI e script bundle locale macOS.

Non sono presenti scanner, scope/rete, progetti persistenti, keychain, CVE, report firmati, browser o AI. Le severità sono dati per provare la GUI, non valutazioni di sicurezza. La copia avviene soltanto su comando dell'utente; svuotare la GUI non cancella gli appunti.

## Piattaforme richieste e verifica

| Piattaforma richiesta | Prova di questo task | Limite |
| --- | --- | --- |
| macOS Apple Silicon / ARM64 | Build e bundle su macOS 26.6.2, Xcode 27, Go 1.27.1 | Versione minima, firma distribuzione e notarizzazione non validate |
| Windows 10/11 x86-64 | Workflow di build su Windows Server 2022 x64 predisposto | Non equivale a esecuzione della GUI su Windows 10/11 |
| Debian/derivati x86-64 | Workflow di build su Ubuntu 24.04 predisposto | Debian e versioni minime non provate |
| Debian/derivati ARM64 | Workflow di build su Ubuntu 24.04 ARM64 predisposto | Hardware e desktop Linux non provati |

Nessun supporto a macOS Intel o architetture a 32 bit richiesto. La matrice è un piano di supporto, non una certificazione. Gli esiti delle esecuzioni remote sono nella [pagina Actions](https://github.com/Matte2599/WebFence/actions); la presenza del workflow non prova che una run sia passata.

## Verifiche locali

Go 1.27.1, Fyne 2.8.1, host macOS 26.6.2 ARM64:

- `go test -tags ci -race ./...`: superato. Test di filtro/identità, cataloghi/fallback, flusso GUI con selezione dell'ultima riga, lingua, copia, stato vuoto e navigazione.
- `go vet -tags ci ./...`: superato.
- Build nativa e `sh scripts/package-macos.sh`: superate; plist valido. Il linker emette un warning su `-lobjc` duplicato senza impedire la build.
- Finestra nativa aperta: verificati caricamento delle 10.000 righe, selezione con prove, cambio IT/EN e filtro su `DEMO-10000`. Questo non è un benchmark di latenza o consumo memoria.
- Il driver di test software verifica le interazioni in-process, non l'accessibilità nativa.

I dati di test sono sintetici; nessun target remoto è stato analizzato. I download della toolchain Go e delle dipendenze sono distinti dal comportamento offline dell'applicazione.

## Accessibilità: gate non superato

La build normale espone all'ispezione macOS la finestra e i menu, ma non i widget. Il tag sperimentale Fyne `accessibility` espone parte delle etichette e dei pulsanti; nella prova i selettori, la tabella e il contenuto del pannello prove mancavano dall'albero, con gruppi duplicati. L'attivazione del pulsante tramite albero non ha caricato le righe e una successiva interazione dello strumento è terminata in timeout. Il timeout da solo non dimostra un crash dell'applicazione.

La lettura del sorgente Fyne 2.8.1 conferma che il bridge macOS visita gli oggetti `fyne.Accessible` e ricorre nei `fyne.Container`; non basta abilitare il tag per rendere accessibili tutti i widget composti. Su Linux questa versione non include il medesimo bridge nativo. Non sono state eseguite prove complete con VoiceOver, NVDA o Orca.

Il bundle normale rimane utile per prove visive e funzionali, **non soddisfa il requisito di accessibilità del prodotto**. Il tag è facoltativo nello script solo per riprodurre l'indagine. Non aggirare il gate dichiarando completata M0 o rendendo permanenti componenti GUI non ancora convalidati.

Prossimo task: verificare se il limite è risolvibile con API pubbliche e supportate; altrimenti confrontare un toolkit desktop con supporto assistivo adeguato sullo stesso scenario (10.000 righe, filtri, prove selezionabili, IT/EN e quattro target). Registrare l'esito nell'ADR prima di sostituire Fyne. Go resta la scelta del core; una GUI web non è una sostituzione autorizzata.

## Lavoro M0 residuo

Accessibilità e scelta finale GUI; prove DPI e testo lungo; packaging sulle altre piattaforme e versioni minime; selezione e prove SQLite/JWS/keychain; laboratorio isolato e policy di rete. I task legali, CLA e canale privato restano distinti dalle verifiche tecniche. La [specifica UX](UX.md) guida il design successivo.

Fonti tecniche: [Fyne 2.8.1](https://github.com/fyne-io/fyne/releases/tag/v2.8.1), [bridge macOS](https://github.com/fyne-io/fyne/blob/v2.8.1/internal/driver/glfw/accessibility_darwin.go), [build senza bridge](https://github.com/fyne-io/fyne/blob/v2.8.1/internal/driver/glfw/accessibility_notdarwin.go), [runner GitHub](https://docs.github.com/en/actions/reference/runners/github-hosted-runners). Le osservazioni WebFence sono prove locali, non garanzie attribuite a queste fonti.
