# Motore AI e inferenza locale

[English](../en/AI.md) · [Indice](../README.md)

Stato: progettazione M5. Il motore tradizionale viene prima e funziona anche senza AI.

## Ruolo

L'AI può classificare evidenze, suggerire controlli contestuali, spiegare impatto e rimedi, correlare percorsi e assistere il triage. Le sue proposte sono ipotesi finché una regola, una prova ripetibile o una verifica umana non le supporta. Non dichiara autonomamente una vulnerabilità confermata, non decide i permessi e non chiude da sola un finding.

Il contesto contiene soltanto dati minimizzati del progetto corrente e riferimenti alle evidenze. Il testo del sito, i commenti, i report importati e i record recuperati restano input non fidati. Un prompt o il RAG non costituiscono un confine di sicurezza: vedi [OWASP Prompt Injection](https://genai.owasp.org/llmrisk/llm01-prompt-injection/).

## Modalità previste

| Modalità | Comportamento |
| --- | --- |
| Disabilitata | Predefinita; nessuna dipendenza da LLM e nessun costo AI |
| Provider remoto | Attivazione per progetto, credenziali proprie, modello/revisione dichiarati, destinazione e dati trasmessi visibili |
| Runtime locale | Processo esterno controllato, endpoint ristretto e nessun fallback cloud automatico |
| Modello integrabile | Pacchetto opzionale scelto e scaricato dall'operatore, con licenza, hash, dimensione e requisiti; non pesi nascosti nell'installer |

«Integrato» indica gestione dal prodotto, non un modello proprietario WebFence già addestrato. Nessun provider, peso o modello frontier è selezionato oggi. [llama.cpp](https://github.com/ggml-org/llama.cpp) è un candidato per inferenza locale, da verificare per piattaforme, formati e distribuzione. La licenza del runtime non concede automaticamente i diritti sui pesi.

## Contratto e controlli

Richiesta all'adattatore: task, modello, riferimenti di progetto/run, evidenze redatte, schema di risposta, timeout e budget. Risposta: stato, suggerimenti strutturati, citazioni a prove, consumo, revisione disponibile e limiti. Salvare il risultato effettivo senza promettere riproducibilità bit per bit.

Nessuna shell, rete arbitraria, accesso a chiavi di firma o database completo per il modello. Ogni controllo proposto passa da validazione dello schema, allowlist degli strumenti e policy deterministica del motore. Rifiutare URL fuori scope, citazioni inesistenti e output troppo grandi. Non eseguire codice generato. Il contenuto AI nel report viene renderizzato come testo sicuro.

Il consenso remoto specifica provider, regione quando disponibile, categorie di dati e policy di conservazione verificata al momento dell'integrazione. Redazione di token, cookie, credenziali, PII e dati di clienti prima dell'invio; se non è possibile, escludere l'evidenza. Telemetria e uso per training non sono attivati implicitamente. L'offline richiede pesi e snapshot già disponibili; le richieste ai target restano traffico di rete.

## Risorse e qualità

Memoria dei soli pesi: circa `parametri × bit / 8`, cui si aggiungono KV cache, runtime, buffer e GUI/browser. Un modello da 8 miliardi di parametri a 4 bit richiede circa 4 GB decimali per i soli pesi: non è un requisito totale né una promessa di velocità. Context window, concorrenza, GPU e quantizzazione cambiano costi e qualità; nessun minimo hardware è ancora validato.

Preflight con spazio disco, RAM/VRAM disponibili e capacità del backend. Budget finiti di token, richieste, tempo e costo; OOM o provider indisponibile lasciano completo il percorso tradizionale e marcano l'arricchimento AI come incompleto.

Valutare modelli sullo stesso corpus separato dallo sviluppo: utilità per finding, precisione delle citazioni, falsi positivi, resistenza a prompt injection, costo e latenza. LLM-as-judge può assistere, non sostituire etichette e revisione umana. Nessuna promessa di scoperta di zero-day.
