# ADR-021 — Chromium sandbox in the Linux CDP lab

[Italiano](../it/ADR-021-CDP-CHROMIUM-SANDBOX.md) · [Lab](M3-BROWSER-CDP-LAB.md) · [ADR-020](ADR-020-CDP-LANDLOCK-BOUNDARY.md) · [M3 status](M3-VALIDATION.md)

Date: 2026-09-28. Status: **adopted for the experimental Linux fixture only**.

## Context and decision

The CDP lab launched Chromium with `--no-sandbox`; the container alone did not demonstrate renderer isolation. In a diagnostic trial, Docker's default seccomp policy denied user namespace creation. Removing that policy for diagnosis still left Chromium unable to start its sandbox while Landlock made `/proc` read-only. The trial without Docker's policy is not the adopted configuration.

The fixture now launches Chromium without `--no-sandbox` and uses the explicit seccomp profile `experiments/m3-cdp-browser/seccomp-profile.json`, derived from the [Playwright profile](https://github.com/microsoft/playwright/blob/2306f1bc1fad885946d915dc2a4d2df6827b851f/utils/docker/seccomp_profile.json) based on Docker's allowlist. In addition to Playwright's permissions for `clone`, `setns` and `unshare`, the profile allows `landlock_create_ruleset`, `landlock_add_rule`, `landlock_restrict_self` and `chroot` needed by the fixture with Chromium's sandbox. The upstream Apache 2.0 license is included in `experiments/m3-cdp-browser/LICENSE.playwright.txt`. The container retains a non-root user, `--cap-drop ALL`, `no-new-privileges`, a read-only base filesystem, 1 GiB/256 PID quotas and both `none`/`bridge` network trials. WebFence's network seccomp filter is installed in the helper and inherited by the browser.

The first Linux AMD64 CI job observed `EPERM` from `posix_spawn` in the Crashpad handler: the copied Playwright profile did not mention `clone3`. The fixture now returns `ENOSYS` (`errnoRet: 38`) for `clone3`, as in the [Moby default policy](https://github.com/moby/moby/blob/master/vendor/github.com/moby/profiles/seccomp/default.json), letting glibc fall back to `clone` without allowing `clone3`.

Landlock grants only read/execute and `WRITE_FILE` on existing files under `/proc`, needed to configure user namespaces; it does not grant creation or removal. The private directory remains writable and the outside synthetic file remains denied. After the HTTP(S) fixture, the program finds a Chromium descendant renderer and requires `NoNewPrivs: 1`, seccomp filter mode with more filters than the browser process, and a distinct user namespace. A diagnostic build with `--no-sandbox` failed precisely the namespace check; the diagnostic change is absent from versioned code.

## Limits

The check covers at least one live renderer in the fixture, not every Chromium process: GPU and some services can have different boundaries. `WRITE_FILE` under `/proc` is broader than the three user namespace mapping files because future PID paths are unknown before Landlock is applied. Kernel permissions, the non-root user and the container therefore remain part of the boundary. The seccomp profile permits namespace management calls Chromium needs; a host denying them through another policy will fail the trial. This does not prove resistance to sandbox escape, aggregate quotas outside the container, macOS/Windows containment or safe desktop integration. The first M3 criterion remains open.

References: [Chromium Linux sandbox](https://chromium.googlesource.com/chromium/src/+/main/docs/linux_sandboxing.md), [AppArmor user namespace restrictions](https://chromium.googlesource.com/chromium/src/+/main/docs/security/apparmor-userns-restrictions.md), [kernel Landlock documentation](https://docs.kernel.org/userspace-api/landlock.html).
