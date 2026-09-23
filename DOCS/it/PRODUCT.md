# Visione e requisiti

[English](../en/PRODUCT.md) · [Indice](../README.md)

Stato: specifica proposta, 2026-09-23.

## Problema e risultato atteso

Un'applicazione che funziona non è necessariamente sicura. WebFence intende rendere accessibile un ciclo ripetibile: definire il perimetro autorizzato, scoprire la superficie raggiungibile, raccogliere evidenze, prioritizzare i problemi, applicare le correzioni e verificarle. La diffusione dello sviluppo assistito da AI motiva il progetto; non è necessaria una classificazione del codice come «umano» o «AI» per usarlo.

Il prodotto è principalmente un DAST: osserva applicazioni in esecuzione. Non promette accesso a tutto il sorgente, alle dipendenze interne o a ogni percorso di business. Importazione SBOM e analisi del sorgente sono estensioni distinte. Vulnerabilità di logica e autorizzazione richiedono spesso più identità e contesto fornito dall'operatore.

## Utenti e percorso principale

Utenti iniziali: sviluppatori individuali e professionisti indipendenti, con uso gratuito anche per analisi retribuite per clienti. Le aziende devono ottenere autorizzazione scritta. Il prodotto sarà un'applicazione desktop nativa; la licenza e il perimetro autorizzato del target restano due requisiti distinti.

1. Crea un progetto e registra proprietario del target, autorizzazione, scadenza e ambienti ammessi.
2. Definisce origini, porte, percorsi, esclusioni, identità di test e budget.
3. Vede il riepilogo delle operazioni e avvia una scansione con profilo appropriato.
4. Consulta inventario, avanzamento, errori, prove e limiti di copertura.
5. Esporta il report firmato e conserva una baseline immutabile.
6. Dopo le correzioni avvia un retest e una verifica di regressione; confronta risultati e copertura.
7. Può esportare o cancellare progetto e dati derivati.

## Requisiti tracciabili

| ID | Requisito | Verifica minima | Fase |
| --- | --- | --- | --- |
| WF-01 | Perimetro e autorizzazione applicati a ogni richiesta | Redirect e DNS fuori scope bloccati | M1 |
| WF-02 | Profili di analisi con limiti indipendenti | Carico e cancellazione misurati su fixture | M1 |
| WF-03 | Controlli tradizionali con prove riproducibili | Casi vulnerabili e corrispondenti casi corretti | M1–M3 |
| WF-04 | Cache CVE aggiornata e consultabile offline | Sync interrotta non corrompe l'ultimo snapshot valido | M2 |
| WF-05 | AI remota o locale facoltativa | Stessa pipeline utilizzabile con AI disabilitata | M5 |
| WF-06 | Report firmati IT/EN | Verifica offline e rifiuto di un artefatto alterato | M2 |
| WF-07 | Memoria persistente e cancellazione | Progetto conservato al riavvio, dati derivati eliminabili | M1–M4 |
| WF-08 | Convalida fix e regressioni | «Non verificabile» per sessione scaduta o endpoint irraggiungibile | M4 |
| WF-09 | Interfaccia, CLI e report in due lingue | Parità di chiavi e risultati indipendenti dalla lingua | M1–M6 |
| WF-10 | Browser e autenticazione | Contenuti dinamici coperti senza uscire dal perimetro | M3 |

## Confini del primo prodotto

La prima alpha copre HTTP(S), crawling limitato e controlli a basso impatto in una singola installazione. L'MVP utile comprende anche report verificabili e confronto dopo i fix; non va dichiarato completo prima di M4. L'AI si aggiunge dopo una baseline tradizionale misurabile.

Non rientrano nell'MVP: WAF, protezione in tempo reale, agente EDR, sfruttamento distruttivo, correzioni automatiche in produzione, scansioni Internet indiscriminate, servizio SaaS multi-tenant o garanzie di conformità. Un retest è un'operazione esplicita; conservare il progetto non autorizza monitoraggio continuo.

## Decisioni ancora aperte

- Definire sistemi operativi supportati dopo prove reali, non per sola compilabilità.
- Validare accessibilità e distribuzione di Qt Widgets/MIQT, scelto per il desktop Go e selezionare browser driver, provider AI e modello locale tramite prototipi e valutazioni.
- Stabilire hardware minimo, corpus di benchmark, canale privato di sicurezza e termini commerciali.

Le priorità e le dipendenze sono nella [roadmap](../ROADMAP.md). Nessuna data di rilascio è impegnata.
