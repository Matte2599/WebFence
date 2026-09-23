# Qt Cocoa correction / Correzione Qt Cocoa

**IT:** Questa patch riguarda codice Qt, non viene rilicenziata sotto WebFence Community License. Il file originale riporta Copyright (C) 2016 The Qt Company Ltd. e `LicenseRef-Qt-Commercial OR LGPL-3.0-only OR GPL-2.0-only OR GPL-3.0-only`. Qui si conserva l’opzione LGPL-3.0-only; testi LGPL e GPL richiamato sono inclusi. La correzione è di Caleb Meadows, revisione upstream indicata sotto. Non è una revisione approvata o un rilascio ufficiale Qt. Vedere [ADR-006](../../DOCS/it/ADR-006-QT-COCOA.md).

**EN:** This patch concerns Qt code and is not relicensed under the WebFence Community License. The original file states Copyright (C) 2016 The Qt Company Ltd. and `LicenseRef-Qt-Commercial OR LGPL-3.0-only OR GPL-2.0-only OR GPL-3.0-only`. The LGPL-3.0-only option is retained here; LGPL and referenced GPL texts are included. Correction by Caleb Meadows, upstream revision below. This is not an approved revision or an official Qt release. See [ADR-006](../../DOCS/en/ADR-006-QT-COCOA.md).

- Source / sorgente: [Qt 6.11.2](https://download.qt.io/official_releases/qt/6.11/6.11.2/submodules/qtbase-everywhere-src-6.11.2.tar.xz).
- SHA-256: `5b2e00eccaf5a4d8c14134ffa0ea8dfd0a35ae1ffc7f8d87fa4305a1ed23cf22`.
- [Gerrit 765434](https://codereview.qt-project.org/c/qt/qtbase/+/765434), revision `c7fd3f34b997bb363be15650647665b3b6b8a5f4`, patch set 1, status NEW on 2026-09-23.
- Only the two production guards are included; upstream native test changes are not copied. / Incluse soltanto le due guardie di produzione, non le modifiche ai test nativi upstream.
- Rebuild / ricompilazione: `sh scripts/build-qt-cocoa.sh QT_PREFIX OUTPUT_DYLIB` on macOS ARM64, matching Qt 6.11.2, CMake, Ninja, MoltenVK/Vulkan headers and Apple developer tools. / su macOS ARM64 con Qt 6.11.2 corrispondente, CMake, Ninja, header MoltenVK/Vulkan e strumenti Apple.

Distribution still requires the complete dependency/source/notices inventory documented in M0 packaging. / La distribuzione richiede ancora l’inventario completo di dipendenze, sorgenti e notices previsto dal packaging M0.
