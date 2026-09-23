# M0 — Metadati licenza delle ricette Windows / Windows recipe license metadata

[Sorgenti IT](../it/M0-WINDOWS-SOURCES.md) · [Sources EN](../en/M0-WINDOWS-SOURCES.md) · [Dossier legale IT](../it/LEGAL-REVIEW.md) · [Legal dossier EN](../en/LEGAL-REVIEW.md)

## Italiano

**Stato: registro tecnico per la revisione, non parere legale.** Il 2026-09-24 sono stati ricontrollati gli SHA-256 dei **21 archivi sorgente MSYS2** e dei relativi **21 `PKGBUILD` e 21 `.SRCINFO`** contro il [manifest conservato](windows-sources-2026-09-23/source-materials.json), SHA-256 `81d63614445faf11dbaf0274d467dd1f970abf19f458b43ccda9d52e28e21fb4`. I materiali sono quelli della raccolta locale `/tmp/webfence-windows-sources-054bb2d/`; nessuna ricetta è stata eseguita. Le 21 ricette corrispondono ai **22 pacchetti binari/43 DLL** della [provenienza Windows](windows-native-provenance-2026-09-23.md), con GCC condiviso da `libgcc` e `libstdc++`. Il piano d'input Git, SHA-256 `1d6c259798e2e9ab68123a3d6dc248fb1a2cc42c9f308db78a27f4ff5d0bba8b`, è ricostruito dai log CI: non sostituisce l'inventario completo del pacchetto.

La tabella trascrive il campo `license` della sezione `pkgbase` di `.SRCINFO` per ogni ricetta, eccetto `libiconv`, per cui indica il campo della sottosezione del pacchetto **effettivamente presente**. I campi sono **dichiarazioni MSYS2 per i sorgenti/pacchetti**, non licenze assegnate da WebFence a ciascuna DLL o a ogni componente incorporato. L'espressione Qt include materiale terzo e non prova che ciascuna voce sia presente nel binario Windows. Le eccezioni di altre sottosezioni, non distribuite, non sono riportate nella tabella.

## English

**Status: technical review ledger, not legal advice.** On 2026-09-24, SHA-256 was rechecked for all **21 MSYS2 source archives**, their **21 `PKGBUILD` files and 21 `.SRCINFO` files** against the [retained manifest](windows-sources-2026-09-23/source-materials.json), SHA-256 `81d63614445faf11dbaf0274d467dd1f970abf19f458b43ccda9d52e28e21fb4`. These are the local materials in `/tmp/webfence-windows-sources-054bb2d/`; no recipe was run. The 21 recipes match the **22 binary packages/43 DLLs** in the [Windows provenance record](windows-native-provenance-2026-09-23.md), with GCC shared by `libgcc` and `libstdc++`. The Git input plan, SHA-256 `1d6c259798e2e9ab68123a3d6dc248fb1a2cc42c9f308db78a27f4ff5d0bba8b`, was reconstructed from CI logs and is not the complete package inventory.

The table transcribes the `license` field in each `.SRCINFO` `pkgbase` section, except `libiconv`, where it shows the subsection for the **shipped binary package**. These fields are **MSYS2 declarations for sources/packages**, not licenses assigned by WebFence to each DLL or every embedded component. The Qt expression includes third-party material and does not prove each item is present in the Windows binary. Overrides in other, unshipped subsections are not listed.

| Source recipe / Ricetta | Binary packages / Pacchetti binari | `.SRCINFO` license metadata for shipped package(s) / Metadati per pacchetto distribuito |
| --- | ---: | --- |
| `brotli` | 1 | `spdx:MIT` |
| `bzip2` | 1 | `custom` |
| `double-conversion` | 1 | `spdx:BSD-3-Clause` |
| `freetype` | 1 | `spdx:GPL-2.0-or-later OR FTL` |
| `gcc` (`libgcc`, `libstdc++`) | 2 | `spdx:GPL-3.0-or-later WITH GCC-exception-3.1 AND GFDL-1.3-or-later` |
| `gettext` (`gettext-runtime`) | 1 | `spdx:GPL-3.0-or-later AND LGPL-2.1-or-later` |
| `glib2` | 1 | `spdx:LGPL-2.1-or-later` |
| `graphite2` | 1 | `spdx:LGPL-2.1-or-later` |
| `harfbuzz` | 1 | `spdx:MIT` |
| `icu` | 1 | `spdx:ICU` |
| `libb2` | 1 | `custom:CC0` |
| `libffi` | 1 | `spdx:MIT` |
| `libiconv` | 1 | `spdx:LGPL-2.1-or-later`; `documentation:spdx:GPL-3.0-or-later` (package override / campo specifico del pacchetto) |
| `libjpeg-turbo` | 1 | `custom:BSD-like` |
| `libpng` | 1 | `custom` |
| `md4c` | 1 | `spdx:MIT` |
| `pcre2` | 1 | `spdx:BSD-3-Clause` |
| `qt6-base` | 1 | `spdx:LGPL-3.0-only WITH Qt-GPL-exception-1.0 AND AFL-2.1 AND Apache-2.0 AND BSL-1.0 AND CC0-1.0 AND BSD-3-Clause AND CC-BY-4.0 AND GFDL-1.3-no-invariants-only AND GPL-2.0-only AND GPL-2.0-or-later AND GPL-3.0-only AND custom` |
| `winpthreads` | 1 | `spdx:MIT AND BSD-3-Clause-Clear` |
| `zlib` | 1 | `spdx:Zlib` |
| `zstd` | 1 | `spdx:BSD-3-Clause OR GPL-2.0-or-later` |

## Da verificare / Follow-up

- **IT:** Collegare i notices realmente inclusi nello ZIP a ciascuno dei 22 pacchetti e ai 43 file, poi esaminare codice incorporato, scelte per espressioni `OR`, voci `custom`, eccezioni e obblighi di distribuzione. La firma degli otto input PGP con `SKIP` non è stata verificata; gli hash dei payload da soli non autenticano la provenienza. Ricostruzione/sostituzione dei binari e revisione legale restano aperte. `distribution_ready=false` e `corresponding_sources_complete=false` non cambiano.
- **EN:** Link notices actually included in the ZIP to each of the 22 packages and 43 files; then review embedded code, choices in `OR` expressions, `custom` entries, exceptions and distribution duties. The eight PGP inputs marked `SKIP` have not had their signatures verified; payload hashes alone do not authenticate provenance. Binary rebuild/replacement and legal review remain open. `distribution_ready=false` and `corresponding_sources_complete=false` are unchanged.
