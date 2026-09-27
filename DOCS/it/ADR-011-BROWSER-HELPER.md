# ADR-011 — Helper Qt supervisionato per il laboratorio M3

[English](../en/ADR-011-BROWSER-HELPER.md) · [Laboratorio](M3-BROWSER-LAB.md) · [ADR-010](ADR-010-QT-BROWSER-LAB.md) · [Roadmap](../ROADMAP.md)

Data: 2026-09-27. Stato: **adottato nel solo laboratorio sintetico**; non autorizza l'uso browser nel desktop.

## Decisione

Il laboratorio avvia Qt WebEngine in un secondo processo, prima senza navigare. Il genitore crea una directory temporanea privata, limita le variabili d'ambiente ereditate, applica il confine di terminazione del processo e solo allora invia su `stdin` origine `.test`, endpoint proxy loopback e credenziale effimera. Nessun segreto passa negli argomenti o nell'output. `stdout` restituisce solo indicatori/conteggi sintetici. Il genitore impone 25 secondi e 64 KiB all'output complessivo; scadenza, errore o eccesso interrompono l'albero. Su macOS/Linux usa un gruppo di processi; il figlio applica anche i [limiti di risorse](M3-BROWSER-RESOURCE-LIMITS.md) prima di leggere la configurazione. Su Windows un Job Object impone arresto alla chiusura, massimo 16 processi e 1 GiB di memoria del job. Il flag Chromium per quattro renderer è solo un suggerimento, non un limite rigido. La configurazione del figlio accetta soltanto origine `http://site.test:<porta>` e proxy `http://127.0.0.1:<porta>`; gate e broker del genitore continuano a convalidare le richieste effettive.

## Evidenza e limiti

I test del supervisore usano un eseguibile sintetico che prova output eccessivo, errore, variabile d'ambiente privata e processo discendente sopravvissuto alla scadenza. La prova Qt locale macOS del 27 settembre 2026 ha restituito `PASS` dopo lo spostamento nel figlio. La CI di branch e merge va verificata per i rispettivi commit; il test di `internal/browser` è stato compilato per Windows localmente, ma il Job Object Windows non è stato eseguito qui.

Il processo separato non dimostra un firewall egress indipendente o una sandbox del file system. I limiti Unix aggiunti non includono memoria né numero aggregato di processi su macOS/Linux. Il browser non è nel desktop, non media HTTPS e non ha sessioni. La credenziale proxy resta nella memoria del figlio durante la prova. Serve ancora collaudo negativo del runtime su Windows e Linux, contenimento delle vie di rete alternative, revisione del packaging WebEngine e dei diritti Qt prima della prima voce M3. Non chiudere la voce sulla sola base di questo ADR.
