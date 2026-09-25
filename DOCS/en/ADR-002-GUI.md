# ADR-002 — Native GUI comparison

[Italiano](../it/ADR-002-GUI.md) · [Index](../README.md)

Date: 2026-09-23. **Status: accepted by the author — “Andiamo con QT”.** Qt Widgets through MIQT becomes the main GUI; Go remains the core language. The technical choice does not close the M0 quality gates.

## Context and decision

The Fyne prototype did not pass the accessibility gate described in the [M0 report](M0-DESKTOP.md). The author requires a restrained, traditional desktop with a simple workflow and expert detail, and requested direct testing before choosing.

Adopt **Qt Widgets through MIQT for Go** as the desktop development foundation. Laboratory code moves into `internal/desktop`, launched by `cmd/webfence`; fixtures, catalogs and preferences remain separate. Fyne and the nested Qt module are removed from the current build and retained in Git history. The [development guide](DEVELOPMENT.md) contains current commands. Accessibility, release packaging and binding maintenance remain open. No scanner, WebView or Rust engine introduced.

## Alternatives and consequences

| Alternative | Advantages | Costs and limitations |
| --- | --- | --- |
| Qt Widgets + MIQT | Model-backed tables, desktop controls, plain-text editor and toolkit accessibility infrastructure; fits the requested UX direction | Go plus C++/CGO, dynamic libraries and packaging; MIQT is a community binding younger than Qt; separate Qt licensing obligations |
| Continue with Fyne | Existing code, Go-centered development, permissively licensed toolkit, previously verified build CI | Native accessibility inadequate in the current trial; requires toolkit work or waiting for improvements, with no estimated timeline |

WebFence's license remains unchanged and covers our code. MIQT is MIT. The proposal for Qt Core/Gui/Widgets is dynamic linking with LGPLv3 compliance: notices, Qt source, library replacement and required rights; dynamic linking alone is insufficient. Distribution requires a module/dependency inventory and verification of terms and any necessary exceptions in package conditions. If incompatible with the intended distribution, evaluate commercial Qt licensing. No purchase or legal review has occurred. Do not apply WebFence restrictions to LGPL/MIT components. See [Qt licensing](https://doc.qt.io/qt-6/licensing.html) and [LGPL obligations](https://www.qt.io/development/open-source-lgpl-obligations).

Qt 6.11 lists Windows 10 from 1809 and Windows 11 x86-64, macOS from 13, and Linux x86-64/ARM64 configurations. **Qt 6.12 is announced as the last version supporting Windows 10**: declaring product support needs an update and support-lifetime plan for that requirement. The upstream matrix does not certify WebFence or MIQT. [Qt platforms](https://doc.qt.io/qt-6/supported-platforms.html).

Later M1 decision: the author set [target minimums](M1-SUPPORT-POLICY.md) to macOS 26 ARM64, Windows 10 1809+ x86-64 and Ubuntu 24.04 LTS x86-64/ARM64. Future Windows 10 maintenance and real minimum-version trials remain open before support claims; this ADR retains the historical M0 gates below.

## Remaining M0 gates

1. Check reading and actions with VoiceOver, NVDA and Orca, beyond the accessibility tree alone.
2. Build and exercise packaging/runtime on the four requested architecture targets and set minimum OS versions; define Windows 10 maintenance.
3. Check selection/copy, focus, DPI, long text, stability and resource usage over extended sessions.
4. Verify distributable licenses/dependencies; test the bundle on a machine without development tools.
5. Keep domain code separate from widgets; revisit this ADR if verification requires an architecture change.

Tests and their limits are in the [practical comparison](GUI-COMPARISON.md). The Qt choice does not close M0; see [integration status](QT-DESKTOP.md).

Sources: [MIQT v0.14.0](https://github.com/mappu/miqt/tree/v0.14.0), [Qt accessibility](https://doc.qt.io/qt-6/accessible.html). Accessed 2026-09-23.
