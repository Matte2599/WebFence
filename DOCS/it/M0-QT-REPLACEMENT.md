# M0 — Ricompilazione e sostituzione del plugin Qt Cocoa

[English](../en/M0-QT-REPLACEMENT.md) · [Indice](../README.md) · [Decisione Cocoa](ADR-006-QT-COCOA.md)

Procedura tecnica per il bundle di sviluppo macOS ARM64 con Qt 6.11.2. Dimostra che una modifica del plugin Cocoa può essere ricompilata e caricata mantenendo la stessa build Go. Non dimostra la ricompilazione completa di Qt, compatibilità con altre versioni/private ABI o conformità legale del pacchetto. M0-02 e revisione legale restano aperti.

## Materiali e prerequisiti

Usare un bundle di sviluppo fidato e gli strumenti descritti nella [guida di sviluppo](DEVELOPMENT.md): macOS Apple Silicon, strumenti Apple, Go, Qt 6.11.2 con header privati corrispondenti, CMake, Ninja e header MoltenVK/Vulkan. L'esecuzione richiede una sessione desktop Cocoa. Il minimo OS dipende dai [binari del bundle](M0-PACKAGING.md).

Dentro `Contents/Resources/notices/scripts` sono inclusi il [builder](../../scripts/build-qt-cocoa.sh), il [collaudo](../../scripts/test-qt-cocoa-replacement.py) e `qt-cocoa/` con CMake, patch e licenze. Il builder acquisisce il sorgente ufficiale Qt oppure riusa `WEBFENCE_QT_SOURCE_ARCHIVE`, verificando sempre SHA-256 prima dell'estrazione. URL, hash e attribuzioni sono nel [README della patch](../../scripts/qt-cocoa/README.md). Le librerie Qt installate forniscono l'ambiente di compilazione; non sono sostituite.

## Prova ripetibile

Dalla radice del repository, dopo avere prodotto il bundle con `sh scripts/package-macos.sh`:

```sh
app="$PWD/dist/WebFence.app"
trial_dir="$(mktemp -d /tmp/webfence-replacement.XXXXXX)/trial"
python3 "$app/Contents/Resources/notices/scripts/test-qt-cocoa-replacement.py" \
  "$app" "$(brew --prefix qtbase)" "$trial_dir"
cat "$trial_dir/result.json"
```

La directory finale deve essere nuova, fuori dal bundle e da Qt. Conservare il prefisso restituito da Homebrew: risolverlo manualmente a un percorso `Cellar` può cambiare i nomi dei framework attesi dal builder. Per lavorare senza scaricare nuovamente l'archivio Qt, impostare `WEBFENCE_QT_SOURCE_ARCHIVE` al file già verificato dal raccoglitore sorgenti. Il collaudo non usa il repository per ricostruire il plugin: copia i materiali inclusi nell'app.

La procedura:

1. Verifica la firma del bundle originale, registra hash dei file/link e copia l'app in una directory separata con un nome contenente spazi.
2. Copia i materiali inclusi, aggiunge solo nella copia della patch il messaggio `WEBFENCE_QT_REPLACEMENT_TRIAL` all'inizializzazione Cocoa e ricompila il plugin. La correzione assistiva resta applicata.
3. Sostituisce `Contents/PlugIns/platforms/libqcocoa.dylib` nella copia, conserva i collegamenti ai framework del bundle e firma nuovamente ad hoc. Non ricompila o ricollega Go; confronta il Go build ID, dato che la firma può cambiare byte dell'eseguibile.
4. Esegue self-test e collaudo sintetico di dieci secondi con backend Cocoa e ambiente Qt/DYLD ripulito. Richiede uscita zero e un messaggio diagnostico per esecuzione: prova che il plugin modificato è effettivamente caricato.
5. Ricontrolla il bundle originale completo, il plugin Qt installato e l'eventuale preferenza personale della lingua. Self-test e soak usano già preferenze temporanee e copia negli appunti intercettata.

`result.json` e i log esterni all'app descrivono l'esito finale. La copia conserva l'inventario del bundle base e aggiunge un `replacement-trial.json` prima della firma: quel sidecar descrive la variante **prima** dei test, con `passed=false`. La copia è soltanto un artefatto di collaudo, non un pacchetto distribuibile con inventario aggiornato. Tornare al bundle originale equivale al ripristino; non viene sostituita l'app originale.

## Variante diagnostica AX delle righe

L'opzione `--ax-direct-rows` aggiunge **solo alla copia privata** una seconda modifica Cocoa: `accessibilityRows` restituisce l'array di righe sintetiche direttamente, senza `NSAccessibilityUnignoredChildren`. La [prova CI](../evidence/macos-ax-table-reset-2026-09-24.md) ha escluso questa singola modifica come correzione: il client Swift AX fallisce nella stessa quantità di stadi sul bundle normale e sulla variante su entrambi i runner macOS. L'opzione resta per riprodurre l'esperimento manualmente; non fa parte della CI ordinaria né del plugin distribuito. Compilazione, firma, self-test e soak restano obbligatori per ogni copia privata. Un self-test Qt positivo da solo non basta.

```sh
python3 "$app/Contents/Resources/notices/scripts/test-qt-cocoa-replacement.py" \
  "$app" "$(brew --prefix qtbase)" "$trial_dir-direct-rows" --ax-direct-rows
swiftc scripts/test-macos-ax-reset.swift -o "$trial_dir-ax-client"
"$trial_dir-ax-client" "$trial_dir-direct-rows/WebFence replacement trial.app"
```

Il parametro `ax_direct_rows` in `result.json` distingue le due varianti. La CI scarica l'archivio Qt fissato una volta per il bundle e la ricompilazione ordinaria; ogni invocazione del builder ne verifica comunque lo SHA-256. Il client usa solo fixture sintetiche e HOME temporanea; il bundle originale, il plugin installato e la lingua personale devono restare invariati. La copia privata conserva l'inventario della build di base e non va distribuita.

## Evidenze e limiti

[Prova locale del 2026-09-23](../evidence/macos-cocoa-replacement-2026-09-23/result.json) superata su macOS 26.6.2 ARM64: variante ricompilata dai materiali inclusi, firma valida, stesso Go build ID, self-test e quattro cicli in 12,003 s; messaggio diagnostico rilevato in entrambe le esecuzioni. Bundle originale, plugin Qt installato e lingua personale invariati. Archivio invalido respinto prima dell’estrazione; output preesistente rifiutato preservando il risultato. Il [contesto](../evidence/macos-cocoa-replacement-2026-09-23/context.json) distingue il bundle precedente con modifiche locali dal nuovo script di collaudo e registra il primo tentativo fallito sul prefisso Homebrew. [CI `3f68e6a`](https://github.com/Matte2599/WebFence/actions/runs/35916839178) completata: sei job superati. Su macOS 15.7.9 la procedura inclusa nel bundle ricompila/carica la variante Cocoa, mantiene il Go build ID e supera firma, self-test e soak; originali preservati. Le 36 regressioni Python e i controlli dei pacchetti Windows/Debian sono verdi. La run precedente `51673d9` (35916784359) è stata annullata dal push successivo, che codifica lo spazio di contesto della patch senza cambiarne i byte prodotti. Un successo tecnico non approva LICENSE/CLA né chiude la revisione della distribuzione. Restano da completare materiali corrispondenti e mappatura delle dipendenze incorporate, procedure per le altre librerie/piattaforme, lettori reali, desktop puliti e sistemi minimi. Nessuna firma Developer ID/notarizzazione o comportamento sotto hardened runtime è verificato da questa firma ad hoc.
