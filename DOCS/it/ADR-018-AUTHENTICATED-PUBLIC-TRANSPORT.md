# ADR-018 — Sessioni di prova su grant pubblici HTTPS

[English](../en/ADR-018-AUTHENTICATED-PUBLIC-TRANSPORT.md) · [Sessioni M3](M3-SESSIONS.md) · [ADR-008](ADR-008-PINNED-PUBLIC-TRANSPORT.md)

Data: 2026-09-27. Stato: **adottato per il solo core sperimentale**.

## Decisione

`NewAuthorizedPublicWithSession` è un costruttore distinto dal trasporto pubblico anonimo M1. Richiede una run gestita, un solo grant di origine esatta con IP pubblici fissati, policy GET per la verifica, URL esatti di login e verifica, conferma del POST e una **seconda conferma esplicita** dell'invio di credenziali all'origine pubblica. L'origine deve essere HTTPS; il broker verifica il certificato e il nome host con TLS 1.2 o superiore. Il budget è al massimo 128 tentativi in 15 minuti, sequenziali e separati da almeno 500 ms. Nessun redirect del login o delle GET autenticate viene seguito. DNS, peer, autorizzazione, route, revoca e budget sono ricontrollati per ogni richiesta.

La scelta evita di ampliare implicitamente `NewAuthorizedPublic`, usato per visite anonime, e non abilita il dialogo desktop, che resta limitato a loopback. L'operatore deve conservare prova dell'autorizzazione e dichiarare URL e IP: il software non può dimostrare la titolarità del target. Il manager di sessione esistente può usare il broker, mantenendo cookie e segreti effimeri e verifica positiva per identità; una run gestita pubblica a due account non è ancora orchestrata nel prodotto.

## Prove e limiti

Una fixture TLS locale simula un'origine pubblica con DNS e peer fissati; verifica POST, cookie su GET, redirect non seguito, URL fuori scope e peer errato. Non effettua richieste a Internet. Restano aperti collaudi su un target di staging autorizzato, flussi login complessi, browser autenticato, controllo del carico aggregato tra processi ed egress indipendente.
