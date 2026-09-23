# M0 — Avvisi Windows collegati agli archivi / Windows notices bound to archives

[Fonti IT](../it/M0-WINDOWS-SOURCES.md) · [Sources EN](../en/M0-WINDOWS-SOURCES.md) · [Matrice IT](../it/M0-VALIDATION.md) · [Matrix EN](../en/M0-VALIDATION.md)

## Italiano

**Esito tecnico sul commit `8a8ea859e8bc4933e554135488da1966d1a2bbff`:** [CI 35933571589](https://github.com/Matte2599/WebFence/actions/runs/35933571589), sei job superati al primo tentativo. Nel [job Windows 107425452758](https://github.com/Matte2599/WebFence/actions/runs/35933571589/job/107425452758), il packaging ha confrontato i bytes dei notices selezionati e delle DLL installate con i membri dei pacchetti binari MSYS2 conservati. L'inventario ha registrato percorso, SHA-256 e membro dell'archivio per ogni notice, oltre al pacchetto proprietario di ciascuna DLL.

Il collaudo sullo ZIP estratto ha superato il controllo di **43 DLL, 22 pacchetti proprietari e 74 notices** collegati agli archivi; l'allegato dei **21 archivi sorgente/316.641.108 byte** è stato ricontrollato per hash, inventario e ricette. Il tentativo con raccolta sorgenti incompleta ha preservato lo ZIP precedente. Self-test e soak breve offscreen/Windows nativo sono passati da una cartella con spazi e Unicode, con solo i percorsi di sistema nel PATH. Sul job Windows sono passati anche 58 test Python. Lo ZIP effimero del runner aveva SHA-256 `42bcdce1d839787ee2a66a1dba7cce6faeea238d6037b5fcaf61538e4d958794`; non è una release pubblicata.

**Limite:** la selezione è basata sui percorsi dei notices presenti nell'installazione MSYS2. La corrispondenza con l'archivio non dimostra che copra tutte le attribuzioni per componenti incorporati, né verifica firme dei pacchetti, diritti di distribuzione, ambiente di ricostruzione, Windows 10/11 desktop o NVDA. `distribution_ready=false` e `corresponding_sources_complete=false` restano corretti; M0-02 e M0-06 sono aperti.

## English

**Technical result on commit `8a8ea859e8bc4933e554135488da1966d1a2bbff`:** [CI 35933571589](https://github.com/Matte2599/WebFence/actions/runs/35933571589), six jobs passed on the first attempt. In [Windows job 107425452758](https://github.com/Matte2599/WebFence/actions/runs/35933571589/job/107425452758), packaging compared bytes of selected notices and installed DLLs with members of retained MSYS2 binary packages. The inventory recorded path, SHA-256 and archive member for each notice, alongside each DLL's owning package.

The extracted-ZIP trial passed checks for **43 DLLs, 22 package owners and 74 archive-matched notices**; the attachment of **21 source archives/316,641,108 bytes** was rechecked for hashes, inventory and recipes. An incomplete source collection trial preserved the previous ZIP. Offscreen and native Windows short self-tests/soaks passed from a Unicode/spaced directory with system paths only in PATH. The Windows job also passed 58 Python tests. The runner's ephemeral ZIP had SHA-256 `42bcdce1d839787ee2a66a1dba7cce6faeea238d6037b5fcaf61538e4d958794`; it is not a published release.

**Limit:** selection uses notice paths present in the MSYS2 installation. Archive correspondence does not prove coverage of every attribution for embedded components, or verify package signatures, distribution rights, rebuild environment, Windows 10/11 desktop behavior or NVDA. `distribution_ready=false` and `corresponding_sources_complete=false` remain correct; M0-02 and M0-06 remain open.
