# M0 — Dossier per la revisione legale

[English](../en/LEGAL-REVIEW.md) · [Licenze](LICENSING.md) · [Matrice M0](M0-VALIDATION.md)

Stato: preparazione tecnica, **nessun parere legale professionale acquisito**. Questo dossier non modifica LICENSE e non attiva un CLA o un contratto commerciale. L’autore ha confermato che la revisione professionale è ancora da organizzare.

## Materiale da esaminare

- [WebFence Community License 1.0](../../LICENSE), testo inglese prevalente e traduzione italiana.
- [CONTRIBUTING](../../CONTRIBUTING.md): copyright dei contributori e assenza di diritti commerciali automatici.
- [ADR-002 Qt](ADR-002-GUI.md) e [ADR-004 SQLite/JWS](ADR-004-STORAGE-SIGNATURE.md).
- [Inventario e packaging](M0-PACKAGING.md), [pacchetto sorgenti nativi macOS](M0-SOURCE-MATERIALS.md#pacchetto-sorgenti-macos-affiancabile-al-bundle) e [sorgenti Windows](M0-WINDOWS-SOURCES.md): materiali verificati disponibili, con limiti e provenienza dichiarati.
- [Ricompilazione e sostituzione Cocoa](M0-QT-REPLACEMENT.md), provata localmente e in CI; riguarda il solo plugin, non tutte le librerie.
- [Esame delle 15 ricette macOS](../evidence/homebrew-recipe-review-2026-09-23.md), che distingue patch/risorse della build da fixture dei test e rami Linux.
- [Mappatura dei nove binari Qt macOS e attribuzioni upstream](../evidence/qt-component-review-2026-09-23.md): associazioni tecniche verificate, non inventario completo degli incorporati o approvazione legale.
- [Report](REPORTING.md): firma dell'operatore, nessuna certificazione della sicurezza, diritti su template distinti dai dati dell'utente.

Requisiti dell'autore da preservare: individui e professionisti indipendenti ammessi gratuitamente anche per incarichi retribuiti; aziende soggette ad autorizzazione, anche per uso interno; distribuzione commerciale o alle aziende soggetta ad accordo; report ai clienti ammessi; royalties eventuali negoziate, non automatiche. Il prodotto è source-available, non open source OSI.

## Questioni da risolvere nel parere

| Area | Verifica richiesta | Esito attuale |
| --- | --- | --- |
| Definizioni | Professionista individuale, società di consulenza, dipendente, impresa individuale; casi di clienti che controllano il programma | Non revisionato |
| Concessioni | Fork/hosting pubblico, uso di professionisti, ridistribuzione, servizi ospitati, eccezione dei report | Non revisionato |
| Validità contrattuale | Modalità di accettazione, legge/foro, consumatori e norme inderogabili, lingua prevalente, cessazione/ripristino | Non revisionato; nessuna giurisdizione o identità fiscale inventata |
| Qt e terzi | Diritti sulle librerie separati; notices, sorgenti corrispondenti, sostituzione/rilink ed esecuzione della versione modificata | Inventari e pacchetti di materiali disponibili; sostituzione Cocoa provata. Mappatura degli incorporati, compatibilità e completezza da revisionare |
| Contributi | Titolarità, autorizzazione del datore, concessione per licenza commerciale, brevetti e accettazione separata | CLA non attivo |
| Contratti commerciali | Ambito/versioni, installazioni, durata, supporto, prezzi/royalties, responsabilità e trattamento dati applicabile | Nessuna offerta o accordo standard attivo |

Per Qt dinamico la sola presenza dei `.dll`/framework non basta: il pacchetto deve rispettare i diritti previsti dalla licenza delle librerie, inclusa la possibilità di modificarle e utilizzare il risultato. Verificare che i termini WebFence e gli eventuali installer non li restringano. Non attribuire automaticamente la LGPL a moduli Qt disponibili solo sotto GPL; controllare i componenti effettivamente inclusi. [Indicazioni ufficiali Qt](https://www.qt.io/development/open-source-lgpl-obligations).

## Specifica dell'accordo contributori da redigere

L'accordo deve identificare parti e contributi (commit/PR), mantenere il copyright del contributore e definire espressamente una concessione non esclusiva che consenta a Matteo Luigi Feroldi di integrare, modificare, distribuire e sublicenziare quei contributi nei rilasci WebFence community e commerciali. Durata, irrevocabilità, trasferibilità, trattamento dei brevetti, dichiarazioni di titolarità, responsabilità e possibili compensi richiedono un testo verificato; non sono già concessi da questa specifica.

Prevedere varianti per persona fisica e soggetto che detiene diritti aziendali, evidenza dell'autorità a firmare, elenco di materiale di terzi e un processo di accettazione separata con versione/hash del testo. Nessuna casella nelle PR deve simulare un accordo non disponibile. Non archiviare firme, documenti d'identità o recapiti privati nel repository pubblico. La concessione sui contributi non dà automaticamente all'azienda del contributore una licenza d'uso del prodotto.

I modelli di [Contributor Agreements](https://contributoragreements.org/ca-cla-chooser/) distinguono diritti in ingresso, licenze in uscita e opzioni brevettuali: sono riferimenti per il revisore, non un accordo adottato da WebFence. Non è stato compilato né inviato alcun modulo esterno.

## Criterio di chiusura

Acquisire un parere identificabile su una precisa versione dei testi, applicare le correzioni concordate in IT/EN, approvare la versione dei termini e attivare il processo di accettazione prima di integrare contributi sostanziali destinati a rilicenza. Registrare pubblicamente solo data, versioni/hash e stato di approvazione; conservare riservati documenti e dati personali. Le aperture commerciali richiedono inoltre i relativi accordi. Questo dossier da solo non chiude M0-06.
