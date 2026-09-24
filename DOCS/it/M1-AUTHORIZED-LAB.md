# M1 — Snapshot di progetto nel laboratorio HTTP

[English](../en/M1-AUTHORIZED-LAB.md) · [Progetto e autorizzazione](M1-PROJECT-AUTHORIZATION.md) · [Trasporto M0](ADR-003-TRANSPORT.md) · [Roadmap](../ROADMAP.md)

`transport.NewAuthorizedLab(ctx, permit, grants, limits, resolver)` collega uno snapshot `project.RunScope` al broker HTTP **solo loopback**. È un incremento del primo punto M1, non un broker per reti o siti reali. La dichiarazione dell'operatore non prova la proprietà del target.

Il costruttore rifiuta snapshot assenti o scaduti. I grant di rete possono coprire più origini dello snapshot: sono una capacità di egress distinta, non ampliano l'autorizzazione del progetto. `Fetch` applica la policy del progetto a ogni URL iniziale e a ogni destinazione dopo un redirect, prima di prenotare budget, risolvere DNS o aprire un socket. Restano applicati anche i controlli del broker su grant IP, peer effettivo, TLS, timeout, redirect e budget. Una destinazione esclusa non consuma un nuovo tentativo; la richiesta che produce il redirect lo consuma.

La scadenza dello snapshot è anche una deadline del contesto di `Fetch`: interrompe attesa, DNS, dial o lettura in corso. L'errore esposto è un codice redatto di autorizzazione scaduta. `NewLab` resta disponibile per le prove M0 senza snapshot e continua ad accettare **solo indirizzi loopback esplicitamente concessi**; la GUI non invoca nessuno dei due costruttori.

Con una [run gestita](M1-MANAGED-RUNS.md), lo scope porta anche la revoca dello store. Il broker annulla le richieste in corso e restituisce `project_authorization_revoked`, controllando la revoca prima di ulteriori hop e socket. Lo scope non gestito usato nei test M0 e nel primo blocco M1 non riceve revoche dello store.

Test sintetici con due server `httptest` dimostrano che una richiesta ammessa passa, mentre il redirect verso una seconda origine presente nei grant di rete ma assente nel progetto non la contatta. Un blocco dial indipendente consente soltanto gli indirizzi dei due server del test. Un'altra prova mantiene una risposta in attesa fino alla scadenza e verifica annullamento, errore e conteggio dell'unico tentativo. Zero snapshot rifiutato. Suite `project`/`transport` con race detector superata localmente; la CI comprende entrambe sui target nativi.

**Limiti:** il controllo di progetto copre origini esatte, scadenza e revoca locale soltanto per run gestite. Mancano policy di metodi, percorsi, esclusioni, CIDR/reti reali, rate limit e integrazione di discovery/UI. Il broker esistente rimane deliberatamente loopback; questo blocco non autorizza scansioni esterne né dimostra protezione SSRF generale. I prossimi passi sono completare i limiti della policy e progettare un trasporto di produzione con verifiche per ogni richiesta.
