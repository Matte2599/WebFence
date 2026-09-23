# M0 — Prolonged stability

[Italiano](../it/M0-STABILITY.md) · [Matrix](M0-VALIDATION.md) · [Index](../README.md)

## Implemented procedure

The prototype accepts `--soak-test=30m` (10 seconds to one hour). It uses the normal Qt event loop and an owner-thread timer. Each cycle covers twelve steps: load 10,000 fixtures, select/read/copy evidence, change language, filter text and severity, check empty results without stale evidence, restore and select another row, hide details and clear. Assertions check state after every action. A phase every 250 ms lets Qt process rendering/events between actions. It ends after a whole cycle beyond the requested duration; early closure means an incomplete trial.

Preferences use a temporary directory removed on ordinary exit; copying is intercepted without writing the clipboard. No networking or external targets. Do not change OS preferences for the trial. Trial commands are development diagnostics, not product user features.

The JSON log records Qt backend, cycles, monotonic elapsed time, Go-managed heap/memory, GC and QVariant cache size. **Go memory excludes many native Qt allocations.** The external macOS/Linux collector measures RSS with `ps` every five seconds and retains raw logs, executable hash, metadata and summary. It requires a new directory and never overwrites previous trials. It stops only its own child after requested duration + 30 s or a sampling gap exceeding 30 s; nonzero exit or missing positive completion event cannot produce a passing outcome.

```sh
# macOS: use the corrected Cocoa bundle without forcing QT_QPA_PLATFORM.
python3 scripts/test-desktop-soak.py \
  dist/WebFence.app/Contents/MacOS/webfence \
  /tmp/webfence-soak-macos-30m --seconds 1800

# Development headless smoke test, separate from desktop trials.
QT_QPA_PLATFORM=offscreen python3 scripts/test-desktop-soak.py \
  bin/webfence /tmp/webfence-soak-offscreen-10s --seconds 10
```

Windows supports the same internal command; the RSS collector requires macOS/Linux. CI runs only a short trial on all four targets, including inside the Windows ZIP with offscreen/native backends and system PATH. This does not establish 30 minutes on every system.

## Interpretation

Retain the commit and build flags as well as the hash. Examine RSS/heap trends, distinguishing startup/warmup, GC oscillation and persistent growth; the first RSS reading may precede library loading. Compare time windows and cache after warmup. Successful exit proves workload completion and assertions, **not absence of leaks**. Do not force repeated GC to conceal growth. `summary.json` does not automatically classify memory as stable.

The trial sends no OS keyboard input and does not listen to screen readers; it does not replace assistive flows, multiple monitors or sessions on minimum systems. It contributes to M0-01 without closing that gate alone.

## Current evidence

2026-09-23: local 10 s offscreen trial completed four cycles (about 12 s), duration-bound unit tests, desktop vet and full self-test passed. Verified argument rejection before Qt initialization, failed exit, missing completion event and existing-directory preservation. A synthetic stalled child ignoring SIGTERM was stopped by the external deadline and forced cleanup; no passing outcome produced. [CI `8fd55b9` passed, six jobs](https://github.com/Matte2599/WebFence/actions/runs/35879464771): smoke trials on four targets and inside the Windows ZIP (offscreen and native backend), plus regressions and Debian packaging. No prolonged-stability outcome presumed.

## macOS session started — outcome still open

Started 2026-09-23 15:11:46 UTC, requested duration 1,800 s. Code `8fd55b95207c2b9319e8b002c8de8be34f31c60f`, clean checkout; Go 1.27.1, Qt 6.11.2 with corrected Cocoa, CGO C++ `-O2 -g -std=c++17`. Apple M5 Pro, 15 logical CPUs, 24 GiB RAM, macOS 26.6.2 (25G83), ARM64; no backend or scaling override.

Ad hoc signed bundle copy with only a separate name/identifier (`WebFence Stability`, `io.github.Matte2599.WebFence.Stability`) to distinguish the older open instance. Copy executable SHA-256: `3a4e4747a8e2482b4d9c345cdc80b621a9e6a28f687b574960120775eef8668a`. Code/plugin are from the stated build; re-signing changes the hash.

Local output: `/tmp/webfence-soak-macos-8fd55b9-30m/`, including metadata, build-info, host/Qt version, events, RSS and stderr. First 105 cycles in 315 s passed; cache 109 entries. These are intermediate results, not a 30-minute trial. Verify process/logs before continuing: do not restart because observation timed out.

Screenshot/AX reads of the new instance confirm UI and evidence visible during cycles. Two AX reads of the older instance timed out; sampling its process showed the main thread waiting for events. Cause undetermined: not evidence of a crash or a VoiceOver verification.
