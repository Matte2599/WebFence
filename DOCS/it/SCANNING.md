# Motore di scansione e profili

[English](../en/SCANNING.md) · [Indice](../README.md)

Stato: specifica da implementare. Riferimento metodologico: [OWASP WSTG](https://owasp.org/projects/web-security-testing-guide), con identificativi fissati alla versione usata dalle regole.

## Pipeline

`autorizzazione → discovery → piano controlli → esecuzione → prove → normalizzazione → correlazione → report`.

Il piano salva versioni di regole, configurazione, ambiente, credenziali per riferimento e snapshot intelligence. Normalizzare URL senza unire percorsi o parametri che l'applicazione distingue. Il deduplicatore considera origine, metodo, route, posizione del parametro e contesto di autenticazione; non solo l'URL.

Discovery iniziale: link HTTP(S) e form osservati senza inviarli automaticamente. Importazione esplicita di specifiche API, browser JavaScript, flussi di login e identità multiple arrivano in M3. I documenti OpenAPI importati sono input non fidati: i loro server non ampliano lo scope.

## Perimetro

Lo scope include schema, host, porte, eventuali CIDR autorizzati, percorsi, metodi, esclusioni e scadenza. Niente wildcard implicite, estensione automatica ai sottodomini o fiducia in link presenti nel sito. Una risorsa di terzi viene registrata come non coperta; l'eventuale permesso di caricarla come dipendenza non autorizza controlli attivi contro di essa.

Ogni nuova connessione e redirect verifica DNS, IPv4/IPv6, indirizzo effettivamente contattato e policy di uscita, evitando una seconda risoluzione non controllata. Per default bloccare reti private, loopback, link-local e metadata cloud; un laboratorio locale può abilitare esplicitamente host e porte precisi. Endpoint di controllo, store e runtime AI restano esclusi dai target. Le connessioni ai provider e ai feed hanno policy distinte.

## Profili proposti

Valori iniziali per test controllati, da misurare: non sono limiti implementati né garanzie di assenza di effetti.

| Profilo | Operazioni | Concorrenza globale / per origine | Richieste al secondo per origine | Budget richieste / durata |
| --- | --- | --- | --- | --- |
| Osservazione | Analisi di traffico importato, nessun nuovo traffico target | 0 / 0 | 0 | 0 / 15 min |
| Prudente | Discovery e controlli a basso impatto; niente invio form | 2 / 1 | 1 | 500 / 15 min |
| Standard | Regole attive selezionate e identità di test autorizzate | 4 / 2 | 2 | 3.000 / 60 min |
| Approfondito | Copertura e budget maggiori, attivazione esplicita | 8 / 4 | 4 | 10.000 / 120 min |

Limiti iniziali comuni: profondità 5, body decompresso 2 MiB, timeout richiesta 15 s, massimo 5 redirect, massimo 2 retry compatibili. Il conteggio include tentativi, retry, redirect e richieste del browser, non solo URL distinti. Raggiungere un limite produce copertura parziale dichiarata. Budget AI separato.

Profondità e invasività sono assi diversi. Nessun profilo abilita automaticamente cancellazioni, acquisti, invio email, upload persistenti, brute force, denial of service o estrazione di dati. Anche GET può avere effetti: escludere logout e operazioni note, usare staging e account di test. Il prodotto non può promettere di riconoscere tutti gli effetti di endpoint sconosciuti.

## Contratto delle regole

Ogni controllo dichiara ID stabile, revisione, prerequisiti, impatto, budget, target compatibili, prove attese, possibili falsi positivi, CWE/WSTG applicabili e testo IT/EN. Usa soltanto il trasporto controllato. Restituisce uno tra `observed`, `not_observed`, `inconclusive`, `skipped`, `error`, con motivo e riferimenti alle prove.

Prime famiglie: configurazioni TLS/HTTP, cookie, esposizioni e controlli contestuali limitati. XSS, injection e accessi tra identità vengono introdotti con fixture positive e negative e prove minimali, senza promettere copertura di un'intera categoria. L'assenza di un header non implica automaticamente severità alta.

Il circuito di arresto risponde a cancellazione, timeout globale, quota, ripetuti errori del target e segnali di sovraccarico. Niente retry automatico di azioni non ripetibili; annullamento e run parziali conservano un riepilogo redatto.
