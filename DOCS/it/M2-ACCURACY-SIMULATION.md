# M2 — Simulazione locale dell'accuratezza CVE

[English](../en/M2-ACCURACY-SIMULATION.md) · [Matching](M2-MATCHING.md) · [Validazione M2](M2-VALIDATION.md)

## Metodo

`TestSyntheticAccuracySimulation` usa **30 casi sintetici etichettati** senza rete, database CVE o sistemi esterni. Le etichette distinguono 20 casi in cui l'assunzione di prodotto/versione è nota (`affected` o `unaffected`) e 10 casi indeterminati che richiedono astensione. I casi coprono NVD/CVE, limiti inclusivi ed esclusivi, versioni esatte, `changes`, segnali deboli, versioni non comparabili, AND NVD, piattaforme, moduli, identità di pacchetto, qualificatori CPE e intervalli sovrapposti. Un test separato verifica che la parte CPE (`a`, `o`, `h`) sia dichiarata per NVD quando manca un CPE completo.

Una conclusione positiva è `applicable`/`verified`; una negativa è `not_applicable`; `candidate` e `unknown` sono astensioni. Il corpus ha esiti attesi espliciti e fallisce se una valutazione cambia. Riproduzione: `go test ./internal/intelligence -run 'TestSyntheticAccuracySimulation|TestNVDProductPartIsExplicit' -v`.

| Misura sul corpus | Esito |
| --- | ---: |
| Casi totali | 30 |
| Casi con verità affetto/non affetto | 20 |
| Conclusioni corrette positive / negative | 8 / 9 |
| Falsi positivi / falsi negativi conclusivi | 0 / 0 |
| Astensioni su casi con verità nota | 3 |
| Casi indeterminati correttamente lasciati `unknown` | 10 |
| Copertura conclusiva sui casi con verità nota | 17/20 (85%) |

## Correzioni derivate dalla revisione

- NVD richiede una parte CPE esplicita se il segnale contiene solo produttore/prodotto: l'identità testuale non dimostra se il prodotto sia applicazione, sistema operativo o hardware.
- CVE con vincoli di piattaforma, modulo, pacchetto o CPE non dimostrati rimane `unknown`.
- Voci di versione CVE sovrapposte vengono valutate tutte; esiti discordanti rimangono `unknown`. I cambi di stato fuori dall'intervallo o con stato non riconosciuto sono respinti.

Questa è una **simulazione di regressione su casi costruiti**, non una stima statistica dell'accuratezza su CVE reali. Non misura qualità dei feed, identità dei prodotti sul campo, prevalenza, recall di vulnerabilità reali o sicurezza di backport dichiarati dall'operatore. L'analisi usa le semantiche pubblicate dal [CVE Record Format](https://cveproject.github.io/cve-schema/schema/docs/) e dalla [documentazione NVD delle configurazioni](https://nvd.nist.gov/vuln/Vulnerability-Detail-Pages).
