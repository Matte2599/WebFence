# M0 — Sorgenti delle dipendenze Windows

[English](../en/M0-WINDOWS-SOURCES.md) · [Materiali sorgente](M0-SOURCE-MATERIALS.md) · [Packaging](ADR-007-PACKAGING.md)

## Procedura disponibile

`scripts/collect_windows_sources.py` raccoglie i pacchetti sorgente MSYS2 corrispondenti alle ricette identificate nel `native-build.json` Windows. Richiede Python 3.9+, zstd e curl 8.4+. L’inventario è un input di build fidato, in una directory controllata; non accetta istruzioni da target o AI. Il programma non esegue ricette né installa dipendenze.

```sh
python3 scripts/collect_windows_sources.py \
  PERCORSO/native-build.json \
  /tmp/webfence-windows-source-materials

# Facoltativo: --reuse-directory DIRECTORY_ARCHIVI oppure --zstd PERCORSO
python3 -m unittest discover -s scripts/tests -p 'test_*.py' -v
```

La destinazione deve essere nuova. Sono conservati archivi originali, `PKGBUILD`, `.SRCINFO`, inventario di input e manifest con hash/percorso di ogni membro regolare. `INCOMPLETE` resta presente fino al successo; un errore conserva i materiali parziali e non pubblica il manifest finale. Il riuso ricontrolla l’archivio. Il packaging può includere esplicitamente i materiali, con le opzioni descritte sotto.

Per ciascuna versione, il collector confronta SHA-256 di `PKGBUILD` con quello registrato nel `.BUILDINFO` del pacchetto binario, già confrontato con le DLL distribuite. Deduplica i pacchetti binari generati dalla stessa ricetta e rifiuta hash discordanti. Controlla identità/versione di `.SRCINFO` e checksum dichiarati per sorgenti generici e x86-64; altre architetture non sono dichiarate verificate. `.SRCINFO` viene letto dall’archivio, non rigenerato eseguendo `PKGBUILD`: i controlli attestano la corrispondenza con quei metadati, non un’autenticazione indipendente o una ricostruzione del binario.

URL fissati a `repo.msys2.org/mingw/sources`; solo HTTPS anche dopo redirect, senza `.curlrc` personale. Massimo 64 pacchetti, 256 MiB per archivio, 1 GiB totale; 100.000 membri e 2 GiB dichiarati per archivio, 1 MiB per metadato. Lettura progressiva, nessuna estrazione generale né link seguito. Percorsi pericolosi, membri duplicati, hash errati e budget superati bloccano la raccolta. I trasferimenti usano gli stessi limiti di redirect/tempo/stallo del raccoglitore macOS.

Gli input VCS rimangono `vcs_unverified`; firme con `SKIP` e checksum deboli rimangono `no_strong_checksum`. SHA-1/MD5 vengono eventualmente confrontati, ma non bastano per assegnare `verified`. Gli hash degli archivi esterni sono osservati, non confrontati con firme indipendenti. `distribution_ready` e `corresponding_sources_complete` restano **false**.

## Verifica del 2026-09-23

Raccolti **21 archivi, 316.641.108 byte, 175 membri regolari**, per i 22 pacchetti binari/43 DLL della [CI `054bb2d`](https://github.com/Matte2599/WebFence/actions/runs/35904059667). Tutte le 21 ricette corrispondono agli hash binari; **103 input superano i checksum**, otto firme distaccate hanno `SKIP` e un input Git richiede verifica separata. I checksum dei payload firmati sono verificati; le otto firme PGP non lo sono.

Il [manifest](../evidence/windows-sources-2026-09-23/source-materials.json) e il [piano di input](../evidence/windows-sources-2026-09-23/native-build.input.json) sono conservati in Git. Il piano è ricostruito dai 22 record del log CI, non è una copia dell’inventario binario completo: vedere [provenienza](../evidence/windows-native-provenance-2026-09-23.md). Archivi e ricette restano fuori Git in `/tmp/webfence-windows-sources-054bb2d/`. Verifica indipendente dei 21 hash archivio e 42 file di ricetta/metadati superata. Quattro nuove regressioni sintetiche, 20 complessive, passate localmente; [CI `37da7cf`](https://github.com/Matte2599/WebFence/actions/runs/35907192055) completata: sei job superati, incluse le 20 regressioni Python sui quattro target. I 22 pacchetti Windows/43 DLL e gli hash delle ricette coincidono con la raccolta; download reali e supplemento Git sono verifiche locali, non eseguite dalla CI.

Per winpthreads, [supplemento offline](../evidence/windows-sources-2026-09-23/winpthreads-offline-check.json): copiati soltanto pack/index/rev verificati in un repository bare nuovo, senza configurazione, hook o ref dell’archivio. `git fsck --strict --full` passa; commit `61d40c4c077b82ed2ad22640742bd01e3000e222`. Il tar prodotto da `git -c core.abbrev=no archive --format tar COMMIT` ha SHA-256 `ee1989c086f380e53663ecaebd932b793b59b1a2b660ffc035723df8640559a5`, identico a quello dichiarato. Metodo verificato leggendo `calc_checksum_git` del [pacchetto MSYS2 pacman 6.1.0-25](https://packages.msys2.org/packages/pacman), il cui archivio è stato confrontato con il checksum pubblicato. Nessun checkout, script o collegamento di rete durante il controllo Git; configurazioni globali e attributi esterni disabilitati. È una verifica manuale aggiuntiva: il collector conserva onestamente lo stato `vcs_unverified`.

Restano mappatura dei componenti incorporati e relativi avvisi, ambiente e istruzioni di ricompilazione/sostituzione, assemblaggio della distribuzione e revisione legale. La raccolta non chiude M0-02 né autorizza una release.

Un [registro tecnico dei metadati licenza delle 21 ricette](../evidence/windows-license-metadata-2026-09-24.md) ricontrolla gli hash degli archivi e dei 42 file `PKGBUILD`/`.SRCINFO`, distingue il campo specifico di `libiconv` e indica i casi `custom` o composti da esaminare. I metadati delle ricette non assegnano automaticamente una licenza ai binari o al codice incorporato; la mappatura ai notices inclusi nello ZIP e la revisione legale restano aperte.

## Vincolo agli SHA-256 pubblicati dei binari

Il [registro verificato](../evidence/windows-binary-checksum-lock-2026-09-24.md) fissa nome, versione e SHA-256 pubblicato dei 22 pacchetti MSYS2 che forniscono le 43 DLL. I 22 archivi originali, 55.884.975 byte, sono stati scaricati dalle pagine ufficiali e confrontati con il registro. Durante il packaging, gli archivi della cache devono corrispondere ai checksum revisionati e l'insieme dei proprietari non può cambiare senza aggiornamento esplicito. Lo ZIP conserva il registro e il test PowerShell lo confronta con l'inventario estratto. È un vincolo tecnico sugli artefatti, non una verifica delle firme PGP o della completezza dei notices. Un aggiornamento dei pacchetti MSYS2 può richiedere di rivedere il registro e ricollaudare la build; la [CI `d753f01`](https://github.com/Matte2599/WebFence/actions/runs/35937923381) ha superato sei job al primo tentativo, incluso il controllo dello ZIP Windows estratto.

## Sidecar di attribuzione

La [prova sui 22 archivi vincolati](../evidence/windows-attribution-sidecars-2026-09-24.md) ha ampliato la selezione dei file di avviso e attribuzione da 74 a 136: 62 aggiunte (PCRE2 `AUTHORS.md` e 61 sidecar Qt). Il packaging confronta ora i percorsi scelti dall'installazione con quelli presenti nell'archivio binario, oltre a verificarne gli hash; file omessi, duplicati o non regolari bloccano la build. I 27 riferimenti `LicenseFile` dei JSON Qt selezionati risolvono a file inclusi. Le 62 regressioni Python locali e la prova dei 22 archivi passano; CI del nuovo codice da verificare. Questo migliora i materiali per la revisione, senza dimostrare che i protocolli Wayland siano incorporati nelle DLL Windows o chiudere il gate legale.

## Includere i sorgenti nello ZIP

In MSYS2 UCRT64, aggiungere una delle due opzioni al comando di packaging:

```sh
python3 scripts/package-windows.py "$(cygpath -w /ucrt64)" "$(cygpath -w "$PWD/bin/webfence.exe")" \
  --source-materials "C:/percorso/raccolta-verificata"

# Alternativa esplicita: scaricare, verificare e conservare una nuova raccolta.
python3 scripts/package-windows.py "$(cygpath -w /ucrt64)" "$(cygpath -w "$PWD/bin/webfence.exe")" \
  --collect-sources "C:/percorso/nuova-raccolta"
```

Le opzioni sono mutuamente esclusive. Senza opzioni resta il pacchetto di sviluppo con avvisi installati; `--source-materials` non accede alla rete. `--collect-sources` richiede una directory nuova ed esegue la raccolta dopo aver identificato le DLL reali. In caso di errore, lo ZIP precedente resta intatto e la raccolta parziale resta ispezionabile.

`scripts/attach_windows_sources.py` confronta il piano dei pacchetti dell’inventario corrente con quello di acquisizione. Verifica archivi, hash, budget, ricette e metadati rigenerati; rifiuta percorsi discordanti e link nei materiali. Copia gli archivi e rigenera `PKGBUILD`/`.SRCINFO` dai byte copiati, ignorando le ricette sciolte modificabili. L’allegato viene pubblicato in uno staging nuovo solo dopo tutti i controlli. Input di build fidati, nessun supporto a scrittori concorrenti.

Lo ZIP contiene `WebFence/msys2-sources`: archivi originali con sorgenti upstream, patch e avvisi annidati, ricette, manifest di raccolta, inventari iniziale/corrente, `attachment.json` e README IT/EN. I percorsi archivio del manifest sono ora risolvibili all’interno del pacchetto. Gli archivi compressi aggiungono circa 302 MiB prima della compressione ZIP; non vengono estratti o eseguiti dall’app. L’inventario corrente è legato per SHA-256 a quello delle DLL del pacchetto. I metadati di acquisizione non sono presentati come provenienza dell’eseguibile nuovo; il supplemento Git manuale resta evidenza separata, non è importato automaticamente.

Il packaging confronta inoltre ogni notice installato e selezionato con il membro omonimo dell'archivio binario MSYS2 conservato, registra nel `native-build.json` percorso, hash SHA-256 e membro originale, e rifiuta membri selezionati mancanti, cambiati, duplicati o non regolari nell'archivio. Il collaudo dello ZIP estratto ricontrolla gli hash delle DLL e dei notices e il collegamento ai pacchetti prima di avviare la GUI. Questa verifica prova la corrispondenza dei file **selezionati** con gli archivi usati per la build di sviluppo; non prova che la selezione copra tutti gli avvisi richiesti o che le firme dei pacchetti siano verificate. La regressione sintetica e la suite locale di 58 test passano; [CI `8a8ea85`, sei job verdi](https://github.com/Matte2599/WebFence/actions/runs/35933571589) ha verificato **43 DLL, 22 pacchetti e 74 notices** nello ZIP estratto. [Registro del collaudo](../evidence/windows-notice-linkage-2026-09-24.md).

Verifica locale: 25 regressioni Python superate; inclusione effettiva di 21 archivi/316.641.108 byte e ricontrollo indipendente di 21 hash archivio e 42 ricette/metadati. Output `/tmp/webfence-windows-source-attachment-a146152`, usando il piano ricostruito della precedente CI come input, non un nuovo eseguibile Windows locale. La CI ora prepara lo ZIP da un inventario reale con `--collect-sources`, controlla tutti gli hash dopo l’estrazione, richiede la presenza dei sorgenti e prova il rifiuto di una raccolta incompleta preservando lo ZIP precedente; CI [`69f1dcc`](https://github.com/Matte2599/WebFence/actions/runs/35908880543) completata: sei job superati, 25 regressioni Python sui quattro target; su Windows raccolti/inclusi 21 archivi, verificati gli hash nello ZIP estratto, preservato lo ZIP su errore e superati self-test/soak offscreen e nativi con PATH di solo sistema.

Da PowerShell 7, la prova completa è `./scripts/test-windows-package.ps1 -Archive ./dist/webfence-windows-amd64.zip -RequireSources`. I controlli dei sorgenti precedono quelli GUI con PATH di solo sistema. L’assemblaggio conserva `distribution_ready=false` e `corresponding_sources_complete=false`: avvisi incorporati, firme, ambiente di build e ricompilazione/sostituzione richiedono ancora revisione.

ZIP effimero della CI, non pubblicato: SHA-256 `47247e77a6f543ae670a69c67914e815ba0d15f6e701df8f240a20331e860ff5`.
