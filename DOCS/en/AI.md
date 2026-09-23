# AI engine and local inference

[Italiano](../it/AI.md) · [Index](../README.md)

Status: M5 design. The traditional engine comes first and works without AI.

## Role

AI may classify evidence, suggest contextual checks, explain impact and remediation, correlate workflows and assist triage. Its proposals remain hypotheses until supported by a rule, repeatable evidence or human verification. It does not independently declare a confirmed vulnerability, decide permissions or close a finding.

Context contains only minimized data from the current project and evidence references. Site text, comments, imported reports and retrieved records remain untrusted inputs. A prompt or RAG is not a security boundary: see [OWASP Prompt Injection](https://genai.owasp.org/llmrisk/llm01-prompt-injection/).

## Planned modes

| Mode | Behavior |
| --- | --- |
| Disabled | Default; no LLM dependency or AI cost |
| Remote provider | Enabled per project, operator credentials, declared model/revision, visible destination and transmitted data |
| Local runtime | Controlled external process, restricted endpoint and no automatic cloud fallback |
| Integrable model | Optional package selected and downloaded by the operator, with license, hash, size and requirements; no hidden installer weights |

“Integrated” means managed by the product, not an already-trained proprietary WebFence model. No provider, weights or frontier model is selected today. [llama.cpp](https://github.com/ggml-org/llama.cpp) is a local-inference candidate requiring platform, format and distribution review. The runtime license does not automatically grant rights to weights.

## Contract and controls

Adapter request: task, model, project/run references, redacted evidence, response schema, timeout and budget. Response: status, structured suggestions, evidence citations, usage, available revision and limitations. Store the actual output without promising bit-for-bit reproducibility.

No shell, arbitrary networking, signing-key access or full database access for the model. Every proposed check passes schema validation, tool allowlists and deterministic engine policy. Reject out-of-scope URLs, nonexistent citations and oversized output. Do not execute generated code. Render AI report content as safe text.

Remote consent specifies provider, region where available, data categories and retention policy verified when integrating. Redact tokens, cookies, credentials, PII and client data before transmission; exclude evidence when that is not possible. Telemetry and training use are not implicitly enabled. Offline operation needs previously available weights and snapshots; target requests still use the network.

## Resources and quality

Weight memory alone is approximately `parameters × bits / 8`, plus KV cache, runtime, buffers and GUI/browser. An 8-billion-parameter model at 4 bits needs approximately 4 decimal GB for weights alone: this is not a total requirement or speed promise. Context window, concurrency, GPU and quantization change costs and quality; no minimum hardware is validated yet.

Preflight checks disk space, available RAM/VRAM and backend capabilities. Enforce finite token, request, time and cost budgets; OOM or provider unavailability preserves the traditional path and marks AI enrichment incomplete.

Evaluate models on the same corpus held apart from development: finding usefulness, citation accuracy, false positives, prompt-injection resistance, cost and latency. LLM-as-judge may assist but cannot replace labels and human review. No zero-day discovery promise.
