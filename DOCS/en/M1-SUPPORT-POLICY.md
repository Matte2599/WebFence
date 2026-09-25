# M1 — Target platforms and verification limits

[Italiano](../it/M1-SUPPORT-POLICY.md) · [M1 validation](M1-VALIDATION.md) · [Development](DEVELOPMENT.md)

Author decision of September 25, 2026 for the **M1 alpha**. “Target” denotes configurations WebFence intends to maintain; it does not mean every version has been tested, that the OS vendor still supports it, or that an end-user release is available.

| System | Architecture and target minimum | Current evidence and limit |
| --- | --- | --- |
| macOS | **26.0**, Apple Silicon ARM64 only | The bundle declares 26.0; local self-tests and trials ran on 26.6.2, with CI on macOS 26. No real 26.0 launch has been recorded. |
| Windows | **10 1809 or later**, x86-64; Windows 11 x86-64 too | This is the first Windows 10 version listed in the [Qt 6.11 matrix](https://doc.qt.io/qt-6/supported-platforms.html). CI builds, tests and starts the ZIP on a Windows runner; no trial on 10 1809 or the author's real PC is recorded. Windows 10 versions before 1809 are outside the target. |
| Linux | **Ubuntu 24.04 LTS**, x86-64 and ARM64 | The [Qt 6.11 matrix](https://doc.qt.io/qt-6/supported-platforms.html) includes both architectures; CI uses Ubuntu 24.04 and builds/tests the `.deb` packages in isolated runtimes. Other Debian distributions and derivatives remain a compatibility goal, with no M1 minimum or support claim. |

The author's “64-bit Windows 10” choice is made precise by the 1809 floor of the current Qt version; it is not a claim that every later build has been verified. [Microsoft](https://learn.microsoft.com/en-us/windows/release-health/release-information) says ordinary Windows 10 support ended on October 14, 2025; [Extended Security Updates](https://learn.microsoft.com/en-us/lifecycle/faq/extended-security-updates) do not restore the normal support lifecycle. Using a security tool on Windows 10 in 2026 calls for attention to the host OS's security updates. [Qt](https://doc.qt.io/qt-6/supported-platforms.html) also identifies Qt 6.12 as the last release supporting Windows 10; any Qt upgrade must revisit this target.

By author decision, **M1 closes on local and CI verification of code/packages**, without claiming that assistive, mixed-monitor or every real-workstation trial passed. Before promising a supported release or verified accessibility, test the actual artifacts and declared environments using the [historical trial plan](M1-PREREQUISITES.md). These checks require no scanning of external targets.
