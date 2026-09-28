# ADR-020 — Landlock boundary in the Linux CDP lab

[Italiano](../it/ADR-020-CDP-LANDLOCK-BOUNDARY.md) · [Lab](M3-BROWSER-CDP-LAB.md) · [ADR-015](ADR-015-BROWSER-PLATFORM-BOUNDARY.md) · [M3 status](M3-VALIDATION.md)

Date: 2026-09-28. Status: **adopted for the experimental Linux fixture only**.

## Context and decision

The CDP lab container has a read-only base filesystem, but that does not stop Chromium from reading files outside its temporary profile. We install a Landlock policy on the thread that starts Chromium, after preparing the Unix connections and CDP pipes. The goroutine stays on that OS thread until Chromium exits; the child process inherits the restriction. The helper sets `no_new_privs` and requires Landlock ABI 3 or newer, needed to handle `REFER` and `TRUNCATE` too; the trial fails when support is unavailable.

The policy permits reading/execution under `/usr`, `/lib`, `/lib64`, `/etc`, `/proc` and `/dev` when present, and reading/writing/execution only inside the helper's private directory. The synthetic test file is in a separate temporary directory. The helper reads it before installing the policy, then requires `EACCES` for both reads and writes; it checks that a child can read a private file and cannot read the denied file. Chromium starts only after these checks pass. The trial remains inside the container with a read-only base filesystem, memory/PID quotas and the seccomp network filter.

A follow-up check observes Chromium through CDP: after the HTTP(S) fixture completes, it disables request interception only for two `file://` navigations to synthetic files. The file in the private profile is read and displayed; the outside file returns `net::ERR_ACCESS_DENIED`. Without the Landlock call, a diagnostic build fails that exact assertion. The check is separate from target traffic and uses no external URL.

## Limits

Landlock applies here to the thread starting Chromium and its descendants, not to every pre-existing helper thread. Descriptors opened before restriction, including Unix sockets to the broker and CDP pipes, remain usable. `/proc` and `/dev` must remain readable for this Chromium fixture; this policy does not protect every datum visible through those filesystems or other uncovered channels. Even navigation to two synthetic files does not prove that a hostile renderer cannot escape the browser. Chromium still uses `--no-sandbox`; the container is essential to the trial. This decision does not enable a desktop browser, establish a macOS/Windows policy or close the first M3 criterion.

Technical reference: [Linux kernel Landlock documentation](https://docs.kernel.org/userspace-api/landlock.html).
