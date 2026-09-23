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

## macOS session completed — 600 cycles

Started 2026-09-23 15:11:46 UTC, requested duration 1,800 s. Code `8fd55b95207c2b9319e8b002c8de8be34f31c60f`, clean checkout; Go 1.27.1, Qt 6.11.2 with corrected Cocoa, CGO C++ `-O2 -g -std=c++17`. Apple M5 Pro, 15 logical CPUs, 24 GiB RAM, macOS 26.6.2 (25G83), ARM64; no backend or scaling override.

Ad hoc signed bundle copy with only a separate name/identifier (`WebFence Stability`, `io.github.Matte2599.WebFence.Stability`) to distinguish the older open instance. Copy executable SHA-256: `3a4e4747a8e2482b4d9c345cdc80b621a9e6a28f687b574960120775eef8668a`. Code/plugin are from the stated build; re-signing changes the hash.

Local output: `/tmp/webfence-soak-macos-8fd55b9-30m/`, including metadata, build-info, host/Qt version, events, RSS and stderr. Completed 600 cycles in 1,800.004 s of workload (1,800.837 s externally), exit zero, 360 RSS samples, maximum interval 5.018 s and no INCOMPLETE marker. [Synthetic logs and analysis](../evidence/macos-soak-2026-09-23/analysis.json) retained in the repository; full stderr remains local with hash and normalized counts in the analysis.

Screenshot/AX reads of the new instance confirm UI and evidence visible during cycles. Two AX reads of the older instance timed out; sampling its process showed the main thread waiting for events. Cause undetermined: not evidence of a crash or a VoiceOver verification.

RSS collector also verified in a Debian ARM64 Linux container: code `9f87c49`, networking disabled during execution, offscreen backend, four cycles in about 12 s. This is a short collector check with toolchain present, not a real desktop or prolonged result. Local builds also ran during the macOS session: measurements are stability observations, not isolated performance benchmarks.

RSS medians in the 60–300, 300–600, 600–900, 900–1200, 1200–1500 and 1500–1800 s windows are respectively 122.46 / 114.84 / 113.88 / 108.80 / 107.64 / 107.80 MiB; sampled peak 166.78 MiB, last sample 107.58 MiB. The first 32 KiB reading precedes loading and is not a useful baseline. Final Go heap 4,290,272 bytes, final cache 109 entries. No sustained RSS growth observed in this window; this proves neither absence of leaks nor product hardware requirements.

The workload completed without crash or deadline expiry, but stderr contains **17,730 AX warnings**, concerning `QComboBoxListView` children and invalid Cocoa notifications. These are not ignored: the assistive gate remains open. This trial uses the stated commit, before the TabFocusAllControls correction, and does not validate that change for 30 minutes.

## Session with patch 772484 — completed

Started on 2026-09-23 at 16:04:37 UTC, requested duration 1,800 s, revision `8d3053fbe90c6101bb336493c149c9cc91f31cf0` with `vcs.modified=false`. Same host and flags as the earlier trial, new Cocoa patch 772484 and Tab/geometry corrections. Isolated `WebFence Stability Current` copy, identifier `io.github.Matte2599.WebFence.StabilityCurrent`, signed ad hoc. Executable SHA-256 `33ad7886d8195463b7a9fd9cbc5325f3107b562c0d9d1ed498b0865a038bb137`.

Logs in `/tmp/webfence-soak-macos-8d3053f-30m/`. CUA AX read during cycles to exercise the bridge; no input changing the workload. Completed 584 cycles in 1,802.340 s of workload (1,803.054 s externally), exit zero, 360 RSS samples, no INCOMPLETE marker and **empty stderr**. Processes terminated. Do not confuse this session with the earlier 600-cycle trial.

[New session logs and analysis](../evidence/macos-soak-8d3053f-2026-09-23/analysis.json). RSS medians in the 60–300, 300–600, 600–900, 900–1200, 1200–1500 and 1500–1803 s windows: 112.45 / 112.69 / 112.93 / 113.83 / 113.10 / 112.80 MiB. Sampled peak 168.27 MiB, last sample 112.89 MiB. Final Go heap 4,309,480 bytes; cache 265 entries from the 30 s sample onward. No sustained RSS growth or recurrence of AX warnings observed in this window; this does not prove general absence of leaks.

Timer intervals are not constant: the largest interval between samples five cycles apart is 38.426 s versus a nominal 15 s; maximum RSS collector gap 5.027 s. Cause undetermined. Other work continued on the same host, including archive collection; a lower cycle count alone does not establish a crash or freeze, but the trial does not certify latency or absence of UI stalls. Actual readers and multiple monitors still need trials. Desktop/Cocoa code is unchanged between `8d3053f` and `39c48ef`, which only adds build tools/materials and documentation.
