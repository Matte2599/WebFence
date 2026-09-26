# ADR-012 — POST esplicito e cookie confinato per account di prova M3

[English](../en/ADR-012-TEST-SESSIONS.md) · [Sessioni M3](M3-SESSIONS.md) · [ADR-009](ADR-009-BROWSER-HTTP-BOUNDARY.md)

Data: 2026-09-27. Stato: **adottato per il core sintetico M3**; non abilita login pubblico o browser autenticato nel desktop.

## Decisione

Il login è un'operazione diversa dal crawler GET M1. Richiede una run gestita e `LoginConfirmed`, URL di POST esatto, origine/IP fissati, body form limitato e budget condiviso con le GET. Non segue redirect con credenziali. Un manager in memoria seleziona un solo cookie host-only senza dominio esteso, con percorso esplicito; verifica che il cookie sia utilizzabile solo sullo stesso schema/origine e sul percorso consentito. Una GET anonima precedente deve differire dalla risposta autenticata attesa. Identità e progetto restano associati a un broker specifico, e il proxy browser non legge cookie del client né restituisce `Set-Cookie`.

Questa scelta applica in modo prudente i confini di origine e percorso dei cookie descritti da [RFC 6265](https://www.rfc-editor.org/rfc/rfc6265). Il percorso di cookie da solo non costituisce una barriera di sicurezza: il broker e la policy di run ripetono i controlli prima della rete. Errori di login e verifica sono stati separati dagli esiti dei controlli.

## Limiti

Il form richiede campi utente/password semplici e un marcatore esatto di validità; non scopre automaticamente CSRF o flussi federati. Il segreto non è persistito da questo modulo, ma copie in memoria Go/OS non possono essere azzerate con garanzia. Il broker autenticato è soltanto loopback e la sessione non è ancora collegata alla GUI o al helper Qt. Cookie aggiornati dopo il login non vengono acquisiti. Occorrono ulteriori prove prima di chiudere M3.
