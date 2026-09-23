# Licenza, contributi e governance

[English](../en/LICENSING.md) · [Indice](../README.md)

Decisione del 2026-09-23, basata sui requisiti e sui chiarimenti di Matteo Luigi Feroldi. Il testo applicabile è [LICENSE](../../LICENSE); questa guida lo riassume. Il testo inglese della licenza prevale sulla traduzione nei limiti della legge applicabile.

## Scelta

**WebFence Community License 1.0**, identificatore di progetto `LicenseRef-WebFence-Community-1.0`, più eventuali accordi commerciali separati. Non usare badge MIT, GPL, «OSI approved» o un identificatore SPDX standard inventato. Non è un canone automaticamente dovuto: royalties e corrispettivi esistono soltanto se concordati.

| Scenario | Regola |
| --- | --- |
| Persona che studia, modifica o usa WebFence per un progetto personale non commerciale | Gratuito |
| Fork pubblico personale e condivisione gratuita con persone ammesse | Consentiti con avvisi e licenza |
| Professionista indipendente che esegue personalmente un'analisi retribuita | Gratuito, anche se il cliente è un'azienda |
| Consegna del report a un cliente aziendale | Consentita; il cliente non riceve il programma |
| Dipendente o collaboratore che opera lo scanner come strumento interno dell'azienda | Autorizzazione scritta dell'autore |
| Società di consulenza o studio costituito in forma societaria che usa il programma | Autorizzazione scritta dell'autore |
| Distribuzione del programma a un'azienda, anche gratuita | Autorizzazione scritta dell'autore |
| Vendita del programma, OEM, noleggio, accesso SaaS/API | Autorizzazione scritta dell'autore |

Il professionista ammesso è una persona fisica autonoma, anche con attività professionale individuale; una società non diventa esente perché svolge consulenza. L'uso personale di un dipendente fuori dall'attività aziendale resta ammesso. L'autorizzazione sul software non autorizza a scansionare target di terzi.

## Perché non una licenza standard

| Alternativa | Incompatibilità con il requisito |
| --- | --- |
| MIT / Apache-2.0 | Consentono sfruttamento commerciale e aziendale senza permesso specifico dell'autore. |
| GPL / AGPL | Il copyleft non è un divieto di uso commerciale; una seconda licenza commerciale non rende obbligatorio comprarla per tutti gli usi aziendali. |
| PolyForm Noncommercial | Non rappresenta l'eccezione richiesta per incarichi professionali retribuiti; inoltre prevede concessioni proprie per organizzazioni non commerciali. |
| Business Source License 1.1 | Include un passaggio futuro a una licenza aperta; non conserva indefinitamente il controllo richiesto per ogni versione. |
| Creative Commons NC | Non scelta per licenziare il software; non risolve da sola la distinzione tra professionisti e aziende. |

La classificazione source-available deriva dai vincoli d'uso: la [definizione OSI](https://opensource.org/osd) non consente di riservare il settore aziendale. Riferimenti per il confronto: [GPL FAQ](https://www.gnu.org/licenses/gpl-faq.html.en), [PolyForm Noncommercial](https://polyformproject.org/licenses/noncommercial/1.0.0), [BSL 1.1](https://mariadb.com/bsl11/).

## Contributi e diritto di rilicenziare

I contributori conservano il copyright. Una PR, un DCO o un commit firmato non attribuiscono automaticamente all'autore tutti i diritti necessari per licenze commerciali alternative. Prima di includere contributi esterni nei rilasci commerciali serve un accordo esplicito che conceda tali diritti, oppure mantenere quei contributi fuori dalla distribuzione interessata.

Il CLA non è ancora predisposto né attivo. La procedura in [CONTRIBUTING.md](../../CONTRIBUTING.md) richiede di risolvere questo aspetto prima di integrare contributi sostanziali esterni destinati anche alla distribuzione commerciale. Non presumere cessione dei diritti dal silenzio.

## Dipendenze, modelli e dati

La licenza di WebFence non cambia le licenze di librerie, motori, regole importate, pesi AI, dataset e CVE. Prima di distribuirli: inventario, versione, provenienza, licenza, avvisi e compatibilità con entrambe le modalità di distribuzione. Un processo separato non elimina automaticamente gli obblighi. Non copiare template di scanner terzi senza verifica.

Il testo personalizzato è una prima stesura tecnica, non una licenza standard già validata da un legale. Una revisione professionale deve verificare definizioni, applicabilità, tutela dei consumatori, accordi per contributori e contratti commerciali. Non sono inventati tariffe, partita IVA, indirizzi, email, foro competente o condizioni di pagamento.

Responsabile delle decisioni: Matteo Luigi Feroldi. Le modifiche pubbliche di licenza devono avere versione e changelog; non revocano retroattivamente i diritti regolarmente ottenuti sulle copie precedenti.

Preparazione M0: [dossier per il revisore e specifica dell’accordo contributori](LEGAL-REVIEW.md). Non è un parere acquisito né un CLA attivo.
