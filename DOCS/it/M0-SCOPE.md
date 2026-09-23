# M0 — Primo controllo delle origini

[English](../en/M0-SCOPE.md) · [Indice](../README.md) · [Specifica del motore](SCANNING.md)

Data: 2026-09-23. Implementato `internal/scope`, indipendente da Qt e senza dipendenze aggiuntive. **È il primo livello di confronto URL, non un trasporto sicuro completo né una prova dell’autorizzazione del proprietario del target.** Il desktop resta offline; il laboratorio HTTP esiste soltanto nei test.

## Contratto attuale

`scope.New(origins)` crea una policy immutabile per letture concorrenti. Richiede almeno un’origine HTTP(S) esplicita; una configurazione parzialmente errata non produce una policy utilizzabile. Il valore zero nega tutto. `Check(rawURL)` restituisce un nuovo `url.URL` se l’origine è ammessa, altrimenti un errore stabile: `scope_invalid_url`, `scope_invalid_configuration` o `scope_origin_not_allowed`. Questi errori del pacchetto non includono input, query o credenziali; eventuali wrapper del chiamante devono conservare la redazione.

L’origine comprende protocollo, host e porta. Schema e hostname DNS sono confrontati in minuscolo; le porte predefinite diventano esplicitamente 80/443; IPv6 viene normalizzato con `net/netip`. HTTPS non autorizza HTTP, né una porta autorizza le altre. Nessun wildcard, sottodominio o alias DNS implicito. Ogni configurazione ammette solo origine e slash finale facoltativo: un percorso o una query vengono rifiutati, non eliminati ampliando il permesso.

Gli URL devono essere assoluti e lunghi al massimo 16 KiB. Sono rifiutati userinfo, frammenti (anche `#` vuoto), backslash, spazi/controlli non codificati, UTF-8 malformato, host Unicode, punto finale DNS, zone IPv6, IPv4 mappato in IPv6 e forme IPv4 alternative. IPv4 deve avere quattro ottetti decimali canonici; IPv6 deve essere tra parentesi quadre. Le porte esplicite devono essere decimali 1–65535 senza zeri iniziali. I nomi DNS usano etichette ASCII alfanumeriche con trattini interni; gli A-label IDNA già codificati possono essere dichiarati esplicitamente, senza conversione Unicode automatica.

I percorsi codificati e la query originale rimangono intatti, inclusi ordine, parametri ripetuti e segmenti `..`: non si confonde il confronto dell’origine con la normalizzazione delle risorse. Un link relativo va risolto dal chiamante contro la base appropriata e la destinazione assoluta risultante va nuovamente controllata, anche per ogni redirect. La policy non risolve DNS e non apre socket.

## Laboratorio e verifiche

`internal/scope/lab_test.go` crea due server `httptest` su IP loopback letterali e porte temporanee. Solo l’origine del primo è nella policy. Un adattatore HTTP **solo di test** verifica la prima richiesta e ogni redirect; un secondo blocco, indipendente dalla policy, permette connessioni esclusivamente ai due indirizzi del laboratorio. Proxy disabilitati, timeout client di 3 secondi, limite di tre risposte nella catena di redirect, lettura fixture limitata a 1 KiB; nessun DNS pubblico, segreto o target esterno.

La suite prova richieste ammesse, redirect relativo, accesso diretto e redirect a una porta esclusa, redirect verso `.invalid` e ciclo di redirect limitato. Il secondo server deve ricevere zero richieste e il blocco di connessione non deve ricevere tentativi esterni: i rifiuti devono avvenire prima. Le fixture non valutano vulnerabilità dei siti.

Le prove pure coprono origini distinte, input ambigui, configurazione che non può ampliare silenziosamente lo scope, conservazione di path/query, indipendenza dagli oggetti mutabili del chiamante, valore zero e letture concorrenti. Il fuzz test verifica che gli input accettati restino nell’origine dichiarata e stabili al roundtrip, senza rete.

Verifica locale: suite con race detector superata, copertura del pacchetto 98,6% delle istruzioni; fuzzing limitato a 20 secondi superato (696.840 esecuzioni riportate). Sono dati di questa prova, non garanzie di sicurezza o benchmark dello scanner. La CI esegue i test del pacchetto sui quattro runner e una sessione fuzz limitata su Linux x86-64.

**CI completata:** [run 35859069506](https://github.com/Matte2599/WebFence/actions/runs/35859069506), codice `ed71bec`: quattro job superati (macOS ARM64, Windows Server x86-64, Ubuntu x86-64/ARM64), inclusi i test scope con race detector; fuzz limitato su Linux x86-64, build/vet/self-test Qt e bundle macOS superati. Non equivale a verifica dei desktop Windows 10/11 o Debian con tecnologie assistive.

## Limiti e passo successivo

Questo controllo può riconoscere un’origine privata/loopback dichiarata, ma **non concede il permesso di collegarsi a quella rete**. Prima di attivare traffico nel prodotto occorrono policy IP/CIDR, DNS e indirizzo effettivo fissato alla connessione, verifica TLS, isolamento dei proxy, scadenza/autorizzazione, metodi e percorsi/esclusioni, budget condivisi, cancellazione e gestione delle risposte. Il laboratorio non prova DNS rebinding, SSRF completo, browser, WebSocket o reti private reali. Il suo adattatore non va riutilizzato come broker di produzione.

Questa è una parte del gate M0 «laboratorio sintetico e primi test scope/rete». I gate GUI, packaging, storage, firma e keychain restano aperti; non viene dichiarata completata M0 o iniziata una release di scansione.

Riferimenti: [net/url](https://pkg.go.dev/net/url), [net/netip](https://pkg.go.dev/net/netip), [client HTTP e redirect](https://pkg.go.dev/net/http#Client), [httptest](https://pkg.go.dev/net/http/httptest). La compatibilità dei parser del futuro browser richiederà prove dedicate.
