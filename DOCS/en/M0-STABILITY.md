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

2026-09-23: local 10 s offscreen trial completed four cycles (about 12 s), duration-bound unit tests, desktop vet and full self-test passed. Verified argument rejection before Qt initialization, failed exit, missing completion event and existing-directory preservation. A synthetic stalled child ignoring SIGTERM was stopped by the external deadline and forced cleanup; no passing outcome produced. Long bundle trial and new CI still to run; no prolonged-stability outcome presumed.
