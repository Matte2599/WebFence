# Qualità, benchmark e criteri di rilascio

[English](../en/QUALITY.md) · [Indice](../README.md)

Stato: piano di qualità del motore, senza benchmark di scansione. I primi test della GUI offline e i limiti di accessibilità sono nel [resoconto M0](M0-DESKTOP.md). L'obiettivo di copertura professionale non viene trasformato in una dichiarazione di equivalenza a prodotti commerciali.

## Corpus

Laboratori propri o espressamente autorizzati, riproducibili e isolati. Per ogni regola introdurre caso vulnerabile, caso corretto, caso non applicabile e caso ambiguo. Aggiungere redirect, DNS variabile, IPv6, autenticazione scaduta, WAF simulato, body compressi, paginazione infinita e limiti di rete. Le fixture non devono contattare sistemi pubblici durante i test.

Separare sviluppo e valutazione per applicazione/famiglia di template, non per URL della stessa applicazione. Versionare dati, etichette, regole, motore e hardware. Per l'AI conservare modello, prompt, parametri, costi e output; ripetere i casi instabili e non usare lo stesso LLM come unica fonte delle etichette.

## Misure

| Misura | Definizione |
| --- | --- |
| Precisione | `TP / (TP + FP)`, per famiglia e livello di verifica |
| Recall | `TP / (TP + FN)` rispetto ai casi noti del corpus, non a tutte le vulnerabilità reali |
| Copertura | Endpoint/contesti scoperti e controlli applicabili eseguiti; mostrare numeratore e denominatore |
| Stabilità | Esiti discordanti tra ripetizioni e cause |
| Costo | Richieste, durata, RAM, CPU, disco, eventuali token e spesa AI |
| Retest | Fix confermati correttamente, falsi «risolti» e inconcludenti |

Denominatore zero significa «non calcolabile». I controlli bloccati non entrano nei veri negativi. Deduplicare secondo una regola pubblicata prima di calcolare le metriche. Riportare numerosità e incertezza; una percentuale su pochi casi non è un benchmark affidabile.

## Gate proposti

- Ogni controllo ha test positivi e negativi e una motivazione verificabile per severità/confidenza.
- Nessun caso della suite di scope, isolamento, segreti, import e firma viola il relativo confine; ogni fallimento blocca il rilascio interessato.
- Nessun falso `fixed_verified` nei casi di errore/ambiguità del corpus di retest.
- Target iniziale di precisione almeno 95% per finding etichettati ad alta confidenza, per famiglia su corpus dichiarato; pubblicare anche numerosità e intervallo d'incertezza. Non è una prestazione raggiunta.
- Il benchmark di recall viene pubblicato per famiglia; una regola non viene promossa nascondendo casi non coperti o riducendo artificiosamente il denominatore.
- Cancellazione: obiettivo interrompere nuovi invii entro 1 secondo; richieste già in volo rispettano cancellazione e timeout configurati. Va misurato per core e browser.
- Interfaccia responsive su corpus ampio, navigabile da tastiera, leggibile con DPI diversi; tecnologie assistive provate sui sistemi dichiarati supportati.
- IT/EN completi nelle funzioni rilasciate, firme verificate anche dopo rendering e localizzazione.

Questi gate sono criteri di progetto, non garanzie matematiche di sicurezza o metriche già ottenute.

## Livelli di verifica futuri

Unità per scope, version matching, fingerprint e stati. Integrazione per trasporto, SQLite, crash/recovery, keychain e contratti provider. Fuzzing per parser URL, import, manifest e decodifica. Test end-to-end desktop per scansione, cancellazione, riavvio, export, retest e cancellazione dati. Race detector e analisi delle dipendenze per il codice Go.

Confronti con Invicti/Acunetix solo con licenza e configurazioni comparabili, stesse applicazioni, tempo, credenziali e regole di conteggio; rispettare i termini del benchmark. Non presentare dati di terzi come misurazioni WebFence.
