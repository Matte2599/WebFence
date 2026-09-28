# M3 — Esecuzione gestita del controllo tra ruoli

[English](../en/M3-CROSS-ROLE-RUN.md) · [Regola](M3-CROSS-ROLE.md) · [Sessioni](M3-SESSIONS.md) · [Roadmap](../ROADMAP.md)

`checks.RunCrossRole` lega il primo controllo `AUTH-CROSSROLE-001` a **una run gestita**. Il piano deve dichiarare progetto, origine/grant IP, policy GET, limiti, URL esatti di login/verifica, due account di prova e risorsa privata. L'operatore conferma separatamente il POST di login, la risorsa e il divieto di accesso per l'altro ruolo. Le due identità devono avere ID, username, riferimenti ai segreti e marcatori di validità distinti. I segreti vengono letti solo da `session.SecretSource`; l'orchestratore non li conserva.

La modalità predefinita e il dialogo Qt usano `CrossRoleRunLoopback`. `CrossRoleRunPinnedPublic` deve essere selezionata esplicitamente e richiede `SessionRoutes.PublicConfirmed` oltre alla conferma del POST: seleziona il [broker HTTPS con IP pubblici fissati](ADR-018-AUTHENTICATED-PUBLIC-TRANSPORT.md), un solo grant e i suoi limiti più stretti. Il piano rifiuta modalità sconosciute, IP loopback in modalità pubblica e IP non loopback nella modalità locale. Il [dialogo Qt](M3-AUTH-DESKTOP.md) offre ora questa modalità con conferma pubblica distinta, mantenendo i limiti più stretti.

Prima della rete, valida il piano, l'autorizzazione corrente, la risorsa sotto scope e la policy. Un solo broker applica a entrambi i login e al controllo IP fissati, budget, cadenza, dimensione body e revoca. Le sessioni, il broker e la run vengono chiusi su ogni uscita. Un login fallito o ambiguo produce `inconclusive`, mai un esito negativo; un'interruzione conserva l'esito inconcludente e la causa. Il risultato contiene soltanto ID/revisione regola, esito e codice redatto. Non è scritto nel database M1 o nel report M2.

Una successiva fixture con rotazione confermata per entrambe le identità produce il finding atteso con 15 richieste sotto lo stesso budget; errori di rotazione rendono invece i controlli dipendenti inconcludenti.

**Fixture sintetiche locali del 27 settembre 2026:** accesso incrociato riprodotto e accesso negato (11 richieste gestite ciascuno), secondo segreto mancante, piano fuori scope/non confermato e budget esaurito. Test del pacchetto con race detector passati; nessun target esterno. Il successivo [flusso desktop](M3-AUTH-DESKTOP.md) espone una prova con due account solo loopback.

**Estensione core del 28 settembre 2026:** una run con modalità pubblica esplicita costruisce il broker HTTPS e fallisce in modo inconcludente su DNS sintetico negato, senza aprire socket. Conferma assente, modalità sconosciuta o IP errati sono respinti prima della rete. Le prove TLS del broker e quelle complete dei due account loopback sono separate: manca ancora una prova end-to-end di due account su staging pubblico autorizzato. Altri flussi CSRF, MFA/OIDC e misure reali di accuratezza restano aperti.
