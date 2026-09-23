# ADR-006 — Correzione temporanea Qt Cocoa

[English](../en/ADR-006-QT-COCOA.md) · [Indice](../README.md)

Data: 2026-09-23. Stato: adottata per il bundle di sviluppo macOS con Qt 6.11.2; CI della nuova build completata sui quattro target. Non chiude il gate assistivo M0.

## Problema e decisione

Con il plugin Cocoa originale di Qt 6.11.2, la lettura dell’albero accessibile dopo selezione della riga, reset del filtro e cambio lingua ha riprodotto un SIGSEGV. Il PC della traccia ricade nel plugin Cocoa; il binario privo di simboli non permette di attribuire direttamente la funzione. La regressione upstream [b1ed5f6](https://github.com/qt/qtbase/commit/b1ed5f656f064e553b33752f8e87d2f5b9553e38) e la [correzione proposta 765434](https://codereview.qt-project.org/c/qt/qtbase/+/765434), di Caleb Meadows, descrivono una gestione errata della proprietà delle interfacce accessibili di tabella.

La prima mitigazione adottava le due guardie di produzione del patch set 1, revisione `c7fd3f34b997bb363be15650647665b3b6b8a5f4`: gli elementi sintetici gestiti dal genitore non devono cancellare l’interfaccia accessibile condivisa della tabella. La revisione upstream è **NEW**, non approvata né inclusa in una release verificata. Non disabilitiamo l’accessibilità. Il confronto locale prima/dopo sostiene questa diagnosi, senza dimostrare l’assenza di altri difetti Qt.

## Build e manutenzione

`scripts/build-qt-cocoa.sh` verifica la versione esatta 6.11.2, scarica l’archivio ufficiale e controlla SHA-256 prima di estrarlo. Applica le modifiche di produzione documentate qui e compila il plugin contro le corrispondenti librerie/private headers installate. Tutto avviene in una directory temporanea; Qt Homebrew rimane invariato. CMake e Ninja sono prerequisiti aggiuntivi. `WEBFENCE_QT_SOURCE_ARCHIVE` permette il riuso di un archivio locale, comunque verificato.

Il packaging sostituisce il plugin dopo `macdeployqt`, rende relativi i collegamenti QtCore/QtGui, rimuove il relativo RPATH Homebrew e firma nuovamente il bundle ad hoc. Un errore impedisce la pubblicazione e conserva il bundle precedente. Una versione Qt diversa richiede riesame esplicito: niente applicazione silenziosa a un’ABI privata diversa. `go run` e i binari non confezionati usano ancora il Qt installato e possono presentare il difetto originale.

[Sorgente, attribuzione, patch e testi di licenza](../../scripts/qt-cocoa/README.md) sono separati dalla licenza WebFence. Distribuire una libreria Qt modificata richiede completare anche sorgenti corrispondenti, notices e istruzioni di sostituzione; questo task non rende il pacchetto una release pronta. Riesaminare e rimuovere la patch quando una versione Qt ufficiale corretta supera lo stesso collaudo.

## Prove e limiti

- Build isolata del plugin e bundle completo locale riuscite; verifica `codesign --deep --strict` superata.
- Copie di prova con scala Qt 150% e 200%, stesso eseguibile Go e sostituzione del solo plugin: selezione `DEMO-10000`, apertura prove, cambio IT/EN e filtro `MO-10000` senza il crash precedente.
- Al 150% verificati anche filtro senza risultati, svuotamento e ricarica. Controlli e prove visibili; nessuna richiesta di rete.
- Al 200% comandi e prove raggiungibili, ma la tabella iniziale è troppo compressa verticalmente e richiede revisione del layout. Non dichiarare DPI completato.
- Alcune celle non compaiono stabilmente nella sintesi AX dopo selezione; serve il collaudo reale VoiceOver e non solo lettura dell’albero. Non sono stati eseguiti i nuovi test nativi upstream, né il ciclo misurato di 30 minuti, né test su più monitor.

Questa è una mitigazione verificata della sequenza di crash riprodotta, non una certificazione di accessibilità o stabilità generale.

Controlli aggiuntivi: archivio invalido rifiutato prima dell’estrazione; dipendenze del plugin limitate a framework nel bundle e librerie di sistema. Il bundle ricompilato dalla nuova procedura passa anche la sequenza GUI a scala standard. Self-test offscreen del binario non confezionato superato. Il bundle include solo Cocoa: tentare offscreen termina perché manca quel plugin; puntarlo ai plugin Homebrew carica due copie Qt e fallisce. Non usare questa combinazione come test del bundle: provarlo tramite la GUI nativa.

La prima CI della correzione (`9655816`, run 35872886041) ha superato Windows/Linux e i test macOS, ma la compilazione Cocoa richiedeva gli header MoltenVK assenti dal runner. Aggiunti MoltenVK e Vulkan headers ai prerequisiti espliciti. In seguito corretto anche il layout: margini interni ridotti, tabella con almeno due righe complete più spazio scrollbar, pannelli non collassabili. Prova visiva al 200% con due righe e ultima colonna raggiunta da tastiera; self-test con finestra logica 756×430 superato alle scale 1/1,5/2 e vet desktop superato. I valori Qt restituiti per copia hanno finalizer rimossi prima del rilascio esplicito; i test hanno intercettato e corretto una doppia liberazione durante lo sviluppo della modifica, prima del commit. CI successiva `953ed2d` superata sui quattro target ([run 35873709410](https://github.com/Matte2599/WebFence/actions/runs/35873709410)), inclusi nuovo self-test e bundle Cocoa.

## Sostituzione con la proposta 772484

Il collaudo di 30 minuti della prima mitigazione ha completato 600 cicli ma prodotto 17.730 avvisi AX; le celle rimanevano intermittenti. La [proposta 772484 di Yuri Barreira](https://codereview.qt-project.org/c/qt/qtbase/+/772484), patch set 1, revisione `de050555112940ed643dfa7bd9ed16470adc5a09`, elimina la cancellazione delle interfacce Qt da parte degli elementi nativi Cocoa. Le interfacce restano di proprietà della cache e della vista Qt. Questo spiega anche i riferimenti invalidati conservati dalle tabelle; è una diagnosi sostenuta dal confronto, non una certificazione generale.

La patch corrente sostituisce interamente le due guardie precedenti con le sole modifiche di produzione a `.h`/`.mm`. Stato upstream NEW verificato il 2026-09-23; nessuna approvazione o inclusione in una release presunta. La distinta proposta 772485 sui figli senza posizione non è applicata, non avendo riprodotto quel caso.

Copia isolata con lo stesso eseguibile e nuovo plugin: 20 cicli in 60,003 s, uscita zero, 13 campioni RSS e stderr vuoto dopo lettura AX tramite CUA durante il ciclo. Nella successiva prova con input OS le quattro celle di `DEMO-10000` restano disponibili dopo selezione, IT/EN, filtro vuoto e ripristino; la riga torna anche come elemento con focus. Preferenza lingua ripristinata a italiano. È una prova breve; non sostituisce VoiceOver, più monitor o una nuova sessione prolungata. I test upstream non sono stati eseguiti.

[Log sintetici e metadati della prova breve](../evidence/macos-ax-772484-2026-09-23/analysis.json).

La patch 772484 e la correzione di geometria sono verificate nella [CI `8d3053f`, sei job verdi](https://github.com/Matte2599/WebFence/actions/runs/35885936443), inclusi bundle Cocoa, self-test nativo e smoke di stabilità. Sessione prolungata aggiornata completata: 584 cicli in oltre 30 minuti, stderr vuoto; andamento RSS e limiti di cadenza in M0-STABILITY.

Ricompilazione e sostituzione del plugin Cocoa con una modifica diagnostica: [procedura e prove](M0-QT-REPLACEMENT.md). Materiali del bundle riutilizzati, stesso Go build ID, self-test e quattro cicli Cocoa superati localmente; originali preservati. È una prova del solo plugin, non dell’intera distribuzione.

Una [nuova prova AX a stato fermo](../evidence/macos-ax-table-reset-2026-09-24.md) ha isolato un limite non escluso dalla prova breve precedente: dopo 10.000 → 1 → 0 → 10.000 righe tramite filtro, l'array `AXRows` torna a 10.000 ma la prima riga restituisce `kAXErrorInvalidUIElement`. Il self-test Qt interno passa. Tre varianti private del plugin non hanno fornito una correzione verificata; la patch adottata resta invariata e il gate assistivo M0-01 rimane aperto.
