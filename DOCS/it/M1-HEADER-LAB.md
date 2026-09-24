# M1 — Primo controllo HTTP nel laboratorio

[English](../en/M1-HEADER-LAB.md) · [Run gestite](M1-MANAGED-RUNS.md) · [Scansione](SCANNING.md) · [Roadmap](../ROADMAP.md)

`scanner.RunHeaderLab(ctx, store, plan)` è il primo percorso core **progetto salvato → run gestita → trasporto autorizzato → esito**. Usa solo URL seed esplicitamente forniti e grant IP loopback; non è collegato al desktop e non è uno scanner di produzione. Il chiamante deve scegliere percorsi di fixture sicuri: anche GET può avere effetti su un'applicazione.

Il piano richiede ID di progetto, da 1 a 32 URL seed distinti, fino a 32 grant con massimo 16 IP ciascuno, resolver e limiti obbligatori del broker. Il componente copia URL e grant all'ingresso, apre la revisione corrente con `Store.BeginRun`, verifica **tutti** i seed prima di inviare il primo byte e crea `NewAuthorizedLab` con lo scope gestito. Esegue una GET per seed, in sequenza, senza retry. Il broker applica di nuovo autorizzazione, redirect, DNS/IP, TLS, budget e timeout a ogni richiesta. Revoca, scadenza e cancellazione interrompono la run. Un URL fuori scope o duplicato nel piano impedisce ogni traffico della run.

## Regola e risultato

La regola iniziale `HTTP-XCTO-001` revisione 1 osserva `X-Content-Type-Options` su risposte `2xx` con `Content-Type: text/html`. `nosniff` univoco produce `observed`; l'assenza produce `not_observed`; valore ambiguo/non riconosciuto o tipo MIME non interpretabile produce `inconclusive`. Risposte non HTML o con stato non pertinente producono `skipped`. Un errore di trasporto produce `error` e interrompe ulteriori seed. **`not_observed` è un'osservazione, non una vulnerabilità o severità automatica.** Non esistono ancora finding, CVE o raccomandazioni di prodotto da questa regola.

Il `Report` contiene ID progetto, revisione di autorizzazione, numero di seed previsti/completati, tentativi HTTP consumati (redirect inclusi), controlli con indice del seed, codice di evidenza e motivo di arresto redatto. `SeedsComplete` indica soltanto che **tutti i seed espliciti** hanno restituito risposta: non indica copertura del sito o sicurezza. In caso di errore dopo una o più risposte, il report parziale è restituito insieme all'errore. Non vengono conservati URL, query, header grezzi, cookie o body nel risultato. Non c'è ancora persistenza dei report o ripresa dopo crash.

I test usano soltanto server `httptest` su loopback: casi con `nosniff`, assenza, valore ambiguo, MIME non applicabile, MIME invalido, stato 500, budget parziale, seed fuori scope, redirect verso seconda origine non autorizzata, revoca durante una richiesta e modifica della slice del chiamante dopo la verifica. Il race detector esercita il percorso. Nessun target esterno è stato contattato.

Il blocco successivo aggiunge [discovery HTML limitata](M1-DISCOVERY-LAB.md) sulle risposte dei seed, senza visite o invio automatico dei form. Restano per M1: policy di metodi/percorsi/esclusioni, rate limit, broker di produzione, dati di run persistenti e quota/recupero, risultati UI IT/EN e copertura incompleta visibile. Il [piano dei prerequisiti umani](M1-PREREQUISITES.md) resta aperto.
