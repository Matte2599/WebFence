# ADR-021 — Sandbox Chromium nel laboratorio CDP Linux

[English](../en/ADR-021-CDP-CHROMIUM-SANDBOX.md) · [Laboratorio](M3-BROWSER-CDP-LAB.md) · [ADR-020](ADR-020-CDP-LANDLOCK-BOUNDARY.md) · [Stato M3](M3-VALIDATION.md)

Data: 2026-09-28. Stato: **adottato solo per la fixture sperimentale Linux**.

## Contesto e decisione

Il laboratorio CDP avviava Chromium con `--no-sandbox`; il solo container non dimostrava l'isolamento del renderer. In una prova diagnostica, la policy seccomp predefinita di Docker ha negato la creazione di user namespace. Tolta quella policy solo per diagnosticare, Landlock con `/proc` in sola lettura ha ancora impedito l'avvio della sandbox. La prova senza la policy Docker non è la configurazione adottata.

La fixture ora avvia Chromium senza `--no-sandbox` e usa il profilo seccomp esplicito `experiments/m3-cdp-browser/seccomp-profile.json`, derivato dal [profilo Playwright](https://github.com/microsoft/playwright/blob/2306f1bc1fad885946d915dc2a4d2df6827b851f/utils/docker/seccomp_profile.json) basato sulla allowlist Docker. Oltre ai permessi per `clone`, `setns` e `unshare` già aggiunti da Playwright, il profilo consente `landlock_create_ruleset`, `landlock_add_rule`, `landlock_restrict_self` e `chroot` necessari alla fixture con sandbox Chromium. La licenza Apache 2.0 del file di origine è inclusa in `experiments/m3-cdp-browser/LICENSE.playwright.txt`. Il container conserva utente non privilegiato, `--cap-drop ALL`, `no-new-privileges`, filesystem di base in sola lettura, quota 1 GiB/256 PID e le due prove con rete `none`/`bridge`. Il filtro di rete seccomp WebFence viene installato nell'helper e rimane ereditato dal browser.

Landlock concede in `/proc` solo lettura/esecuzione e `WRITE_FILE` su file esistenti, necessario alla configurazione degli user namespace; non concede creazione o rimozione. La directory privata resta scrivibile e il file sintetico esterno rimane negato. Dopo la fixture HTTP(S), il programma cerca un renderer discendente del processo Chromium e richiede che il renderer abbia `NoNewPrivs: 1`, seccomp in modalità filtro con più filtri del browser e uno user namespace diverso da quello del processo browser. Una build diagnostica con `--no-sandbox` ha fallito esattamente la verifica del namespace; la modifica diagnostica non è versionata.

## Limiti

La verifica riguarda almeno un renderer vivo della fixture, non tutti i processi Chromium: GPU e alcuni servizi possono avere confini differenti. `WRITE_FILE` su `/proc` è più ampio dei tre file di mappatura degli user namespace, perché i percorsi dei PID futuri non sono noti prima di applicare Landlock. Permessi kernel, utente non privilegiato e container restano quindi parte del confine. Il profilo seccomp concede le chiamate di gestione dei namespace richieste da Chromium; un host che le nega tramite altre policy farà fallire la prova. Il controllo non dimostra resistenza a evasione della sandbox, quote aggregate fuori dal container, contenimento su macOS/Windows o integrazione sicura nel desktop. Il primo criterio M3 resta aperto.

Riferimenti: [sandbox Linux di Chromium](https://chromium.googlesource.com/chromium/src/+/main/docs/linux_sandboxing.md), [restrizioni AppArmor sugli user namespace](https://chromium.googlesource.com/chromium/src/+/main/docs/security/apparmor-userns-restrictions.md), [documentazione Landlock del kernel](https://docs.kernel.org/userspace-api/landlock.html).
