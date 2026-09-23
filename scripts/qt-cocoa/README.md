# Qt Cocoa correction / Correzione Qt Cocoa

**IT:** Questa patch riguarda codice Qt, non viene rilicenziata sotto WebFence Community License. Il file originale riporta Copyright (C) 2016 The Qt Company Ltd. e `LicenseRef-Qt-Commercial OR LGPL-3.0-only OR GPL-2.0-only OR GPL-3.0-only`. Qui si conserva l’opzione LGPL-3.0-only; testi LGPL e GPL richiamato sono inclusi. La correzione attuale è di Yuri Barreira, revisione upstream indicata sotto. Non è una revisione approvata o un rilascio ufficiale Qt. Vedere [ADR-006](../../DOCS/it/ADR-006-QT-COCOA.md).

**EN:** This patch concerns Qt code and is not relicensed under the WebFence Community License. The original file states Copyright (C) 2016 The Qt Company Ltd. and `LicenseRef-Qt-Commercial OR LGPL-3.0-only OR GPL-2.0-only OR GPL-3.0-only`. The LGPL-3.0-only option is retained here; LGPL and referenced GPL texts are included. Current correction by Yuri Barreira, upstream revision below. This is not an approved revision or an official Qt release. See [ADR-006](../../DOCS/en/ADR-006-QT-COCOA.md).

- Source / sorgente: [Qt 6.11.2](https://download.qt.io/official_releases/qt/6.11/6.11.2/submodules/qtbase-everywhere-src-6.11.2.tar.xz).
- SHA-256: `5b2e00eccaf5a4d8c14134ffa0ea8dfd0a35ae1ffc7f8d87fa4305a1ed23cf22`.
- [Gerrit 772484](https://codereview.qt-project.org/c/qt/qtbase/+/772484), revision `de050555112940ed643dfa7bd9ed16470adc5a09`, patch set 1, status NEW on 2026-09-23.
- Only production changes to the header/implementation are included; upstream native test changes are not copied. Replaces the earlier guards from Caleb Meadows’ proposal 765434. / Incluse soltanto le modifiche di produzione a header/implementazione, non i test nativi upstream. Sostituisce le precedenti guardie della proposta 765434 di Caleb Meadows.
- Rebuild / ricompilazione: `sh scripts/build-qt-cocoa.sh QT_PREFIX OUTPUT_DYLIB` on macOS ARM64, matching Qt 6.11.2, CMake, Ninja, MoltenVK/Vulkan headers and Apple developer tools. / su macOS ARM64 con Qt 6.11.2 corrispondente, CMake, Ninja, header MoltenVK/Vulkan e strumenti Apple.

Distribution still requires the complete dependency/source/notices inventory documented in M0 packaging. / La distribuzione richiede ancora l’inventario completo di dipendenze, sorgenti e notices previsto dal packaging M0.
