# Sviluppo, desktop e localizzazione

[English](../en/DEVELOPMENT.md) · [Indice](../README.md)

## Stato attuale e avvio

Il repository contiene soltanto documentazione e configurazione editor/Git. Non ci sono `go.mod`, build, installer, server, CLI o suite del motore. Per ottenere i documenti:

```sh
git clone https://github.com/Matte2599/WebFence.git
cd WebFence
```

Leggere [README](../../README.md), [roadmap](../ROADMAP.md) e [CONTRIBUTING](../../CONTRIBUTING.md). Nessun comando `webfence scan` è disponibile: non aggiungere istruzioni eseguibili finché il percorso non è implementato e verificato.

## Struttura futura proposta

```text
cmd/webfence/          applicazione desktop
cmd/webfence-cli/      eventuale CLI successiva
internal/app/         casi d'uso
internal/ui/          GUI e localizzazione
internal/policy/      autorizzazione e scope
internal/scan/        scheduler, discovery e regole
internal/intelligence/
internal/ai/
internal/report/
internal/storage/
testdata/             fixture sintetiche
DOCS/                 specifiche bilingui
```

Creare i pacchetti quando esiste una prima funzione utile, non tutti come contenitori vuoti. Il nucleo non importa il toolkit grafico; i controlli non chiamano direttamente rete e filesystem arbitrari.

## Sistemi e installazione

Obiettivo: macOS, Windows e Linux. La matrice di supporto dipende da prove su OS/architettura reali; il computer usato per la progettazione non dimostra compatibilità delle altre piattaforme. Go/Fyne richiede una toolchain grafica/native appropriata: vedi [documentazione Fyne](https://docs.fyne.io/started/). Non assumere `CGO_ENABLED=0` o semplice cross-compilation per l'intero desktop.

M0 deve verificare bundle macOS, pacchetto Windows e formato Linux scelto, dipendenze grafiche, keychain e accessibilità. Firma/notarizzazione dell'app e firma dei report sono sistemi separati. Non sono presenti certificati di distribuzione. Versioni di Go, GUI, browser e librerie saranno fissate dopo il prototipo, con lockfile e CI riproducibile.

L'app usa directory dati del sistema per utente, non la cartella del sorgente; esportazioni in destinazioni scelte dall'operatore. Nessun privilegio amministrativo ordinario. Modelli, browser e snapshot voluminosi sono pacchetti opzionali verificati; indicare download e spazio prima dell'installazione. Una disinstallazione non cancella silenziosamente i dati: offrire una scelta esplicita.

## IT/EN

Cataloghi di stringhe con chiavi stabili, italiano e inglese completi. Lingua iniziale dal sistema se supportata, altrimenti inglese; selettore persistente nel desktop. Lingua del report configurabile separatamente. No concatenazione di frasi tradotte; plurali e numeri/date formattati in presentazione. Persistenza sempre UTC, codici di stato e ID in forma canonica.

I testi delle regole includono titolo, impatto, spiegazione e rimedio in entrambe le lingue. Le prove originali non sono tradotte né alterate; eventuali traduzioni sono annotazioni separate. Il fallback inglese evita blocchi, ma una traduzione mancante nelle funzioni rilasciate è un difetto da correggere.

Ogni PR che cambia requisiti o funzioni aggiorna i documenti omologhi IT/EN. La lingua non deve cambiare fingerprint, severità, conteggi, scope o decisioni. Una traduzione di un report firmato è un nuovo artefatto da firmare.

## Rilascio e operatività

Prima di un rilascio: test dei rischi pertinenti, scansione dipendenze/segreti, SBOM, inventario licenze, checksum, firma dei pacchetti, note IT/EN e procedura di rollback. Prima di migrare dati creare un backup verificato; il rollback del binario non inverte automaticamente lo schema. Aggiornamenti di regole e modelli sono versionati separatamente e non cambiano una run già avviata.

Log locali redatti con ID di run, durata, limiti ed errori; niente telemetria predefinita. In caso di spazio insufficiente, feed scaduto, chiave bloccata, runtime assente o OOM, mostrare il componente interessato e ciò che resta utilizzabile. Non convertire una degradazione in successo silenzioso.
