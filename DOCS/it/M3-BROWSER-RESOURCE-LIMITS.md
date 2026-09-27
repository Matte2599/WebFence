# M3 — Limiti di risorse dell'helper browser

[English](../en/M3-BROWSER-RESOURCE-LIMITS.md) · [ADR-011](ADR-011-BROWSER-HELPER.md) · [Stato M3](M3-VALIDATION.md) · [Roadmap](../ROADMAP.md)

Il laboratorio Qt chiama `browser.ApplyHelperResourceLimits` nel processo figlio, prima di leggere la configurazione, creare Qt o avviare discendenti. Un errore nell'applicazione interrompe il figlio; il genitore restituisce `browser_helper_failed` senza propagare output o configurazione. I limiti ereditati più restrittivi non vengono aumentati. Il genitore conserva la scadenza, il limite dell'output e la terminazione dell'albero di processi.

Su macOS e Linux il figlio riduce il limite rigido e quello corrente di descrittori aperti a **1024**, la dimensione massima dei file creati a **64 MiB** e i core dump a **zero**. Su Linux riduce anche lo spazio di indirizzamento virtuale a **16 GiB per processo**. I discendenti ereditano questi limiti. La prova sintetica controlla i valori nel figlio e in un discendente e verifica che il genitore conservi i propri limiti; la fixture Qt locale macOS continua a passare.

**Limiti della prova:** macOS 26.6.2 ha rifiutato `RLIMIT_AS` con `EINVAL` nella prova locale; quindi non dichiariamo un limite di memoria macOS. Il limite Linux riguarda ogni processo, non la memoria aggregata dell'albero. Il numero totale di processi non è limitato da questo blocco: `RLIMIT_NPROC` conta i processi dell'intero utente, non quelli del solo helper. L'helper è codice fidato che applica i limiti prima di elaborare pagine; non è una sandbox di rete o file system. Il Job Object Windows resta il meccanismo distinto già previsto dall'ADR-011. Non abilitare il browser nel desktop in base a questo blocco.
