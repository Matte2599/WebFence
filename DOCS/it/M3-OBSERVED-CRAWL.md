# M3 — Visite esplicite delle route `fetch` osservate

[English](../en/M3-OBSERVED-CRAWL.md) · [Osservazioni](M3-DYNAMIC-OBSERVATIONS.md) · [Run gestite](M1-MANAGED-RUNS.md) · [Roadmap](../ROADMAP.md)

`scanner.RunObservedCrawl` riceve fino a 256 osservazioni effimere del proxy e **ripete solo fino a 32 indici scelti esplicitamente dall'operatore**, dopo una conferma separata. Ammette soltanto `fetch` GET completate con stato `2xx`, percorso iniziale/finale identico e nessuna query. La ripetizione usa il percorso senza query osservato: può quindi avere semantica diversa dalla richiesta JavaScript originale. Non ripete POST, cookie, credenziali, body, link o redirect.

Le osservazioni e i percorsi sono input non fidati, non autorizzazioni. L'origine è dichiarata separatamente; il servizio rifiuta selezioni duplicate, percorsi ambigui e piani con seed o link impliciti. Una nuova `RunCrawl` controlla nuovamente autorizzazione corrente, scope, policy GET, grant IP e budget prima della rete. Il risultato e il ledger contengono soltanto codici e conteggi redatti. Questo servizio core non è collegato alla GUI né avvia autonomamente il browser o la ripetizione.

**Fixture sintetiche locali del 27 settembre 2026:** due route JSON scelte e visitate sotto run, route non selezionata mai visitata, corpo privato assente dal risultato; conferma mancante, metodo/tipo/stato non ammessi, percorso alterato, query e grant invalido respinti senza richieste; redirect verso route non selezionata fermato dopo un tentativo. Suite Go completa, race detector mirato, `go vet` e collegamenti documentali locali passati; CI del branch/merge da verificare per commit. Non è una misura di copertura o accuratezza reale.

Restano aperti navigazione e interazione DOM, browser isolato nel prodotto, gestione sicura di query/parametri dinamici e controlli API specifici. La terza voce M3 non è chiusa.
