# Report, evidenze e firma

[English](../en/REPORTING.md) · [Indice](../README.md)

Stato: contratto proposto M2, da validare con librerie e vettori di interoperabilità.

## Contenuto

Un report include progetto e ambiente, target autorizzati, scope escluso, intervallo UTC, versione del motore/regole, profilo e budget, snapshot CVE, eventuale modello AI, stato della scansione e copertura. I controlli saltati, limitati, falliti e bloccati devono essere visibili accanto ai risultati.

Ogni finding contiene ID stabile, titolo IT/EN, categoria, CWE e CVE se pertinenti, severità distinta dalla confidenza, stato di verifica, asset/route, contesto d'identità redatto, prerequisiti, prove, passi di riproduzione minimali, impatto e rimedio. CVSS richiede versione, vettore e fonte; un punteggio sconosciuto rimane tale. Priorità operativa e severità tecnica sono campi diversi.

Le evidenze sono minimizzate prima dell'esportazione: segreti, cookie e contenuti personali non necessari vengono esclusi. Conservare hash e descrizione delle omissioni; l'hash identifica ciò che è stato incluso, non prova il contenuto rimosso. Le pagine acquisite non vengono renderizzate come HTML attivo nel desktop o negli export.

## Formati e ordine

JSON strutturato e HTML statico in M2; PDF in M6 dopo QA del rendering. La lingua selezionata cambia il testo, non ID o esiti. Un bundle bilingue elenca separatamente gli artefatti IT ed EN. Il PDF non è automaticamente un documento PAdES o una firma qualificata: eventuali requisiti legali di firma richiedono un progetto distinto.

1. Bloccare uno snapshot dei risultati; generare gli artefatti finali.
2. Creare un manifest con versione schema, ID report/run/progetto, UTC dichiarato dal firmatario, lingua, identità dichiarata, ID chiave e lista dei file con percorso relativo, dimensione e SHA-256.
3. Canonicalizzare il manifest con [JCS, RFC 8785](https://www.rfc-editor.org/rfc/rfc8785). Rifiutare chiavi duplicate, numeri non validi e stringhe non valide; usare stringhe per identificatori numerici lunghi.
4. Firmare i byte canonici come payload incorporato in un [JWS, RFC 7515](https://www.rfc-editor.org/rfc/rfc7515.html). Profilo iniziale proposto: algoritmo `Ed25519` definito da [RFC 9864](https://www.rfc-editor.org/rfc/rfc9864.html), header protetto con algoritmo, `kid` e tipo di documento. Usare una libreria mantenuta che supporti questo profilo; non inventare una costruzione crittografica o sostituire algoritmi silenziosamente.
5. Distribuire artefatti e JWS; un'eventuale copia leggibile del manifest deve corrispondere esattamente al payload verificato. La firma non è elencata nel manifest per evitare autoriferimenti.

## Verifica e fiducia

Il verificatore offline deve accettare soltanto il profilo previsto e una chiave pubblica scelta come fidata fuori dal bundle. `kid` è un selettore, non prova di identità. Rifiutare `none`, algoritmi imprevisti, chiavi recuperate da URL del report, header critici sconosciuti, percorsi assoluti, traversal, duplicati e file mancanti o alterati. Un file aggiunto non elencato non è firmato e va segnalato come tale.

Verificare JWS, schema, corrispondenza del manifest e hash dei file prima di mostrare «firma valida e firmatario fidato». Distinguere firma non valida, chiave sconosciuta, chiave revocata e stato di revoca non aggiornato. La chiave pubblica inclusa nel bundle può aiutare il trasporto, ma non fonda la fiducia.

## Gestione delle chiavi e limiti

Chiavi generate per installazione/operatore, private nel portachiavi o contenitore cifrato, mai una chiave universale incorporata. La firma è dell'operatore configurato, non automaticamente di Matteo Luigi Feroldi. Prevedere esportazione sicura, rotazione, revoca e conservazione delle vecchie chiavi pubbliche. Se la chiave è indisponibile, produrre soltanto un export esplicitamente non firmato.

Una firma valida rileva alterazioni e identifica una chiave fidata; non dimostra che ogni osservazione sia corretta, che il sito sia sicuro o che l'orario locale sia attendibile. Timestamp fidato e revoca aggiornata non sono garantiti offline. Correzioni, traduzioni successive e retest producono nuovi report firmati che referenziano il predecessore, senza modificarlo.
