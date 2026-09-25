# M0 — Matrice di verifica e resoconto di chiusura

[English](../en/M0-VALIDATION.md) · [Roadmap](../ROADMAP.md) · [Indice](../README.md)

Aggiornato: 2026-09-24. Questa è l'unica fonte per lo stato di M0. La chiusura riguarda il perimetro automatizzabile definito dall'autore, non una release supportata o una certificazione di accessibilità, sicurezza o conformità legale. Le attività che richiedono persone o hardware reale sono prerequisiti di M1; il loro rinvio non equivale a un esito positivo.

**Nota successiva (25 settembre 2026):** il trasferimento a M1 sotto registra la decisione di chiusura M0 del 24 settembre. L'autore ha poi [chiuso M1 sui criteri tecnici e test locali/CI](M1-VALIDATION.md), rinviando le prove umane e la consulenza esterna alle attività di supporto/distribuzione pertinenti. Questa nota non modifica l'esito storico di M0 e non marca tali prove come superate.

## Gate di M0

| ID | Criterio | Esito automatizzabile e prove | Limite trasferito a M1 |
| --- | --- | --- | --- |
| M0-01 | Desktop nativo, tabella ampia, evidenze, IT/EN | **Chiuso per la parte automatizzabile.** Prototipo Qt con 10.000 fixture, filtri, lettura/copia, lingua persistente, focus e tastiera. [AT-SPI su Debian ARM64 in ambiente isolato](../evidence/debian-atspi-2026-09-24.md), [Orca headless](../evidence/debian-orca-headless-2026-09-24.md) e [test Cocoa AX bloccante su macOS 15/26](https://github.com/Matte2599/WebFence/actions/runs/35975333332) passano sui flussi sintetici. La [sessione Cocoa di oltre 30 minuti](M0-STABILITY.md) riguarda una build precedente alla patch AX finale; sulla patch finale risultano self-test, sequenza AX di 30 cicli e soak breve. | Annunci e uso con VoiceOver/NVDA/Orca su desktop reali; passaggio tra monitor con scale diverse. |
| M0-02 | Build e packaging di sviluppo sui target richiesti | **Chiuso per la parte automatizzabile.** Bundle macOS, ZIP Windows x86-64 e `.deb` Debian x86-64/ARM64 con avvio in ambienti isolati; controlli di firme, hash, import, dipendenze, origine e materiali nativi descritti in [ADR-007](ADR-007-PACKAGING.md), [materiali sorgente](M0-SOURCE-MATERIALS.md) e [inventario Windows](../evidence/windows-pe-import-inventory-2026-09-24.md). **Per decisione dell'autore, l'inventario corrente di sorgenti e notices è sufficiente per M0: non sono richiesti ulteriori livelli di verifica.** Questo non approva licenze né distribuzione commerciale. | Prove su Windows 10/11 e Debian reali e sui sistemi minimi dichiarati; valutazione professionale della distribuzione. |
| M0-03 | Scelta GUI, SQLite, JWS, portachiavi | **Completato.** Qt Widgets/MIQT, SQLite, JWS Ed25519 e [adattatore di credenziali native](ADR-005-CREDENTIALS.md) selezionati e provati nei test pertinenti. | Integrazione dei componenti nel prodotto di M1. |
| M0-04 | Versioni, struttura, cataloghi, CI | **Completato.** Modulo Go, cataloghi IT/EN, controlli di formato, test e matrice CI per macOS, Windows e Linux; packaging Debian separato. | Mantenere CI verde nei blocchi successivi. |
| M0-05 | Laboratorio sintetico e primi controlli scope/rete | **Completato.** [Policy di trasporto](ADR-003-TRANSPORT.md), fixture `.invalid`, traffico dei test limitato a loopback, budget e regressioni. | Motore di scansione e policy per target reali appartengono a M1. |
| M0-06 | Canale di sicurezza, testo legale, contributori | **Chiuso per la parte automatizzabile.** Private vulnerability reporting verificato su GitHub; [SECURITY](../../SECURITY.md), [dossier legale](LEGAL-REVIEW.md) e specifica dell'accordo contributori predisposti. | Revisione legale qualificata e accordo contributori approvato prima di aprire i rispettivi processi. |

## Verifica complessiva del blocco di chiusura

Il blocco comprende aggiornamento delle regole di lavoro, matrici IT/EN, collegamenti README/roadmap e CI attivata anche per modifiche Markdown. Sul codice di prodotto invariato rispetto a `51f6219`, i controlli locali del 2026-09-24 hanno superato: 79 test Python (4 saltati), verifica di 93 file Markdown/800 collegamenti locali, `go test ./...`, suite race dei componenti di fondazione, `go vet ./...`, `go mod verify`, due fuzz test da 10 secondi, formato Go e `git diff --check`. Il bundle Apple Silicon ricostruito dal medesimo codice ha superato firma, self-test Cocoa, soak di 12,043 secondi/quattro cicli e client AX nativo per tre cicli 10.000 → 1 → 0 → 10.000. Il soak da 30 minuti avviato sulla build finale è stato **interrotto su richiesta dell'autore** e non è contato come superato. La CI del branch e del commit squash finale va verificata su GitHub; i relativi esiti non richiedono un commit di sola registrazione.

## Richiede intervento umano — prerequisito M1

| Gate | Motivo e prova necessaria |
| --- | --- |
| VoiceOver su macOS | Il client AX e i self-test non dimostrano ciò che il lettore annuncia. Servono navigazione, intestazioni, righe, selezione e lettura delle evidenze con una persona. |
| NVDA su Windows 10/11 | I test Windows usano backend e sessioni automatizzate; manca ascolto/navigazione di un lettore su desktop reale. |
| Orca su Debian o derivati | La prova headless usa uscita audio nulla e registra comandi, non annunci udibili né usabilità su desktop reale. |
| Più monitor con scale diverse | L'hardware locale censito ha un solo display; scaling 150/200% nella stessa finestra non riproduce il trasferimento tra schermi. |
| Windows 10/11 e Debian reali | Runner, container e ambienti isolati verificano pacchetti e flussi sintetici, non installazione/uso su workstation indipendenti. |
| Sistemi minimi | Il minimo macOS deriva dai binari del bundle, ma non è un collaudo su macchina con quel sistema; analogamente mancano prove sulle versioni minime Windows/Debian supportate. |
| Revisione legale | [LICENSE](../../LICENSE), notices e dossier sono predisposti; un professionista deve valutarne efficacia, compatibilità e condizioni di distribuzione. La revisione è ancora da organizzare. |
| Accordo contributori | Esiste una specifica, non un testo approvato e adottato; occorre definire e approvare i diritti prima di accettare contributi esterni che richiedano rilicenza. |

## Procedura per i collaudi umani

Usare solo fixture sintetiche; registrare commit, hash del pacchetto, versione OS, lettore, Qt, scala, configurazione dei monitor ed esito IT/EN. Avviare il pacchetto fuori dalla toolchain e percorrere tastiera, menu, filtro 10.000 → 1 → 0 → 10.000, selezione/copia delle prove, cambio lingua e riavvio. Annotare annunci effettivi, difetti e limiti senza modificare stabilmente le preferenze personali. Nessun sito esterno è un target di questi collaudi.

Il [piano operativo M1](M1-PREREQUISITES.md) contiene schede e criteri per raccogliere gli esiti futuri.
