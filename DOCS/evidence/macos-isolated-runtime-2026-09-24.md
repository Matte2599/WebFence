# M0 — Runtime macOS con ambiente ridotto / macOS runtime with a reduced environment

[Roadmap](../ROADMAP.md) · [Packaging IT](../it/M0-PACKAGING.md) · [Packaging EN](../en/M0-PACKAGING.md)

## Italiano

Il 2026-09-24 è stata usata **una copia privata** del bundle locale storico (`3598b5f` con modifiche dichiarate), non l'artefatto originale. La copia aveva sei `LC_RPATH` esterni rimossi e una nuova firma ad hoc verificata; il bundle originale firmato e la preferenza lingua personale non sono stati modificati. La prova locale usa macOS 26.6.2 e il comando `python3 scripts/test-macos-bundle-isolated.py /percorso/WebFence.app`.

Lo script lancia il self-test Cocoa con un ambiente figlio ridotto a `PATH=/usr/bin:/bin`, `QT_QPA_PLATFORM=cocoa` e `QT_DEBUG_PLUGINS=1`. Il test è passato; Qt ha registrato **due plugin caricati dal bundle**, Cocoa e macstyle. In un secondo processo, senza debug Qt e con `DYLD_PRINT_LIBRARIES=1`, la prova di durata è passata: **quattro cicli in 12,005 s**. Il tracciamento ha osservato **1.127 caricamenti di immagini**, di cui **24 nel bundle** e gli altri in percorsi di sistema Apple (compresi i Cryptex); nessuno da Homebrew o da altre posizioni esterne. Il controllo rifiuta un plugin o un'immagine osservata fuori dal bundle/sistema Apple e richiede la presenza di Cocoa in entrambe le prove. Fixture e output sono sintetici; nessun sito esterno è stato analizzato.

Una prova esplorativa che combinava `DYLD_PRINT_LIBRARIES` **con il self-test di focus** è fallita in cinque asserzioni di Tab. Senza quel tracciamento, il self-test con debug Qt è passato. Il tracciamento modifica i tempi: la causa precisa non è isolata e non si usa quella combinazione come certificazione del focus. La CI esegue le due prove distinte. Questa evidenza riguarda solo le immagini caricate nei flussi esercitati: non esegue tutti i 28 binari inventariati, non simula un Mac pulito e non risolve la revisione legale o il minimo macOS supportato.

## English

On 2026-09-24, the trial used **a private copy** of the historical local bundle (`3598b5f` with declared modifications), not the original artifact. Six external `LC_RPATH` entries had been removed from the copy and its new ad hoc signature verified; the original signed bundle and personal language preference were unchanged. The local trial ran on macOS 26.6.2 using `python3 scripts/test-macos-bundle-isolated.py /path/WebFence.app`.

The script runs the Cocoa self-test with a child environment reduced to `PATH=/usr/bin:/bin`, `QT_QPA_PLATFORM=cocoa`, and `QT_DEBUG_PLUGINS=1`. It passed; Qt reported **two plugins loaded from the bundle**, Cocoa and macstyle. A second process, with Qt debugging off and `DYLD_PRINT_LIBRARIES=1`, passed the soak: **four cycles in 12.005 s**. The trace observed **1,127 image loads**, **24 from the bundle** and the remainder from Apple system paths (including Cryptexes); none came from Homebrew or another external location. The check rejects an observed plugin or image outside the app/Apple system and requires Cocoa in both runs. Fixtures and output are synthetic; no external site was scanned.

An exploratory run combining `DYLD_PRINT_LIBRARIES` **with the focus self-test** failed five Tab assertions. Without that tracing, the self-test with Qt debugging passed. Tracing changes timing: the exact cause has not been isolated, and that combination is not used to certify focus. CI runs the two tests separately. This evidence covers only images loaded by the exercised flows: it does not execute all 28 inventoried binaries, simulate a clean Mac, or settle legal review or the supported macOS minimum.
