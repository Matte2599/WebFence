# ADR-008 — Trasporto con IP pubblici fissati e policy per route

[English](../en/ADR-008-PINNED-PUBLIC-TRANSPORT.md) · [Visite controllate](M1-CONTROLLED-CRAWL.md) · [ADR-003](ADR-003-TRANSPORT.md)

Data: 2026-09-25. Stato: **adottato per il core M1 sperimentale**; distribuzione e scansione di produzione non approvate.

## Contesto e decisione

Il broker M0 prova i confini HTTP su loopback, ma non permette di verificare l'integrazione con origini pubbliche autorizzate. Uno scope basato sul solo hostname non impedisce DNS rebinding o redirect verso reti interne. Estendiamo il broker con `NewAuthorizedPublic`: ogni origine esatta deve comparire nello snapshot dell'autorizzazione e avere IP pubblici esplicitamente fissati; ogni risposta DNS deve contenere solo quegli IP. Un lifecycle vincolato è obbligatorio; `RunCrawl` lo ottiene dallo store. Le singole run sono sequenziali, con budget e cadenza per origine. La policy indipendente dei metodi e dei prefissi di percorso viene ricontrollata su ogni hop.

Un grant non dimostra la titolarità del target. Solo l'operatore può attestare l'autorizzazione e scegliere percorsi senza effetti indesiderati. Non supportiamo per ora reti private/VPN, override di IP speciali, proxy, cookie, autenticazione o browser. Questa scelta è volutamente restrittiva: accessi a target interni richiederanno un modello di autorizzazione e di egress separato.

## Confini applicati

Il costruttore pubblico limita grant, IP per grant, numero di tentativi, durata e intervallo minimo. Gli IP mappati IPv4-in-IPv6, con zone, non global-unicast, privati, loopback, link-local e intervalli speciali sono respinti. La tabella speciale è una fotografia conservativa dei registri [IANA IPv4](https://www.iana.org/assignments/iana-ipv4-special-registry/) e [IANA IPv6](https://www.iana.org/assignments/iana-ipv6-special-registry/) al 25 settembre 2026; va aggiornata con prove quando cambiano i registri. Non è una verifica che un IP sia raggiungibile o appartenga al target.

Ogni hop ricontrolla autorizzazione, origine, route, budget, DNS e peer; la connessione è diretta all'IP selezionato e il nome dell'origine resta per Host e verifica TLS. Un solo tentativo/catena alla volta impedisce che la latenza DNS o TLS riordini le partenze rispetto al rate limit. Errori di rete esposti al report sono codici redatti. La policy rifiuta percorsi ambigui invece di indovinare come il server li interpreterà. I risultati e la superficie HTML restano effimeri.

## Conseguenze e riesame

Il limite di una sola catena riduce la velocità ma rende controllabile il ritmo iniziale. Il rate è locale al broker: due processi possono sommarsi. Gli indirizzi pubblici fissati richiedono aggiornamento esplicito se DNS cambia; per un dominio CDN questo può ridurre molto la copertura. Nessun test su Internet reale, su Windows/Debian reali o con firewall di sistema è incluso in questo blocco. Prima di dichiarare pronto un trasporto di produzione servono coordinamento tra run/processi, controllo del carico del target, UI di autorizzazione e grant, persistenza sicura, egress indipendente dal broker e collaudi su piattaforme reali. La [roadmap M1](../ROADMAP.md) li mantiene aperti.
