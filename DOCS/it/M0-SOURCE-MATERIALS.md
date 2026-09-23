# M0 — Raccolta dei materiali sorgente

[English](../en/M0-SOURCE-MATERIALS.md) · [Packaging](M0-PACKAGING.md) · [Indice](../README.md)

## Funzione disponibile

`scripts/collect-native-sources.py` raccoglie archivi upstream e avvisi in una directory separata dal bundle macOS. L’input è il `native-build.json` prodotto dal nostro packaging, **un input di build fidato**, mai testo proveniente da siti analizzati o AI. Non esegue script contenuti negli archivi e non modifica l’app firmata, Qt installato o le preferenze personali.

Richiede Python 3.9+ e curl 8.4+ per i download: [da questa versione il limite di dimensione opera anche senza Content-Length](https://curl.se/docs/manpage.html#--max-filesize). Gli URL e gli hash SHA-256 provengono dalle SBOM Homebrew conservate nell’inventario. Il confronto garantisce corrispondenza con quei metadati, non una verifica indipendente dell’autenticità del loro editore.

```sh
python3 scripts/collect-native-sources.py \
  dist/WebFence.app/Contents/Resources/notices/native-build.json \
  /tmp/webfence-source-materials

# Facoltativo: riusare un archivio locale, nuovamente verificato per hash.
# Aggiungere --reuse-archive PERCORSO al comando; l’opzione è ripetibile.
python3 -m unittest discover -s scripts/tests -p 'test_*.py' -v
```

La directory di destinazione deve essere nuova. L’output contiene gli archivi originali, avvisi copiati con percorso e hash, eventuali link registrati ma mai seguiti, copia dell’inventario iniziale e `source-materials.json`. Fino al successo rimane `INCOMPLETE`; un errore non produce il manifest finale. I materiali già scaricati non vengono cancellati: possono essere ispezionati e riutilizzati, verificandoli di nuovo, in una nuova destinazione. Non distribuire un risultato incompleto.

Solo HTTPS, anche dopo redirect; massimo cinque redirect, 180 s per trasferimento e interruzione per stallo. Il file `.curlrc` personale non viene caricato. Limiti: 32 archivi, 128 MiB ciascuno, 512 MiB complessivi; per archivio 100.000 membri e 2 GiB dichiarati dopo decompressione, 4 MiB per avviso e 32 MiB di avvisi. La lettura tar è progressiva: nessuna estrazione generale, nessun symlink creato o codice eseguito. Percorsi assoluti/traversal, avvisi duplicati, hash errati e budget superati bloccano il risultato.

Sono copiati nomi comuni di licenza/avviso, directory `LICENSES`, metadati `qt_attribution.json` e i file richiamati esplicitamente dal `LICENSE.TXT` di FreeType. Se una futura versione FreeType perde un riferimento atteso, la raccolta richiede revisione. Alcuni avvisi si trovano dentro file sorgente, conservati integralmente. Questa selezione è un aiuto alla revisione, non dimostra che ogni obbligo di attribuzione sia coperto.

## Verifica del 2026-09-23

Inventario del bundle pulito `8d3053f`: **15 archivi, 170.920.531 byte (circa 163 MiB), 247 file di avvisi/metadati**. Download effettivi verificati; Qt e GLib riusati da archivi locali con hash verificato. Seconda raccolta interamente da archivi verificati, per includere i riferimenti FreeType inizialmente non intercettati dai nomi generici. GLib ora fornisce i testi `LICENSES`, assenti nel keg installato; `COPYING` resta un link nell’archivio, registrato senza seguirlo.

[Manifest e hash della raccolta](../evidence/macos-sources-2026-09-23/source-materials.json) e [inventario di input](../evidence/macos-sources-2026-09-23/native-build.input.json) sono conservati nel repository; gli archivi e i testi raccolti restano in `/tmp/webfence-native-sources-8d3053f-reviewed/`, fuori da Git. Sette test sintetici passati: conservazione dei file esistenti, hash errato, percorsi e duplicati, limiti, riferimenti mancanti, metadati non validi e requisiti curl. [CI `39c48ef` superata, sei job](https://github.com/Matte2599/WebFence/actions/runs/35888497914): sette test del raccoglitore sui quattro target, oltre a regressioni del prodotto e packaging.

Verificati indipendentemente tutti gli hash di archivi e avvisi. Trasferimento GLib fresco con la configurazione curl finale riuscito; un limite di 1 KiB rifiuta il download senza salvare byte. Il curl locale segnala questo rifiuto con codice 56 e messaggio di dimensione massima superata: il raccoglitore tratta qualsiasi uscita non zero come fallimento, senza dipendere da un codice specifico.

## Confini ancora aperti

Il manifest mantiene **`distribution_ready: false` e `corresponding_sources_complete: false`**. Sono sorgenti upstream delle librerie identificate, non ancora tutti i sorgenti corrispondenti della distribuzione: servono patch/risorse dei package manager, ambiente e istruzioni di ricompilazione/sostituzione. Il supplemento GLib descritto sotto copre una parte di questi materiali; il resto della chiusura deve ancora essere verificato e assemblato.

Gli avvisi dell’intero albero sorgente possono includere componenti non distribuiti; occorre mapparli ai binari reali e scegliere/verificare i termini applicabili. Le ricette, le SBOM e la patch Cocoa restano nel bundle originale identificato dall’inventario. Gli avvisi possono ora essere inclusi esplicitamente prima della firma, con la procedura seguente; gli archivi restano separati e nessuna release è pubblicata. Revisione legale, minimi OS, prove desktop pulite e materiali Windows restano parte di M0-02/M0-06; raccogliere file non li approva.

## Supplemento GLib identificato

La [ricetta Homebrew alla revisione 550d1a4](https://github.com/Homebrew/homebrew-core/blob/550d1a4e5c2da2dd3c8c630e43b1ef9a84afd700/Formula/g/glib.rb) corrisponde byte per byte a quella installata dopo la sola rimozione del blocco `bottle`. Acquisiti la patch `Patches/glib/hardcoded-paths.diff` dello stesso albero Git e l’archivio gobject-introspection 1.86.0, con SHA-256 verificato rispetto alla ricetta. Il dry-run della patch sul sorgente GLib 2.90.0 verificato passa senza fuzz; nessuna installazione, esecuzione di codice o modifica del keg.

[Provenienza e hash](../evidence/macos-sources-2026-09-23/glib-supplement.json) conservati; materiali in `/tmp/webfence-glib-materials-550d1a4/`, separati dall’output automatico. Il confronto identifica i materiali della ricetta, senza dimostrare una ricostruzione binaria identica della bottle. L’integrazione nel pacchetto completo e la verifica delle altre ricette restano aperte.

Anche la patch configure Big Sur di libb2 0.98.1 è stata acquisita dall’URL fissato nella ricetta, verificata per SHA-256 e provata con dry-run senza fuzz: [ledger](../evidence/macos-sources-2026-09-23/libb2-supplement.json), file locale `/tmp/webfence-libb2-materials/configure-big_sur.diff`. La patch D-Bus è già incorporata nella ricetta conservata. Altre ricette contengono sostituzioni testuali e risorse di test (font/immagini): non equipararle automaticamente a contenuti distribuiti; conservare le ricette e verificare il ruolo di ciascun materiale durante l’assemblaggio.

## Includere gli avvisi verificati nel bundle di sviluppo

Dopo la raccolta dei sorgenti, ricompilare con la directory di input facoltativa:

```sh
WEBFENCE_NATIVE_SOURCE_MATERIALS=/tmp/webfence-source-materials \
  sh scripts/package-macos.sh
```

`scripts/attach-native-sources.py` confronta identità dei pacchetti, URL sorgente e SHA-256 con il nuovo inventario del bundle. Ricontrolla ogni archivio sorgente e rigenera gli avvisi direttamente dall’archivio, confrontandoli con il manifest della raccolta. I file di avvisi sciolti eventualmente modificati vengono ignorati. Raccolte incomplete, versioni diverse delle dipendenze, archivi corrotti, metadati degli avvisi discordanti, symlink nei materiali di input e superamento dei budget della raccolta bloccano l’inclusione. Le directory di avvisi esistenti sono preservate; l’output temporaneo viene rimosso in caso di errore. Gli input sono artefatti di build fidati in una directory controllata dal costruttore; modifiche concorrenti non sono supportate.

L’output è `Contents/Resources/notices/upstream-source`, incluso prima della firma ad hoc finale. `attachment.json` collega l’hash dell’inventario corrente separatamente dai metadati di acquisizione: il commit WebFence della raccolta originaria non viene presentato come commit del nuovo eseguibile. Per le dipendenze locali correnti si possono includere tutti i 247 avvisi dell’albero sorgente. Gli archivi completi restano nella directory separata di raccolta; i percorsi archivio nel manifest copiato si riferiscono a quella directory, non all’app. Conservare entrambi i materiali nella preparazione di una futura distribuzione. Patch/risorse Homebrew supplementari richiedono ancora assemblaggio.

Questa opzione non effettua download. Senza la variabile, il packaging di sviluppo mantiene avvisi installati e inventario esistenti. Entrambi i manifest conservano `distribution_ready=false`; gli avvisi sorgente copiati possono includere componenti non distribuiti e richiedono mappatura/revisione. Non usare il solo helper per modificare un bundle già firmato: usare l’entry point di packaging, che firma le risorse e impedisce a uno staging fallito di sostituire l’app precedente.

Verifica locale: 12 regressioni Python superate (sette raccolta, cinque inclusione), estrazione effettiva da 15 archivi e ricontrollo indipendente dei 247 hash. Bundle da `0fd78f0` con modifiche dichiarate: firma ad hoc, collegamento all’inventario corrente e self-test Cocoa superati. La CI precedente `0fd78f0`, run 35897426940, ha completato tutti e sei i job; [CI `56840c5`](https://github.com/Matte2599/WebFence/actions/runs/35898848286) delle nuove modifiche completata: tutti e sei i job superati, comprese le 12 regressioni Python sui quattro target. L’inclusione effettiva dei 247 avvisi è verificata localmente; in CI il packaging resta senza la variabile facoltativa.

Iniezione di errore nel packaging completo: raccolta sintetica con `INCOMPLETE` rifiutata; hash dell’eseguibile e dell’allegato precedenti invariati, firma ad hoc precedente ancora valida.

## Prima acquisizione Windows

Acquisito l’archivio MSYS2 Qt 6.11.2-2 con sorgente upstream, ricetta e dieci patch; hash interni verificati senza eseguire codice. [Provenienza, hash e limiti](../evidence/windows-qt-source-2026-09-23.md). La successiva [raccolta Windows](M0-WINDOWS-SOURCES.md) copre 21 pacchetti sorgente con ricette collegate ai binari; assemblaggio completo e revisione restano aperti.

Il manifest Windows ora collega le DLL ai pacchetti binari in cache e all’hash della ricetta sorgente: [procedura e limiti](ADR-007-PACKAGING.md#corrispondenza-con-i-pacchetti-binari-msys2). Verificato localmente anche il collegamento PKGBUILD sorgente ↔ BUILDINFO del pacchetto Qt con checksum pubblicato; firma distaccata ancora non verificata.

## Supplementi Homebrew nel bundle macOS

Il piano revisionato `packaging/macos/homebrew-supplements.json` identifica quattro file: ricetta GLib alla revisione Homebrew già verificata, patch dei percorsi GLib, gobject-introspection 1.86.0 e patch configure di libb2. Ciascun gruppo è vincolato all’hash della ricetta installata nel bundle, oltre che a nome/versione nell’inventario. Il piano è un input di build fidato da revisionare quando cambiano le dipendenze: il programma non interpreta Ruby e non deduce automaticamente tutte le risorse necessarie.

```sh
# Inclusione esplicita prima della firma, con download HTTPS verificati.
WEBFENCE_NATIVE_SUPPLEMENT_PLAN=packaging/macos/homebrew-supplements.json \
  sh scripts/package-macos.sh

# Raccolta separata, senza modificare un’app firmata.
python3 scripts/native_supplements.py \
  dist/WebFence.app/Contents/Resources/notices/native-build.json \
  packaging/macos/homebrew-supplements.json \
  /tmp/webfence-homebrew-supplements
```

La variabile è indipendente da `WEBFENCE_NATIVE_SOURCE_MATERIALS`: si possono includere insieme i 247 avvisi upstream e i supplementi. Senza un piano esplicito il packaging non acquisisce supplementi. Il collector standalone accetta `--reuse-directory DIRECTORY` per riutilizzare file nella struttura `pacchetto@versione/nome-file`, verificandoli senza rete. La destinazione deve essere nuova; il risultato viene pubblicato soltanto dopo tutti i controlli. Un errore rimuove lo staging temporaneo e preserva le destinazioni precedenti.

Limiti: 16 pacchetti, 64 file, 16 MiB per file/ricetta e 64 MiB complessivi di supplementi; 16 MiB per manifest. HTTPS senza credenziali, con i limiti curl già descritti; dimensione e SHA-256 verificati anche dopo la copia. Versioni, ricette, percorsi discordanti, duplicati, link o hash errati vengono rifiutati. Nessuna patch, ricetta o contenuto degli archivi viene eseguito. Input e staging sono controllati dal costruttore, senza scrittori concorrenti.

Nel bundle l’output è `Contents/Resources/notices/homebrew-supplements`, incluso prima della firma ad hoc. Contiene i quattro materiali, le due ricette installate, il piano, una copia dell’inventario corrente e `attachment.json` con hash e provenienza. Conserva `distribution_ready=false` e `corresponding_sources_complete=false`: questo piano copre GLib/libb2, non tutte le risorse Homebrew, gli ambienti o gli obblighi di distribuzione.

Verifica locale del 2026-09-23: download reale di quattro file/1.093.639 byte e corrispondenza delle due ricette riusciti; 30 regressioni Python passate. Bundle locale con 247 avvisi e supplementi: hash ricontrollati, firma ad hoc e self-test Cocoa superati. Piano errato rifiutato dal packaging completo: eseguibile, allegato e preferenza lingua precedenti invariati, firma precedente valida. CI da verificare.

CI `62a8b54`, run 35910855230: cinque job superati; macOS rifiuta il piano per una versione assente nell’inventario del runner; i log mostrano che GLib non è stato aggiornato come dipendenza transitiva di Qt. Preparazione corretta richiedendo esplicitamente GLib a Homebrew e registrando versioni prima/dopo; hash e versioni del piano restano obbligatori, nessuna accettazione automatica di ricette diverse. CI correttiva da verificare.

Il tentativo `b6467cb` mostra GLib 2.88.3 prima/dopo: il catalogo Homebrew del runner la considera ancora aggiornata. Aggiunto `brew update` esplicito prima dell’installazione; piano 2.90.0 invariato. Verificati anche i pacchetti arm64_sequoia per checksum pubblicato: GLib usa la stessa ricetta revisionata, libb2 una ricetta con stesso sorgente/patch/build ma diversa direttiva no_autobump e delimitatore del test. Il piano enumera entrambi gli hash libb2 verificati, senza normalizzazione o accettazione di hash ulteriori. 31 regressioni locali passate; CI successiva da verificare. [Evidenza del confronto](../evidence/macos-sources-2026-09-23/sequoia-recipe-check.json).
