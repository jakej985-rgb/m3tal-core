# M3TAL Core Debian Package CPU Compatibility and Release Pipeline Plan

**Status:** Proposed  
**Scope:** `jakej985-rgb/m3tal-core` and its existing publishing path to `jakej985-rgb/m3tal-apt-key`  
**Incident:** `m3tal` package version `0.20.3` installs, but `m3tal-api.service` crashes on an Intel Core i5-750 with `SIGILL` at `x_cgo_getstackbound`.

## 1. Goal

Ship a corrected **single `m3tal` Debian package** containing both `/usr/bin/m3tal` and `/usr/bin/m3tal-api`, plus its existing GUI assets, configuration, service units, and maintainer scripts. The package must start on the older x86-64 CPU used by the server and remain installable through the central APT repository.

Do not create a separate `m3tal-api` Debian package. Do not work around the crash by weakening APT signature or checksum validation.

## 2. Confirmed findings

- The server reports an Intel Core i5-750, which does not support AVX.
- `/usr/bin/m3tal-api` from package version `0.20.3` contains AVX-encoded instructions at `x_cgo_getstackbound`; the process exits with `SIGILL`.
- `scripts/build-deb.sh` invokes plain `go build` for both CLI and API, with no explicit CPU target or compatibility test.
- `go.mod` declares Go `1.25.0`, while `.github/workflows/ci-deb.yml` and `.github/workflows/pub-deb.yml` specify Go `1.23`. Align the workflow toolchain with the module requirement.
- `pub-deb.yml` builds one `.deb`, creates a GitHub Release, and dispatches the package to `m3tal-apt-key`.
- The package script has both an `nfpm` path and a `dpkg-deb` fallback; both must include equivalent package contents and metadata.
- Current `scripts/test-pkg.sh` verifies basic installation and `m3tal --help`, but doesn't prove the API starts successfully on an older CPU; the CI workflow does not currently run this package test.

## 3. Phase 1 — Reproduce and trace the unsupported instruction

1. Inspect Go version, build environment, compiler settings, and the exact source/build artifact that produced `m3tal-api`.
2. Determine whether AVX instructions are emitted by Go assembly/runtime, cgo-generated code, an external C compiler, or a dependency. Do not assume `GOAMD64=v1` alone controls all C/compiler output.
3. Record the failing instruction bytes and instruction address from the installed binary.
4. Reproduce the crash in a controlled environment that represents an older x86-64 CPU, using real compatible hardware or an emulator with AVX disabled.

**Exit criteria:** Root cause is documented and the team can reproduce the failure before the fix and demonstrate the corrected build no longer executes unsupported instructions during startup.

## 4. Phase 2 — Make the build reproducible and compatible

1. Align Go versions in `go.mod`, `ci-deb.yml`, `pub-deb.yml`, and any other active build workflows. Pin a supported toolchain compatible with the declared module version.
2. Set an explicit baseline for the Linux AMD64 build (initial target: Go `GOAMD64=v1`) for both `m3tal` and `m3tal-api`.
3. Inspect cgo use and compiler flags. If the problematic code is cgo-related, either configure the C compiler for the supported baseline or remove the unnecessary cgo dependency for this build only after validating behavior.
4. Keep the build settings centralized so local `build.sh`, `Makefile`, `scripts/build-deb.sh`, CI, and release workflows do not silently produce binaries with different CPU requirements.
5. Ensure build scripts fail immediately when compilation fails and never package stale binaries from an earlier build.

**Exit criteria:** Both binaries build from a clean checkout with consistent, documented toolchain and CPU settings. API startup works on an AVX-less x86-64 target.

## 5. Phase 3 — Keep the single Debian package consistent

1. Keep `Package: m3tal` and the current package layout; do not split API into a separate package.
2. Confirm both `nfpm.yaml` and the `dpkg-deb` fallback include the same required files: `/usr/bin/m3tal`, `/usr/bin/m3tal-api`, GUI assets, config templates, version files, icons, desktop entry, service unit(s), and maintainer scripts.
3. Compare generated package contents from both paths, or choose one canonical packaging path and make the fallback explicitly tested.
4. Validate Debian metadata, architecture, dependencies, service files, executable permissions, and post-install/remove behavior.
5. Preserve existing user configuration at `/etc/m3tal/.env` across upgrades. Do not overwrite user data or reset state databases.
6. Verify systemd service dependencies and restart behavior. Add sensible start-rate limiting so a deterministic startup failure does not generate an unlimited stream of core dumps.

**Exit criteria:** A clean package inspection confirms expected files, permissions, metadata, service definitions, and upgrade-safe configuration handling.

## 6. Phase 4 — Strengthen CI and package tests

1. Update `.github/workflows/ci-deb.yml` to use the aligned Go toolchain and explicit CPU baseline.
2. Add a clean build and `go test ./...`; retain formatting and dependency-integrity checks.
3. Build the complete `.deb` in CI and run `scripts/test-pkg.sh` (or a reliable replacement) instead of testing only standalone binaries.
4. In package tests, install the package in a clean Debian environment, inspect installed files, and start `m3tal-api` under a bounded test. Check service status and a health/API endpoint where one exists; stop the service after the test.
5. Add an older-CPU validation job or emulator test with AVX unavailable. Confirm `m3tal-api` launches and stays running for a reasonable test window. A normal modern GitHub runner is not sufficient by itself.
6. Add a regression check for the instruction/build configuration responsible for this incident.
7. Make any failed build, package test, compatibility test, or release validation block publishing.

**Exit criteria:** CI fails if the API binary cannot start on the baseline target or the Debian package is incomplete.

## 7. Phase 5 — Fix the release and APT publication sequence

1. Keep release artifacts versioned as a single `m3tal_<version>_amd64.deb`.
2. Ensure the release workflow runs the same validated build/package steps as CI; do not rebuild a different artifact after tests and publish that untested output.
3. Verify package filename, embedded Debian version, checksum, and release tag agree before creating the release and dispatching to `m3tal-apt-key`.
4. Publish the corrected package as a new version (proposed `0.20.4`, only if that version is unused and the repository versioning policy permits it).
5. Confirm the central APT repository accepted the new artifact and regenerated consistent index metadata and signatures.
6. Fetch the published `.deb` and compare its checksum with the exact tested release artifact.
7. Do not replace version `0.20.3` in place unless the repository's explicit policy supports replacement and the resulting APT metadata remains correct.

**Exit criteria:** The APT repository advertises the new version with valid signatures and hashes, and the downloaded package is byte-for-byte the artifact tested in CI.

## 8. Phase 6 — Upgrade and verify the affected server

Before upgrading, retain the current configuration and state. Keep the crashing API service stopped while preparing the fix.

After the corrected package is available, run on `m3tal-Server`:

```bash
sudo apt update
apt-cache policy m3tal
apt-cache show m3tal
sudo apt install --only-upgrade m3tal
sudo systemctl daemon-reload
sudo systemctl reset-failed m3tal-api.service
sudo systemctl start m3tal-api.service
systemctl --no-pager --full status m3tal-api.service
sudo journalctl -u m3tal-api.service -n 80 --no-pager
```

Then confirm the installed package version, test the API health endpoint, confirm the dashboard works, and verify the service remains active without fresh core dumps. If the service fails, stop it and capture the new core dump rather than repeatedly restarting it.

**Exit criteria:** The actual i5-750 server runs the upgraded API successfully, the GUI/API is reachable, and the service remains stable after restart/reboot.

## 9. Phase 7 — Documentation and release guardrails

1. Document the supported AMD64 CPU baseline and the reason for it.
2. Update build instructions so developers do not bypass the shared build settings.
3. Update release notes to identify the CPU compatibility fix.
4. Add a pre-release checklist: clean build, package inspection, package installation test, old-CPU startup test, checksum match, APT signature/index validation, and post-publish verification.
5. Review any other active release workflows or scripts that build/publish `m3tal` so they cannot bypass the corrected build path.

## 10. Definition of Done

- [ ] Root cause of the AVX instruction in the API startup path is documented.
- [ ] Go toolchain versions are aligned with `go.mod`.
- [ ] CLI and API build with the documented baseline; C/cgo compilation is checked separately where applicable.
- [ ] Corrected API starts on AVX-less x86-64 hardware or a faithful emulator.
- [ ] One complete `m3tal` Debian package contains both binaries and all expected assets.
- [ ] Package tests cover fresh install, upgrade/config preservation, and API startup.
- [ ] CI compatibility tests are required for release.
- [ ] Release publishes the exact tested `.deb` as a new version.
- [ ] Central APT metadata, signatures, and checksums validate.
- [ ] The affected server upgrades and `m3tal-api.service` stays active without `SIGILL`.
- [ ] Documentation and release notes are updated.

## 11. Implementation order

1. Reproduce the CPU instruction failure and establish the cause.
2. Fix and centralize build settings/toolchain.
3. Build and inspect the single `.deb`.
4. Add CPU compatibility and package-install tests.
5. Make the release workflow publish only the tested artifact.
6. Publish a new version through the central APT repo.
7. Upgrade and verify the affected server.
8. Document the supported CPU baseline and release guardrails.
