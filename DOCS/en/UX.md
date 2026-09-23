# Interface and usability

[Italiano](../it/UX.md) · [Index](../README.md)

## Confirmed direction

Author request dated 2026-09-23: a simple GUI, innovative yet traditional, familiar to users of Windows tools from 2015–2020. Understandable for people with little security knowledge and sufficiently detailed for professionals. This is an interaction direction, not a copy of another product's branding or interface.

The application remains native desktop on macOS and Linux too. Menus, shortcuts and dialogs should follow each system's conventions. Visual styling cannot compensate for inaccessible toolkit controls.

## Planned structure

- Traditional menus: File, Project, Analysis, View, Help. Actions with text and icons; avoid essential commands identifiable only by a symbol.
- Compact toolbar for the current task; at most one prominent primary action.
- Stable navigation: Projects, Analysis setup, Results, Fix validation, Reports. Introduce sections when functional.
- Central tables and a resizable detail pane. Search, filtering, sorting and multi-selection only when their behavior is defined.
- Status bar with actual progress, requests/budgets and coverage state. Never fabricate progress or derive “secure” from zero results.

Use readable typography, consistent spacing, subtle borders, restrained colors and a modest visual hierarchy. Prefer traditional controls to oversized cards, decorative effects, gradients or pervasive animation. Match the system's light/dark theme; severity always has a label, not just color. Adjustable density and adaptive layouts are later goals, not currently available capabilities.

## Progressive complexity

| Guided workflow | Expert detail |
| --- | --- |
| Identify the project and clarify authorization | Exact scope, exclusions, origins and identities |
| Suggest a profile with explained limits | Budgets, concurrency, timeouts and individual checks |
| Explain what was observed and what to do | Redacted requests/responses, rule IDs, confidence and provenance |
| Validate a fix with an understandable outcome | Retest prerequisites, comparability, coverage and regressions |
| Export a report with visible limitations | Signature details, manifest and key verification |

“Advanced” sections expose more detail without automatically changing scope, aggressiveness, AI or authorizations. Switching must preserve data, results and state. The guided workflow must not hide errors, skipped checks or uncertainty, or promise absence of vulnerabilities. Always distinguish “no issue observed” from “check not performed”.

Errors explain the problem, consequence and available action, with expandable technical details. Explain technical terms briefly in context. Confirm only actions that need it, such as permanent project-data deletion; avoid sequences of popups for ordinary work.

## Verification and state

Test mouse-free workflows, focus order, control names and states in screen readers, evidence text selection/copy, resizing and DPI, contrast and IT/EN text. Evaluate both beginners and professionals on concrete tasks; participant usability studies have not yet been conducted.

The M0 prototype implements a command bar, filters, table, evidence pane and language selection. It is a feasibility laboratory: full menus, wizards, advanced sections, scanning workspace and project management are **planned**. Current styling uses Fyne's system theme; it is not the final design. The [M0 report](M0-DESKTOP.md) records an open accessibility gate.

M0 update: [practical Qt/Fyne comparison](GUI-COMPARISON.md) and [proposed ADR-002](ADR-002-GUI.md). The Qt experiment is separate; the author’s choice and adoption gates remain open.
