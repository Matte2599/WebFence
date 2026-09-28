# ADR-020 — Confine Landlock nel laboratorio CDP Linux

[English](../en/ADR-020-CDP-LANDLOCK-BOUNDARY.md) · [Laboratorio](M3-BROWSER-CDP-LAB.md) · [ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md) · [Stato M3](M3-VALIDATION.md)

Data: 2026-09-28. Stato: **adottato per la sola fixture sperimentale Linux**.

## Contesto e decisione

Il container del laboratorio CDP è in sola lettura, ma questo non impedisce a Chromium di leggere file estranei al profilo temporaneo. Installiamo una policy Landlock nel thread che avvia Chromium, dopo aver preparato le connessioni Unix e le pipe CDP. Il thread resta bloccato al goroutine fino alla fine di Chromium; il processo figlio eredita la restrizione. L'helper applica `no_new_privs` e richiede Landlock ABI 3 o successiva, necessaria per gestire anche `REFER` e `TRUNCATE`; se manca il supporto, la prova fallisce.

La policy consente lettura/esecuzione in `/usr`, `/lib`, `/lib64`, `/etc`, `/proc` e `/dev` se presenti, e lettura/scrittura/esecuzione solo nella directory privata dell'helper. Il file sintetico di controllo si trova in una directory temporanea separata. L'helper ne verifica la lettura prima di installare la policy, poi richiede `EACCES` sia in lettura sia in scrittura; verifica che un figlio possa leggere un file privato e non possa leggere quello vietato. Il browser viene avviato soltanto dopo queste verifiche. La prova rimane nel container con filesystem di base in sola lettura, quote di memoria/PID e filtro seccomp della rete.

Una verifica successiva osserva Chromium tramite CDP: terminata la fixture HTTP(S), disabilita l'intercettazione delle richieste solo per due navigazioni `file://` a file sintetici. Il file nel profilo privato viene letto e mostrato; il file esterno restituisce `net::ERR_ACCESS_DENIED`. Senza la chiamata Landlock, una build diagnostica fallisce proprio quest'ultima asserzione. La prova resta separata dal traffico verso target e non usa URL esterni.

## Limiti

Landlock si applica qui al thread che lancia Chromium e ai suoi discendenti, non a tutti i thread già esistenti dell'helper. I descrittori aperti prima della restrizione, inclusi i socket Unix verso il broker e le pipe CDP, rimangono utilizzabili. `/proc` e `/dev` devono restare leggibili per questa fixture Chromium; la policy non protegge da letture di ogni dato visibile in quei filesystem o da altri canali non coperti. Anche la navigazione a due file sintetici non prova che un renderer ostile non possa evadere il browser. Chromium usa ancora `--no-sandbox`; il container è parte indispensabile della prova. Questa decisione non abilita il browser nel desktop, non stabilisce una policy per macOS/Windows e non chiude il primo criterio M3.

Riferimento tecnico: [documentazione Landlock del kernel Linux](https://docs.kernel.org/userspace-api/landlock.html).
