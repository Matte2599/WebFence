# M0 — Sorgenti delle dipendenze Windows

[English](../en/M0-WINDOWS-SOURCES.md) · [Materiali sorgente](M0-SOURCE-MATERIALS.md) · [Packaging](ADR-007-PACKAGING.md)

## Procedura disponibile

`scripts/collect_windows_sources.py` raccoglie i pacchetti sorgente MSYS2 corrispondenti alle ricette identificate nel `native-build.json` Windows. Richiede Python 3.9+, zstd e curl 8.4+. L’inventario è un input di build fidato, in una directory controllata; non accetta istruzioni da target o AI. Il programma non esegue ricette né installa dipendenze.

```sh
python3 scripts/collect_windows_sources.py \
  PERCORSO/notices/native-build.json \
  /tmp/webfence-windows-source-materials

# Facoltativo: --reuse-directory DIRECTORY_ARCHIVI oppure --zstd PERCORSO
python3 -m unittest discover -s scripts/tests -p 'test_*.py' -v
```

La destinazione deve essere nuova. Sono conservati archivi originali, `PKGBUILD`, `.SRCINFO`, inventario di input e manifest con hash/percorso di ogni membro regolare. `INCOMPLETE` resta presente fino al successo; un errore conserva i materiali parziali e non pubblica il manifest finale. Il riuso ricontrolla l’archivio. Non c’è inclusione automatica nel pacchetto Windows.

Per ciascuna versione, il collector confronta SHA-256 di `PKGBUILD` con quello registrato nel `.BUILDINFO` del pacchetto binario, già confrontato con le DLL distribuite. Deduplica i pacchetti binari generati dalla stessa ricetta e rifiuta hash discordanti. Controlla identità/versione di `.SRCINFO` e checksum dichiarati per sorgenti generici e x86-64; altre architetture non sono dichiarate verificate. `.SRCINFO` viene letto dall’archivio, non rigenerato eseguendo `PKGBUILD`: i controlli attestano la corrispondenza con quei metadati, non un’autenticazione indipendente o una ricostruzione del binario.

URL fissati a `repo.msys2.org/mingw/sources`; solo HTTPS anche dopo redirect, senza `.curlrc` personale. Massimo 64 pacchetti, 256 MiB per archivio, 1 GiB totale; 100.000 membri e 2 GiB dichiarati per archivio, 1 MiB per metadato. Lettura progressiva, nessuna estrazione generale né link seguito. Percorsi pericolosi, membri duplicati, hash errati e budget superati bloccano la raccolta. I trasferimenti usano gli stessi limiti di redirect/tempo/stallo del raccoglitore macOS.

Gli input VCS rimangono `vcs_unverified`; firme con `SKIP` e checksum deboli rimangono `no_strong_checksum`. SHA-1/MD5 vengono eventualmente confrontati, ma non bastano per assegnare `verified`. Gli hash degli archivi esterni sono osservati, non confrontati con firme indipendenti. `distribution_ready` e `corresponding_sources_complete` restano **false**.

## Verifica del 2026-09-23

Raccolti **21 archivi, 316.641.108 byte, 175 membri regolari**, per i 22 pacchetti binari/43 DLL della [CI `054bb2d`](https://github.com/Matte2599/WebFence/actions/runs/35904059667). Tutte le 21 ricette corrispondono agli hash binari; **103 input superano i checksum**, otto firme distaccate hanno `SKIP` e un input Git richiede verifica separata. I checksum dei payload firmati sono verificati; le otto firme PGP non lo sono.

Il [manifest](../evidence/windows-sources-2026-09-23/source-materials.json) e il [piano di input](../evidence/windows-sources-2026-09-23/native-build.input.json) sono conservati in Git. Il piano è ricostruito dai 22 record del log CI, non è una copia dell’inventario binario completo: vedere [provenienza](../evidence/windows-native-provenance-2026-09-23.md). Archivi e ricette restano fuori Git in `/tmp/webfence-windows-sources-054bb2d/`. Verifica indipendente dei 21 hash archivio e 42 file di ricetta/metadati superata. Quattro nuove regressioni sintetiche, 20 complessive, passate localmente; CI delle nuove modifiche da verificare.

Per winpthreads, [supplemento offline](../evidence/windows-sources-2026-09-23/winpthreads-offline-check.json): copiati soltanto pack/index/rev verificati in un repository bare nuovo, senza configurazione, hook o ref dell’archivio. `git fsck --strict --full` passa; commit `61d40c4c077b82ed2ad22640742bd01e3000e222`. Il tar prodotto da `git -c core.abbrev=no archive --format tar COMMIT` ha SHA-256 `ee1989c086f380e53663ecaebd932b793b59b1a2b660ffc035723df8640559a5`, identico a quello dichiarato. Metodo verificato leggendo `calc_checksum_git` del [pacchetto MSYS2 pacman 6.1.0-25](https://packages.msys2.org/packages/pacman), il cui archivio è stato confrontato con il checksum pubblicato. Nessun checkout, script o collegamento di rete durante il controllo Git; configurazioni globali e attributi esterni disabilitati. È una verifica manuale aggiuntiva: il collector conserva onestamente lo stato `vcs_unverified`.

Restano mappatura dei componenti incorporati e relativi avvisi, ambiente e istruzioni di ricompilazione/sostituzione, assemblaggio della distribuzione e revisione legale. La raccolta non chiude M0-02 né autorizza una release.
