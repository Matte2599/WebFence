# Materiali sorgente Qt Windows / Windows Qt source materials

Data / Date: 2026-09-23. Package: `mingw-w64-ucrt-x86_64-qt6-base`, version `6.11.2-2`.

**IT:** Archivio indicato dalla [pagina ufficiale MSYS2](https://packages.msys2.org/packages/mingw-w64-ucrt-x86_64-qt6-base), acquisito tramite HTTPS, fuori da Git. La versione coincide con l’installazione registrata nel job Windows 107309521033 della CI `56840c5`. Contiene ricetta PKGBUILD, .SRCINFO, sorgente Qt e dieci patch. Gli undici SHA-256 dichiarati in .SRCINFO corrispondono ai file; l’archivio Qt interno coincide anche con il sorgente upstream già verificato per macOS. Nessuna ricetta eseguita, patch applicata o libreria installata.

**EN:** Archive listed by the [official MSYS2 page](https://packages.msys2.org/packages/mingw-w64-ucrt-x86_64-qt6-base), acquired over HTTPS, outside Git. Its version matches the installation recorded in Windows job 107309521033 of CI `56840c5`. Contains PKGBUILD, .SRCINFO, Qt source and ten patches. All eleven SHA-256 values in .SRCINFO match their files; the inner Qt archive also matches the upstream source already verified for macOS. No recipe executed, patch applied or library installed.

- Source URL: `https://mirror.msys2.org/mingw/sources/mingw-w64-qt6-base-6.11.2-2.src.tar.zst`
- Local archive: `/tmp/webfence-msys2-qt6-base-6.11.2-2.src.tar.zst`
- Bytes: 50590681
- Observed SHA-256: `81a47ca828f9f2f9f88e2d341233fddf2073ae824fa34c123fe3f3911344a0e6`
- Local ledger: `/tmp/webfence-msys2-qt6-base-materials/ledger.json`

**Limiti / Limits:** hash esterno osservato localmente, firma distaccata e autenticazione indipendente della ricetta non verificate. Il confronto interno non prova da solo l’autenticità delle patch. Raccolta Windows completa, altre DLL, revisione licenze, riproduzione build e integrazione ancora aperte. / Outer hash observed locally; detached signature and independent recipe authentication not verified. Internal matching alone does not prove patch authenticity. Complete Windows collection, other DLLs, license review, rebuild and integration remain open.

| File | Bytes | SHA-256 |
| --- | ---: | --- |
| `.SRCINFO` | 4326 | `a7870f4bb8344e53229e6f036a8381141120ccbc4ad2808455a136d174c97e76` |
| `003-adjust-qmake-conf-mingw.patch` | 2056 | `68156b8b7717a0ce19c4b991942469171bfa048cd5c90765115a546e65669a1d` |
| `004-qt-6.2.0-win32-g-Add-QMAKE_EXTENSION_IMPORTLIB-defaulting-to-.patch` | 7486 | `ed5b61bcb367bbda459bec903d796ea45604278f577a988d602ade07ec6bf363` |
| `005-qt-6.7.0-opengl-header.patch` | 1356 | `a2afc74d181864409dc96eca368b647c0f79e25751db88e3263f2d1101edf8e4` |
| `006-qt-6.2.0-dont-add-resource-files-to-qmake-libs.patch` | 432 | `4085a10b290b8e3d930de535cbad2ba3e643432cba433aa2b28fe664f86d38a3` |
| `008-freetype-fonts-fallback-dir.patch` | 1314 | `e2fbd970a20773f0d914f6ffc96aafc8212192227577ec007a460e35398038bf` |
| `009-qfileinfo-undefine-mingw-stat.patch` | 238 | `f4261d43a142a24e5fa3b23e25813754839db84078cc8c6dc611139bf531e64a` |
| `010-export-some-constexpr-variables.patch` | 1614 | `2cd0b791209d53ce5a0ff2afc76cb1aedbdc1a92fe406c27bc6e5cd2f4ed0ad4` |
| `011-qt6-windeployqt-fixes.patch` | 1536 | `9d9131f412f0505c38449718fcf2fcd4ae548563eb94b71d55c8238efffbbe5a` |
| `012-qt6-windeployqt-ignore-debug.patch` | 849 | `ce3449f8202766f1ceab9b44a0e94efdbb6b6a1c833381c93136a3786de1bea7` |
| `013-qt6-windeployqt-qmlimportscanner-path.patch` | 863 | `c4783da805c1189c747dc899fb4275a593b2e08522d3af88e74b3dec42d9b4fe` |
| `PKGBUILD` | 9794 | `42bd77bdf7d864af917e4cf415736a4f23ce708ef9809135125d64a6d9999c65` |
| `qtbase-everywhere-src-6.11.2.tar.xz` | 50582668 | `5b2e00eccaf5a4d8c14134ffa0ea8dfd0a35ae1ffc7f8d87fa4305a1ed23cf22` |

## Collegamento al pacchetto binario / Binary package binding

**IT:** Acquisito il pacchetto binario Qt 6.11.2-2 (17.003.163 byte) e verificato SHA-256 `ccc98698391f78419de72640175508a68b7ba9e25a43a54e01a05b51079ad6a2` rispetto alla pagina ufficiale MSYS2 sopra citata. L’hash PKGBUILD nel suo BUILDINFO è `42bd77bdf7d864af917e4cf415736a4f23ce708ef9809135125d64a6d9999c65`, identico alla ricetta dell’archivio sorgente. Questo aggiunge una corrispondenza indipendente dai checksum interni dell’archivio sorgente; non è verifica di firma distaccata. Lettore provato sulle tre DLL QtCore/Gui/Widgets contenute nell’archivio; la successiva CI Windows `054bb2d` ha confrontato tutte le 15 DLL Qt incluse con l’archivio binario.

**EN:** Acquired the Qt 6.11.2-2 binary package (17,003,163 bytes), verifying SHA-256 `ccc98698391f78419de72640175508a68b7ba9e25a43a54e01a05b51079ad6a2` against the official MSYS2 page cited above. Its BUILDINFO PKGBUILD hash is `42bd77bdf7d864af917e4cf415736a4f23ce708ef9809135125d64a6d9999c65`, matching the source-archive recipe. This adds a match independent of internal source-archive checksums; it is not detached-signature verification. Reader exercised on the archived QtCore/Gui/Widgets DLLs; subsequent Windows CI `054bb2d` matched all 15 included Qt DLLs against the binary archive.

Local binary archive: `/tmp/webfence-msys2-qt6-base-6.11.2-2.pkg.tar.zst`; ledger and metadata: `/tmp/webfence-msys2-qt6-base-materials/`.
