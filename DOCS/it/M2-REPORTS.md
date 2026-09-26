# M2 — Report verificabili

[English](../en/M2-REPORTS.md) · [Roadmap](../ROADMAP.md) · [Contratto](REPORTING.md)

Stato: **terzo blocco core M2**. `internal/reporting` esporta una run SQLite conclusa in un nuovo bundle `.wfr` ZIP con JSON e HTML statico in italiano e inglese. Il report usa la revisione dell'autorizzazione associata alla run, il ledger redatto M1 e conteggi di copertura; non ricostruisce URL, header, body o controlli che non sono stati registrati. Le aree non esplorate rimangono esplicitamente sconosciute. Le correlazioni CVE e lo stato della cache si aggiungono solo quando passati come snapshot esplicito: l'export da CLI della sola run li indica come non disponibili.

Il manifest `webfence-manifest-v1` elenca i quattro artefatti con dimensione e SHA-256, è canonicalizzato secondo JCS e firmato con il profilo JWS Ed25519 ristretto esistente. I byte `manifest.json` coincidono con il payload firmato `manifest.jws`. Il bundle non contiene chiavi private; l'export non sovrascrive un file preesistente. Un export richiesto senza firma è marcato `unsigned` in entrambi i report e viene rifiutato dal verificatore come prova firmata.

## Uso tecnico locale

Compilare `go build -o bin/webfence-report ./cmd/webfence-report` e `go build -o bin/webfence-verify ./cmd/webfence-verify`. I percorsi devono essere assoluti. I comandi sotto sono esempi di sintassi: non descrivono una scansione eseguita.

```text
webfence-report keygen -trust /percorso/trust.json -name "Operatore"
webfence-report export -db /percorso/progetti.sqlite -run RUN_ID -out /percorso/report.wfr -trust /percorso/trust.json -kid KEY_ID
webfence-report public -trust /percorso/trust.json -kid KEY_ID -out /percorso/chiave-pubblica.json
webfence-report trust-import -trust /altro/trust.json -public /percorso/chiave-pubblica.json -fingerprint SHA256_CONFERMATO_SEPARATAMENTE
webfence-verify -bundle /percorso/report.wfr -trust /altro/trust.json
```

`rotate -trust … -kid …` disattiva la firma con la vecchia chiave e conserva la chiave pubblica per verifiche storiche; `revoke -trust … -kid …` fa fallire la verifica locale dei report relativi. `export -unsigned` è una scelta esplicita, per esempio se il portachiavi non è disponibile; la modalità firmata fallisce chiusa. La chiave privata vive nel portachiavi nativo, non nel file pubblico di trust. Importare un descrittore richiede un'impronta SHA-256 ottenuta tramite canale indipendente: il file ricevuto non basta per stabilire fiducia.

## Verifica e limiti

`webfence-verify` funziona offline con un registro di fiducia esterno al bundle. Controlla il profilo JWS, il manifest canonico, tutti i file attesi e gli hash; rifiuta duplicati, percorsi inattesi, file aggiunti, alterazioni, chiavi sconosciute o revocate. Restituisce identità e stato della chiave dal registro locale, inclusa la data del suo ultimo aggiornamento. Non consulta una fonte di revoca: l'attualità della fiducia e dell'orologio non è garantita. Una firma non certifica la correttezza della scansione, l'autorizzazione o la sicurezza del target.

Le prove sintetiche coprono export bilingue, escaping HTML, revisione storica, bundle non firmato, manomissioni di artefatto/manifest/JWS, file extra, chiave errata/non fidata/revocata, rotazione e portachiavi indisponibile. Non sono stati esportati report di clienti o contattati target esterni. L'integrazione desktop e l'interazione guidata per cache/matching/report sono il blocco successivo; l'alpha CLI non è un pacchetto di distribuzione supportato.
