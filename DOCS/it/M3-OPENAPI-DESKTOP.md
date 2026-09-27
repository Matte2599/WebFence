# M3 — Selezione OpenAPI nel desktop

[English](../en/M3-OPENAPI-DESKTOP.md) · [Import core](M3-OPENAPI-IMPORT.md) · [Roadmap](../ROADMAP.md)

La finestra di scansione Qt Widgets/MIQT offre **Importa OpenAPI JSON** per il progetto selezionato. Legge un file regolare locale di massimo 1 MiB, apre una breve run gestita per verificare l'autorizzazione corrente e usa l'unica origine salvata e i prefissi GET correnti. Il documento non sceglie un server: `servers` e `$ref` non avviano rete né ampliano lo scope. Le route statiche GET ammesse sono mostrate in una scelta non modificabile; solo la route scelta imposta il campo seed. Annullare lascia il seed precedente. La run di import si chiude prima della selezione; la successiva scansione apre una nuova run e ripete i controlli del broker.

Questo pulsante prepara **un seed per la scansione ordinaria**; il [flusso API separato](M3-API-DESKTOP.md) consente poi più GET statiche confermate. L'import offline non esegue HEAD, POST, template, autenticazione OpenAPI o visite. Errori di file, formato, autorizzazione o policy producono un messaggio senza percorso del file o contenuto. Nessuna route importata viene conservata nel database o nel report. La GUI non offre ancora browser o account M3.

**Verifica locale del 27 settembre 2026:** self-test Qt offscreen IT/EN con progetto e file OpenAPI sintetici, `servers` esterno inerte, scelta esplicita di `/app/api`, zero hit durante l'import, rifiuto di JSON ambiguo senza cambiare il seed e successiva scansione controllata. I test Go, i link documentali e la CI del branch/merge vanno valutati per commit. Questo blocco rende usabile l'inventario offline, ma non chiude il crawling API M3.
