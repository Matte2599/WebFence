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

[Manifest e hash della raccolta](../evidence/macos-sources-2026-09-23/source-materials.json) e [inventario di input](../evidence/macos-sources-2026-09-23/native-build.input.json) sono conservati nel repository; gli archivi e i testi raccolti restano in `/tmp/webfence-native-sources-8d3053f-reviewed/`, fuori da Git. Sette test sintetici passati: conservazione dei file esistenti, hash errato, percorsi e duplicati, limiti, riferimenti mancanti, metadati non validi e requisiti curl. Aggiunti alla CI sui quattro target; esito della nuova modifica da verificare.

Verificati indipendentemente tutti gli hash di archivi e avvisi. Trasferimento GLib fresco con la configurazione curl finale riuscito; un limite di 1 KiB rifiuta il download senza salvare byte. Il curl locale segnala questo rifiuto con codice 56 e messaggio di dimensione massima superata: il raccoglitore tratta qualsiasi uscita non zero come fallimento, senza dipendere da un codice specifico.

## Confini ancora aperti

Il manifest mantiene **`distribution_ready: false` e `corresponding_sources_complete: false`**. Sono sorgenti upstream delle librerie identificate, non ancora tutti i sorgenti corrispondenti della distribuzione: servono patch/risorse dei package manager, ambiente e istruzioni di ricompilazione/sostituzione. La patch GLib `Patches/glib/hardcoded-paths.diff` e la risorsa gobject-introspection della ricetta sono esempi ancora da acquisire con provenienza esatta.

Gli avvisi dell’intero albero sorgente possono includere componenti non distribuiti; occorre mapparli ai binari reali e scegliere/verificare i termini applicabili. Le ricette, le SBOM e la patch Cocoa restano nel bundle originale identificato dall’inventario. La directory raccolta non è incorporata automaticamente nel pacchetto né pubblicata come release. Revisione legale, minimi OS, prove desktop pulite e materiali Windows restano parte di M0-02/M0-06; raccogliere file non li approva.
