# M0 — Righe AX dopo il reset dei filtri / AX rows after filter reset

[Matrice IT](../it/M0-VALIDATION.md) · [Matrix EN](../en/M0-VALIDATION.md) · [Cocoa ADR IT](../it/ADR-006-QT-COCOA.md) · [Cocoa ADR EN](../en/ADR-006-QT-COCOA.md)

## Italiano

**Stato: difetto aperto in M0-01.** Il 2026-09-24, su macOS 26.6.2 Apple Silicon, è stata esaminata una **copia privata** del bundle storico `3598b5f` con la patch Cocoa corrente, RPATH esterni rimossi e firma ad hoc verificata. L'app normale è stata avviata con `HOME` temporanea e sole fixture sintetiche. Il client AX aveva già accesso (`AXIsProcessTrusted=true`); nessuna preferenza di sistema è stata modificata. Il bundle originale è rimasto firmato e l'hash della preferenza lingua personale è rimasto `d2bc81b3c6edc55e4b775d89ba679708610c7a417d45c696819ea8a73620ca82`.

La sequenza controllata dall'API macOS `AXUIElement` è stata ripetuta su due avvii con stati lasciati fermi almeno 0,35–1 s:

| Passo | Tabella AX | Prima riga |
| --- | --- | --- |
| Attivare «Carica 10.000 esempi» con `AXPress` | 10.000 righe, 4 colonne | `AXRow`, 4 `AXCell`, ID `DEMO-00001` |
| Impostare filtro `DEMO-10000` via `AXValue` | 1 riga, 4 colonne | `AXRow`, prima cella `DEMO-10000` |
| Impostare `no-such-fixture` | 0 righe, 4 colonne | Nessuna riga |
| Svuotare il filtro | 10.000 righe, 4 colonne | Lettura di `AXRole` della prima riga: **−25202**, `kAXErrorInvalidUIElement` |

Una nuova istanza del client AX ha confermato il riferimento invalido mentre l'app era ferma. Impostare il focus AX sulla tabella è riuscito ma non lo ha ripristinato. Il self-test interno `QAccessibleTableInterface` continua a passare dopo lo stesso filtro; pertanto quel test non dimostra la validità dei riferimenti pubblicati dal bridge del sistema. Un soak Cocoa da 30 s/10 cicli con letture AX durante il ciclo è terminato senza crash, ma le letture durante reset rapidi non sono una verifica atomica della riga.

Sono state compilate e firmate **solo copie private** del plugin Cocoa per isolare tre ipotesi: identità/hash distinti delle righe sintetiche; notifica `NSAccessibilityRowCountChangedNotification` al reset; restituzione diretta dell'array di righe sintetiche. Le prime due hanno superato il self-test interno ma **non** hanno corretto il riferimento AX invalido. La terza è stata provata mentre l'attivazione dell'app non era stabile: l'albero AX anomalo e i fallimenti di focus non possono essere attribuiti con certezza alla modifica. Nessuna variante è entrata nel repository o nel bundle originale. In prove successive la stessa sessione AX ha talvolta restituito `AXApplication` al posto di `AXWindow` per nuovi processi. Un controllo UI successivo ha segnalato che il Mac era bloccato; il momento esatto del blocco non è noto. Perciò le anomalie delle prove tardive sono inconcludenti. Il difetto delle prime due riproduzioni va ricontrollato a schermo sbloccato prima di decidere una correzione.

La prossima verifica richiede ripetere la sequenza reset 10.000 → 1 → 0 → 10.000 con Mac sbloccato e sessione AX pulita; se il difetto persiste, isolare la durata dei riferimenti nativi e correggerlo prima del collaudo con lettore reale. Non chiudere M0-01 sulla base dei soli contatori di righe, del self-test Qt o del soak.

Il commit `4635117` aggiunge `scripts/test-macos-ax-reset.swift`: avvia il bundle con HOME temporanea e verifica, a stato fermo, ruolo e quattro celle della prima riga nelle quattro fasi. La compilazione con Swift 6.4 locale è riuscita; il bundle non è stato eseguito nella sessione Mac bloccata. La CI macOS esegue il test come diagnostica non bloccante: l'uscita 77 significa che il client non ha il permesso Accessibilità e **non** costituisce un risultato positivo. Anche un esito AX positivo del runner non sostituirà VoiceOver su desktop reale.

Nella [run `4635117`](https://github.com/Matte2599/WebFence/actions/runs/35954825284/job/107490798295), lo step AX sul runner macOS 15 **ha eseguito la prova**, senza uscita 77: quattro esiti `PASS` per 10.000/1/0/10.000 righe; nella fase finale la prima riga espone `AXRow`, quattro celle e `DEMO-00001`. La run completa ha concluso con **sei job verdi**. Questo valida il bridge del bundle CI in quel contesto, non risolve la discrepanza con la prova locale: cambiano sia la versione macOS (15 contro 26.6.2) sia il bundle (commit corrente contro copia storica `3598b5f`). Il Mac locale risulta ancora bloccato; non si inferisce che il difetto sia corretto o specifico della versione OS.

## English

**Status: open M0-01 defect.** On 2026-09-24, the trial used a **private copy** of the historical `3598b5f` bundle with the current Cocoa patch, external RPATHs removed and ad hoc signature verified, on Apple Silicon macOS 26.6.2. The normal app ran with a temporary `HOME` and synthetic fixtures only. The AX client was already trusted (`AXIsProcessTrusted=true`); no system preference was changed. The original bundle retained a valid signature, and the personal language preference hash stayed `d2bc81b3c6edc55e4b775d89ba679708610c7a417d45c696819ea8a73620ca82`.

The sequence driven through macOS `AXUIElement` was repeated on two launches, leaving each state stable for at least 0.35–1 s:

| Step | AX table | First row |
| --- | --- | --- |
| Press “Load 10,000 examples” through `AXPress` | 10,000 rows, 4 columns | `AXRow`, 4 `AXCell` children, ID `DEMO-00001` |
| Set filter `DEMO-10000` through `AXValue` | 1 row, 4 columns | `AXRow`, first cell `DEMO-10000` |
| Set `no-such-fixture` | 0 rows, 4 columns | No row |
| Clear the filter | 10,000 rows, 4 columns | First-row `AXRole` read returns **−25202**, `kAXErrorInvalidUIElement` |

A fresh AX client process confirmed the invalid reference while the app was idle. Setting AX focus on the table succeeded but did not restore it. The internal `QAccessibleTableInterface` self-test still passes after the same filter sequence; therefore it does not prove that OS bridge references remain valid. A 30-second/10-cycle Cocoa soak with concurrent AX reads ended without a crash, but reads during rapid resets are not an atomic row check.

Three hypotheses were built and signed **only in private Cocoa plugin copies**: distinct identity/hash for synthesized rows; `NSAccessibilityRowCountChangedNotification` on model reset; and directly returning the synthesized row array. The first two passed the internal self-test but **did not** fix the invalid AX reference. The third was tested while app activation was unstable: the abnormal AX tree and focus failures cannot confidently be attributed to that modification. No variant entered the repository or original bundle. In later trials the same AX session sometimes returned `AXApplication` in place of `AXWindow` for newly started processes. A subsequent UI check reported that the Mac was locked; the exact time it locked is unknown. Those later anomalies are therefore inconclusive. Recheck the defect from the first two reproductions with the screen unlocked before deciding on a correction.

Next, repeat the 10,000 → 1 → 0 → 10,000 reset sequence with the Mac unlocked and a fresh AX session; if the defect persists, isolate and fix native-reference lifetime before testing with a real screen reader. Do not close M0-01 based only on row counts, the Qt self-test or the soak.

Commit `4635117` adds `scripts/test-macos-ax-reset.swift`: it launches the bundle with a temporary HOME and checks the first row's role and four cells at rest at each of the four stages. Compilation with local Swift 6.4 passed; the bundle was not run in the locked Mac session. macOS CI runs this as a nonblocking diagnostic: exit 77 means the client lacks Accessibility permission and is **not** a passing result. Even a passing AX result on the runner will not replace VoiceOver on a real desktop.

In [run `4635117`](https://github.com/Matte2599/WebFence/actions/runs/35954825284/job/107490798295), the AX step on the macOS 15 runner **did execute**, without exit 77: four `PASS` results for 10,000/1/0/10,000 rows; the final first row exposes `AXRow`, four cells and `DEMO-00001`. The complete run finished with **six passing jobs**. This validates the CI bundle bridge in that setting but does not resolve the discrepancy with the local trial: both macOS version (15 versus 26.6.2) and bundle (current commit versus historical `3598b5f` copy) differ. The local Mac is still locked; neither a fixed defect nor an OS-specific defect can be inferred.
