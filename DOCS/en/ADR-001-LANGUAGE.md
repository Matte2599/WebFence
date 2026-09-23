# ADR-001 — Go for the core and desktop

[Italiano](../it/ADR-001-LANGUAGE.md) · [Index](../README.md)

Date: 2026-09-23. Status: recommended initial decision; GUI toolkit subject to the M0 prototype. Author-confirmed requirement: native desktop application.

## Context

The expected main workload coordinates HTTP requests, crawling, rules, persistence, browsers and AI providers. There are no measurements demonstrating a CPU bottleneck. The project starts without code and must remain manageable by one author with occasional contributors. LLM weights and the browser may consume more resources than the coordinator in either language.

## Comparison

| Criterion | Go | Rust |
| --- | --- | --- |
| Network concurrency | Goroutines and standard library suit coordination | Async runtime and libraries require explicit composition |
| Memory | Automatic management with garbage collection; limits and profiling remain necessary | Ownership and control without GC; `unsafe` and FFI require review |
| Maintenance | Favors a simple core and contributor onboarding | Larger initial investment in ownership, async and integrations |
| Desktop | Fyne provides a compiled Go GUI, requiring product validation | Iced is a possible compiled Rust GUI, requiring equivalent validation |
| Distribution | Core can stay simple; GUI and browser add native dependencies | GUI and system libraries also require platform-specific packaging |
| Local inference | Separate process, without placing weights in the binary | Same separation is possible; rewriting the LLM runtime is unnecessary |

Maintenance assessments are engineering judgments, not benchmarks. Either language can produce a secure or insecure scanner: scope controls, authorization and limits do not follow from language choice.

## Decision

Use **Go** for the domain, scanning, persistence and desktop. Prototype **Fyne** before committing to the GUI: a native window with toolkit-rendered widgets, without requiring a browser-based interface. This does not mean widgets identical to each operating system's controls. Test assistive-technology accessibility, keyboard navigation, large tables, evidence text, DPI, text selection/coverage and keychain integration.

Avoid introducing both a Go core and a Rust core initially. Isolate browsers and LLM runtimes behind process contracts, timeouts and permissions. Do not promise a single static executable for the entire desktop product.

## Reconsideration criteria

Reopen this ADR if the GUI prototype misses essential requirements, the development team has a demonstrable operational advantage in Rust, or repeatable profiles reveal GC/CPU costs that cannot be addressed with limits, algorithms and better allocation patterns. Compare Go/Fyne and Rust/Iced on the same scenario; aesthetic preference alone does not justify rewriting the engine.

First M0 implementation: Go 1.27.1 and Fyne 2.8.1 pinned in `go.mod`/`go.sum`; CI reads the Go version from the module. Development build and bundle verified on macOS 26.6.2 ARM64 with Xcode 27. Accessibility does not pass the gate: incomplete tree and unverified assisted interaction. **Fyne remains a candidate; this limitation requires renewed evaluation before adopting it for the product.** Details and alternatives to compare are in the [M0 report](M0-DESKTOP.md).

Sources: [Go FAQ](https://go.dev/doc/faq), [Rust ownership](https://doc.rust-lang.org/nomicon/ownership.html), [Fyne](https://github.com/fyne-io/fyne), [Iced](https://github.com/iced-rs/iced). Checked 2026-09-23.

M0 update: [practical Qt/Fyne comparison](GUI-COMPARISON.md) and [proposed ADR-002](ADR-002-GUI.md). The Qt experiment is separate; the author’s choice and adoption gates remain open.
