# M1 — Piattaforme target e limiti della verifica

[English](../en/M1-SUPPORT-POLICY.md) · [Validazione M1](M1-VALIDATION.md) · [Sviluppo](DEVELOPMENT.md)

Decisione dell'autore del 25 settembre 2026 per la **alpha M1**. «Target» indica le configurazioni sulle quali si intende mantenere WebFence; non equivale a collaudo su ogni versione, supporto del produttore del sistema operativo o disponibilità di una release per utenti finali.

| Sistema | Architettura e minimo target | Evidenza attuale e limite |
| --- | --- | --- |
| macOS | **26.0**, solo Apple Silicon ARM64 | Il bundle dichiara 26.0; self-test e prove locali su 26.6.2, CI su macOS 26. Non è stato avviato su 26.0 reale. |
| Windows | **10 1809 o successivo**, x86-64; anche Windows 11 x86-64 | È la prima versione Windows 10 indicata dalla [matrice Qt 6.11](https://doc.qt.io/qt-6/supported-platforms.html). CI compila, testa e avvia lo ZIP su runner Windows; nessuna prova registrata su 10 1809 o sul PC reale dell'autore. Versioni Windows 10 precedenti a 1809 non rientrano nel target. |
| Linux | **Ubuntu 24.04 LTS**, x86-64 e ARM64 | La [matrice Qt 6.11](https://doc.qt.io/qt-6/supported-platforms.html) include entrambe le architetture; CI usa Ubuntu 24.04 e costruisce/prova i `.deb` in runtime isolati. Altre distribuzioni Debian/derivate restano obiettivo di compatibilità, senza minimo o supporto dichiarato in M1. |

La scelta «Windows 10 a 64 bit» dell'autore viene precisata con il limite 1809 imposto dalla versione di Qt attuale; non costituisce un'affermazione di compatibilità verificata per ogni build successiva. [Microsoft](https://learn.microsoft.com/en-us/windows/release-health/release-information) indica che il supporto ordinario di Windows 10 è terminato il 14 ottobre 2025; gli [Extended Security Updates](https://learn.microsoft.com/en-us/lifecycle/faq/extended-security-updates) non ripristinano il normale ciclo di supporto. Per usare uno strumento di sicurezza con Windows 10 nel 2026 occorre valutare gli aggiornamenti di sicurezza del sistema ospite. [Qt](https://doc.qt.io/qt-6/supported-platforms.html) annuncia inoltre Qt 6.12 come ultima versione con supporto Windows 10: ogni upgrade Qt dovrà riesaminare questo target.

Per decisione dell'autore, **M1 si chiude con verifiche locali e CI del codice/pacchetti**, senza dichiarare superate prove assistive, multimonitor o su tutte le workstation reali. Prima di promettere una release supportata o accessibilità verificata, servono prove mirate sugli artefatti e sugli ambienti dichiarati, seguendo il [piano storico dei collaudi](M1-PREREQUISITES.md). Nessuna scansione di target esterni è richiesta per queste verifiche.
