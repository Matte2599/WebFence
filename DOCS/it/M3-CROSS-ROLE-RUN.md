# M3 — Esecuzione gestita del controllo tra ruoli

[English](../en/M3-CROSS-ROLE-RUN.md) · [Regola](M3-CROSS-ROLE.md) · [Sessioni](M3-SESSIONS.md) · [Roadmap](../ROADMAP.md)

`checks.RunCrossRole` lega il primo controllo `AUTH-CROSSROLE-001` a **una run gestita solo loopback**. Il piano deve dichiarare progetto, origine/grant IP loopback, policy GET, limiti, URL esatti di login/verifica, due account di prova e risorsa privata. L'operatore conferma separatamente il POST di login, la risorsa e il divieto di accesso per l'altro ruolo. Le due identità devono avere ID, username, riferimenti ai segreti e marcatori di validità distinti. I segreti vengono letti solo da `session.SecretSource`; l'orchestratore non li conserva.

Prima della rete, valida il piano, l'autorizzazione corrente, la risorsa sotto scope e la policy. Un solo broker applica a entrambi i login e al controllo IP fissati, budget, cadenza, dimensione body e revoca. Le sessioni, il broker e la run vengono chiusi su ogni uscita. Un login fallito o ambiguo produce `inconclusive`, mai un esito negativo; un'interruzione conserva l'esito inconcludente e la causa. Il risultato contiene soltanto ID/revisione regola, esito e codice redatto. Non è scritto nel database M1 o nel report M2.

**Fixture sintetiche locali del 27 settembre 2026:** accesso incrociato riprodotto e accesso negato (11 richieste gestite ciascuno), secondo segreto mancante, piano fuori scope/non confermato e budget esaurito. Test del pacchetto con race detector passati; nessun target esterno. La CI del branch e del merge va verificata per commit. Il successivo [flusso desktop](M3-AUTH-DESKTOP.md) espone una prova con due account solo loopback; non sono coperti grant pubblici, login CSRF/MFA/OIDC né misure reali di accuratezza.
