# M0 — Sidecar di attribuzione Windows / Windows attribution sidecars

[Fonti IT](../it/M0-WINDOWS-SOURCES.md) · [Sources EN](../en/M0-WINDOWS-SOURCES.md) · [Matrice IT](../it/M0-VALIDATION.md) · [Matrix EN](../en/M0-VALIDATION.md)

## Italiano

**Prova locale del 2026-09-24; CI del codice modificato da verificare.** I 22 archivi binari UCRT64 originali, già vincolati agli SHA-256 [pubblicati nel registro](windows-binary-checksum-lock-2026-09-24.md), sono stati riletti senza installare o eseguire i loro contenuti. La regola precedente selezionava **74** file sotto `ucrt64/share`; la nuova seleziona **136** membri regolari: i 74 precedenti più **62** sidecar. I nuovi file sono `AUTHORS.md` di PCRE2 e, nel pacchetto Qt, 27 `qt_attribution.json`, 30 `REUSE.toml`, tre testi di licenza e un `README` della sottocartella dei protocolli Wayland. I 27 riferimenti `LicenseFile` nei JSON Qt risolvono tutti a file presenti nell'archivio e selezionati. Il controllo locale tramite `inspect` ha verificato byte/hash dei 136 file e identità/metadati dei 22 archivi; 62 test Python locali passano.

Il packaging confronta ora l'insieme dei percorsi selezionati nell'installazione con quello ricavato **indipendentemente dall'archivio binario**: un sidecar mancante nella copia installata o nell'inventario fa fallire la build, così come un candidato non regolare o duplicato. Gli hash dei file selezionati restano nel `native-build.json` e sono ricontrollati nello ZIP estratto. La selezione riguarda metadati e avvisi, non i file XML dei protocolli o i file CMake di build; questi ultimi non sono parte del runtime Windows confezionato.

**Limite:** includere i sidecar Qt non dimostra che i protocolli Wayland siano incorporati nelle DLL Windows, né attribuisce una licenza definitiva ai binari o prova la completezza per componenti incorporati. Non sono verificate firme PGP né condizioni legali di distribuzione. `distribution_ready=false` resta corretto e M0-02/M0-06 rimangono aperti.

## English

**Local trial on 2026-09-24; changed-code CI still to verify.** The 22 original UCRT64 binary archives, already bound to [published SHA-256 values](windows-binary-checksum-lock-2026-09-24.md), were reread without installing or executing their contents. The previous rule selected **74** files under `ucrt64/share`; the new rule selects **136** regular members: the original 74 plus **62** sidecars. The additions are PCRE2's `AUTHORS.md` and, in the Qt package, 27 `qt_attribution.json` files, 30 `REUSE.toml` files, three license texts and one `README` from the Wayland protocol subtree. All 27 `LicenseFile` references in the Qt JSON files resolve to present, selected archive members. The local `inspect` trial verified bytes/hashes of all 136 files and the identities/build metadata of all 22 archives; 62 local Python tests pass.

Packaging now compares the set of paths selected from the installation with the set derived **independently from the binary archive**: a sidecar missing from the installed copy or inventory fails the build, as does a nonregular or duplicate candidate. Selected-file hashes remain in `native-build.json` and are rechecked in the extracted ZIP. The selection covers metadata and notices, not protocol XML files or CMake build files; those are outside the packaged Windows runtime.

**Limit:** shipping Qt sidecars does not establish that Wayland protocols are embedded in Windows DLLs, assign final licenses to binaries, or prove completeness for embedded components. PGP signatures and legal distribution terms are unverified. `distribution_ready=false` remains appropriate; M0-02/M0-06 stay open.
