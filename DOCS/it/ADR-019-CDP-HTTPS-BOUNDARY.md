# ADR-019 — HTTPS mediato nel laboratorio CDP Linux

[English](../en/ADR-019-CDP-HTTPS-BOUNDARY.md) · [Laboratorio](M3-BROWSER-CDP-LAB.md) · [ADR-009](ADR-009-BROWSER-HTTP-BOUNDARY.md) · [ADR-017](ADR-017-CDP-BROKER-BOUNDARY.md) · [Stato M3](M3-VALIDATION.md)

Data: 2026-09-28. Stato: **adottato per la sola fixture sperimentale Linux**.

## Contesto e decisione

Il proxy iniziale nega `CONNECT`: un tunnel opaco non permetterebbe al gate e al broker di verificare ciascun percorso, metodo e redirect. Il laboratorio CDP già intercetta le richieste nel processo Chromium e usa socket Unix connessi dal genitore. Estendiamo quel percorso senza introdurre una CA MITM nel browser: l'helper invia una richiesta HTTP in forma assoluta con URL `https://` su uno dei socket ereditati; **solo il proxy su socket Unix** ammette tale forma. Il listener TCP continua a negarla e ogni forma di `CONNECT` rimane vietata.

Il gate controlla URL, tipo e budget; il broker crea la connessione TLS verso l'IP fissato, verifica certificato e hostname e ricontrolla policy e redirect. L'helper non apre socket di rete verso il target, non sceglie la fiducia TLS e non inoltra cookie o header del renderer. Per la fixture loopback, `NewAuthorizedLabWithPolicyTLSRoots` riceve una copia delle sole radici effimere del server sintetico; il costruttore pubblico mantiene la fiducia di sistema. Il renderer vede l'origine `https://` tramite CDP, senza terminare TLS al suo interno.

## Verifica e limiti

Il laboratorio esegue varianti HTTP e HTTPS con documento, script e `fetch` sotto una run ciascuna; in HTTPS controlla anche `isSecureContext`. Un'immagine fuori scope e un redirect esterno sono fermati; il canary TCP diretto riceve `EPERM` sia con rete container assente sia con rete `bridge`. Un test separato ammette il certificato della fixture, respinge hostname errato e origine esterna senza contatto target, e verifica che il listener TCP rifiuti HTTPS assoluto. Le prove non usano target esterni.

Questo è un confine sperimentale Linux/container, con Chromium `--no-sandbox` e certificato di fixture. Non abilita HTTPS nel browser desktop, non prova pagine ostili, sessioni, quote fuori dal container o i runtime macOS/Windows. La prima voce M3 resta aperta secondo [ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md).
