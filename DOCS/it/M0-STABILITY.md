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

2026-09-23: prova locale offscreen da 10 s completata in quattro cicli (circa 12 s), unit test dei limiti durata, vet desktop e self-test completo superati. Verificati rifiuto degli argomenti prima di inizializzare Qt, uscita fallita, assenza dell’evento finale e conservazione di directory esistente. Un processo sintetico bloccato che ignora SIGTERM è stato terminato dalla deadline esterna e dalla pulizia forzata; nessun esito positivo prodotto. Prova lunga sul bundle e nuova CI ancora da eseguire; nessun esito di stabilità prolungata presunto.
