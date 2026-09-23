# M0 — Stabilità prolungata

[English](../en/M0-STABILITY.md) · [Matrice](M0-VALIDATION.md) · [Indice](../README.md)

## Procedura implementata

Il prototipo accetta `--soak-test=30m` (durata tra 10 secondi e un’ora). Usa il normale event loop Qt e un timer sul thread proprietario. Ogni ciclo attraversa dodici passi: carica 10.000 fixture, seleziona/legge/copia una prova, cambia lingua, filtra testo e severità, verifica risultati vuoti senza prove residue, ripristina e seleziona un’altra riga, nasconde dettagli e svuota. Le asserzioni controllano lo stato dopo ciascuna azione. Una fase ogni 250 ms consente a Qt di elaborare rendering ed eventi fra le azioni. Si conclude dopo un ciclo completo oltre la durata richiesta; chiusura anticipata significa prova incompleta.

Preferenze in directory temporanea rimossa all’uscita ordinaria; copia intercettata senza scrivere gli appunti. Nessuna rete o target esterno. Non cambiare preferenze OS per il test. I comandi del test sono diagnostica di sviluppo, non funzionalità per l’utente del prodotto.

Il log JSON riporta backend Qt, cicli, tempo monotono, heap/memoria gestita da Go, GC e dimensione della cache QVariant. **La memoria Go non comprende tutte le allocazioni native Qt.** Il raccoglitore esterno macOS/Linux misura RSS con `ps` ogni cinque secondi e conserva log grezzi, hash dell’eseguibile, metadati e sintesi. Richiede una nuova directory: non sovrascrive prove precedenti. Interrompe soltanto il processo avviato dal test dopo durata richiesta + 30 s oppure un intervallo di campionamento superiore a 30 s; uscita non zero o assenza dell’evento finale positivo non producono un esito superato.

```sh
# macOS: usare il bundle con Cocoa corretto, senza forzare QT_QPA_PLATFORM.
python3 scripts/test-desktop-soak.py \
  dist/WebFence.app/Contents/MacOS/webfence \
  /tmp/webfence-soak-macos-30m --seconds 1800

# Smoke test headless di sviluppo, distinto dal collaudo desktop.
QT_QPA_PLATFORM=offscreen python3 scripts/test-desktop-soak.py \
  bin/webfence /tmp/webfence-soak-offscreen-10s --seconds 10
```

Su Windows il comando interno è lo stesso; il raccoglitore RSS richiede macOS/Linux. La CI esegue solo una prova breve sui quattro target, anche dentro lo ZIP Windows con backend offscreen/nativo e PATH di sistema. Questo non dimostra 30 minuti su ogni sistema.

## Interpretazione

Conservare il commit e i flag della build oltre all’hash. Esaminare andamento RSS e heap, distinguendo startup/riscaldamento, oscillazioni GC e crescita persistente; la prima lettura RSS può precedere il caricamento delle librerie. Confrontare finestre temporali e cache dopo il riscaldamento. Un codice di uscita positivo prova completamento e asserzioni del carico, **non certifica assenza di perdite**. Non forzare GC ripetuti per nascondere la crescita. `summary.json` non classifica automaticamente la memoria come stabile.

La prova non invia input di tastiera OS né ascolta screen reader e non sostituisce il percorso assistivo, più monitor o sessioni sui sistemi minimi. Integra il gate M0-01 senza chiuderlo da sola.

## Evidenze attuali

2026-09-23: prova locale offscreen da 10 s completata in quattro cicli (circa 12 s), unit test dei limiti durata, vet desktop e self-test completo superati. Verificati rifiuto degli argomenti prima di inizializzare Qt, uscita fallita, assenza dell’evento finale e conservazione di directory esistente. Un processo sintetico bloccato che ignora SIGTERM è stato terminato dalla deadline esterna e dalla pulizia forzata; nessun esito positivo prodotto. [CI `8fd55b9` superata, sei job](https://github.com/Matte2599/WebFence/actions/runs/35879464771): smoke test sui quattro target e nello ZIP Windows (offscreen e backend nativo), oltre alle regressioni e al packaging Debian. Nessun esito di stabilità prolungata presunto.

## Sessione macOS avviata — esito ancora aperto

Avvio 2026-09-23 15:11:46 UTC, durata richiesta 1.800 s. Codice `8fd55b95207c2b9319e8b002c8de8be34f31c60f`, checkout pulito; Go 1.27.1, Qt 6.11.2 con Cocoa corretto, CGO C++ `-O2 -g -std=c++17`. Apple M5 Pro, 15 CPU logiche, 24 GiB RAM, macOS 26.6.2 (25G83), ARM64; nessun override di backend o scala.

Copia del bundle firmata ad hoc con solo nome/identificatore separati (`WebFence Stability`, `io.github.Matte2599.WebFence.Stability`) per distinguere la vecchia istanza aperta. SHA-256 dell’eseguibile della copia: `3a4e4747a8e2482b4d9c345cdc80b621a9e6a28f687b574960120775eef8668a`. Codice e plugin sono quelli della build indicata; la nuova firma modifica l’hash.

Output locale: `/tmp/webfence-soak-macos-8fd55b9-30m/`, con metadata, build-info, versione host/Qt, eventi, RSS e stderr. I primi 105 cicli in 315 s sono passati; cache 109 voci. Questi sono risultati intermedi, non una prova da 30 minuti. Verificare il processo e i log prima di proseguire: non ricominciare per un timeout di osservazione.

Screenshot e letture AX della nuova istanza confermano UI e prove visibili durante i cicli. Due letture AX della vecchia istanza hanno raggiunto il timeout; il campionamento del suo processo mostrava il thread principale in attesa di eventi. Causa non determinata: non è una prova di crash né una verifica di VoiceOver.
