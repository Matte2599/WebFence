# M0 — Import Qt Windows e dipendenze di sistema / Windows Qt imports and system dependencies

[Matrice IT](../it/M0-VALIDATION.md) · [Matrix EN](../en/M0-VALIDATION.md) · [Sorgenti IT](../it/M0-WINDOWS-SOURCES.md) · [Sources EN](../en/M0-WINDOWS-SOURCES.md)

## Italiano

**Prova tecnica parziale del 2026-09-24.** Il pacchetto sorgente `mingw-w64-qt6-base-6.11.2-2` (SHA-256 `81a47ca828f9f2f9f88e2d341233fddf2073ae824fa34c123fe3f3911344a0e6`) è quello del [manifest Windows verificato](windows-signatures-2026-09-24/source-materials.json). Il suo `PKGBUILD` ha SHA-256 `42bd77bdf7d864af917e4cf415736a4f23ce708ef9809135125d64a6d9999c65`, identico al campo `pkgbuild_sha256sum` nel `.BUILDINFO` dell'[archivio binario Qt 6.11.2-2 vincolato e firmato](windows-binary-signatures-2026-09-24.md), SHA-256 `ccc98698391f78419de72640175508a68b7ba9e25a43a54e01a05b51079ad6a2`. Le due letture sono state fatte senza eseguire la ricetta o i binari.

La ricetta imposta `FEATURE_system_*=ON` per otto dipendenze. Dall'archivio binario originale sono stati letti componenti PE regolari e i loro import con Apple LLVM `objdump -p` 21.0.0:

| Flag della ricetta | Componente nel pacchetto Qt | Import osservato |
| --- | --- | --- |
| `system_doubleconversion` | `Qt6Core.dll` | `libdouble-conversion.dll` |
| `system_freetype` | `Qt6Gui.dll` | `libfreetype-6.dll` |
| `system_jpeg` | `qjpeg.dll` | `libjpeg-8.dll` |
| `system_harfbuzz` | `Qt6Gui.dll` | `libharfbuzz-0.dll` |
| `system_pcre2` | `Qt6Core.dll` | `libpcre2-16-0.dll` |
| `system_png` | `Qt6Gui.dll` | `libpng16-16.dll` |
| `system_sqlite` | `qsqlite.dll` | `libsqlite3-0.dll` |
| `system_zlib` | `Qt6Core.dll` | `zlib1.dll` |

Gli hash SHA-256 dei componenti esaminati sono: `Qt6Core.dll` `a54f0e354ff38670df336267333333117d623f7d7a6811e98c345fd68f339b1f`, `Qt6Gui.dll` `2b12d68d2dbd2171b9dc221884baf2e7a5769984156a42ef1572ad298d427a68`, `qjpeg.dll` `ec2965dd10bde9e5253ea6496e168aac4fe69eac87b8565416661536b0bfc5d0` e `qsqlite.dll` `21618638663f338ac67776537907b96b899d96585106b67f69b6faada2bd797b`. `Qt6Core.dll` importa inoltre `libzstd.dll`. Questi hash identificano i membri dell'archivio originale, non da soli i file effettivamente presenti nello ZIP WebFence.

**Limiti:** il collegamento tra ricetta, `.BUILDINFO` e import PE indica dipendenze dinamiche per questi componenti del pacchetto Qt; non dimostra che ogni componente sia incluso nello ZIP, che non esistano altri componenti incorporati, o che il solo import rappresenti una scelta di licenza. In particolare `qsqlite.dll` è un plugin del pacchetto Qt, non è qui dichiarato distribuito da WebFence. La mappatura deve essere confrontata con un inventario concreto dello ZIP e sottoposta a revisione legale; M0-02/06 restano aperti.

## English

**Partial technical trial on 2026-09-24.** Source package `mingw-w64-qt6-base-6.11.2-2` (SHA-256 `81a47ca828f9f2f9f88e2d341233fddf2073ae824fa34c123fe3f3911344a0e6`) matches the [verified Windows manifest](windows-signatures-2026-09-24/source-materials.json). Its `PKGBUILD` SHA-256 `42bd77bdf7d864af917e4cf415736a4f23ce708ef9809135125d64a6d9999c65` equals `pkgbuild_sha256sum` in `.BUILDINFO` from the [locked and signed Qt 6.11.2-2 binary archive](windows-binary-signatures-2026-09-24.md), SHA-256 `ccc98698391f78419de72640175508a68b7ba9e25a43a54e01a05b51079ad6a2`. Neither recipe nor binary was executed to collect this evidence.

The recipe sets `FEATURE_system_*=ON` for eight dependencies. Regular PE members of the original binary archive were read and their imports inspected with Apple LLVM `objdump -p` 21.0.0:

| Recipe flag | Component in Qt package | Observed import |
| --- | --- | --- |
| `system_doubleconversion` | `Qt6Core.dll` | `libdouble-conversion.dll` |
| `system_freetype` | `Qt6Gui.dll` | `libfreetype-6.dll` |
| `system_jpeg` | `qjpeg.dll` | `libjpeg-8.dll` |
| `system_harfbuzz` | `Qt6Gui.dll` | `libharfbuzz-0.dll` |
| `system_pcre2` | `Qt6Core.dll` | `libpcre2-16-0.dll` |
| `system_png` | `Qt6Gui.dll` | `libpng16-16.dll` |
| `system_sqlite` | `qsqlite.dll` | `libsqlite3-0.dll` |
| `system_zlib` | `Qt6Core.dll` | `zlib1.dll` |

SHA-256 values for inspected components: `Qt6Core.dll` `a54f0e354ff38670df336267333333117d623f7d7a6811e98c345fd68f339b1f`, `Qt6Gui.dll` `2b12d68d2dbd2171b9dc221884baf2e7a5769984156a42ef1572ad298d427a68`, `qjpeg.dll` `ec2965dd10bde9e5253ea6496e168aac4fe69eac87b8565416661536b0bfc5d0` and `qsqlite.dll` `21618638663f338ac67776537907b96b899d96585106b67f69b6faada2bd797b`. `Qt6Core.dll` also imports `libzstd.dll`. These hashes identify members of the original archive, not by themselves files actually shipped in the WebFence ZIP.

**Limits:** recipe, `.BUILDINFO` and PE imports show dynamic dependencies for these Qt package components; they do not prove that every component is in the ZIP, rule out other embedded code, or determine a license choice from an import alone. In particular `qsqlite.dll` is a plugin in the Qt package and is not claimed here as distributed by WebFence. Compare this mapping against an actual ZIP inventory and obtain legal review; M0-02/06 remain open.
