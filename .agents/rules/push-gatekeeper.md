# Push Gatekeeper Rule

This repository uses a strict validation gate for pushes to **`main`** and merging Pull Requests.

**M3TAL Core** officially targets **Linux (Debian/Ubuntu x86_64/arm64), systemd services, daemon control-plane API, and embedded Web GUI**.

---

## Operating Boundaries

- **Feature Work Separation**: Feature development and bug fixing take place in feature branches (`feat/*`, `fix/*`) using the `review-orchestrator` worker agent.
- **Main Branch Protection**: Only the **Push Gatekeeper** is authorized to push to `origin main` or merge PRs into `main`.
- **PR Testing Mode**: Before merging any PR targeting `main`, the Push Gatekeeper checks out the PR branch, confirms multi-agent review status is `PASSED`, and executes the full local CI baseline.
- **Pre-Push Hook Scope**: Pre-push hooks allow feature branch pushes freely for iterative review and testing, but strictly block direct pushes or merges to `main` unless all gates pass.

---

## Mandatory Behavior for Main & PR Gates

- **Local Execution First**: Before any merge or push to `main`, the Push Gatekeeper MUST execute all tests, static analysis, code formatting checks, binary builds, and packaging in the working tree. Never push assuming remote CI will catch or pass untested changes.
- **Complete Test & CI Suite**:
  1. `go mod verify`
  2. `if [ "$(gofmt -s -l . | grep -v vendor | wc -l)" -gt 0 ]; then exit 1; fi`
  3. `go test -v ./...`
  4. `(cd gui && npm ci && npm run build)` -> ensures `gui/dist/index.html` exists
  5. `go build -v -o /tmp/m3tal-test ./cli && go build -v -o /tmp/m3tal-api-test ./api/cmd && rm -f /tmp/m3tal-test /tmp/m3tal-api-test`
  6. `./scripts/build-deb.sh` -> packages `m3tal_${VERSION}_amd64.deb`
  7. `./scripts/test-pkg.sh` -> validates package installation & configuration preservation
- **PR Code Review Verification**: When evaluating a PR, confirm that the local Multi-Agent Review Engine (`scripts/review/engine.py --repo . --base origin/main --head HEAD --action verify-push`) reports `PASSED` with 0 blocking defects.
- **Active Diagnostics & Auto-Repair**: If any check fails, actively diagnose the error, repair it in the working tree, and retry validation up to 3 repair cycles.
- **Remote CI/Release Monitoring**: After pushing to `main`, quickly verify workflow kickoff via non-blocking check (`gh run list -L 3`). If remote jobs fail, inspect the failure log (`gh run view --log-failed`), apply the fix locally, re-run all local validation, and ensure the release is green.
- Never force-push unless explicitly authorized.
- Never commit secrets, credentials, GPG private keys, or generated build artifacts (`./m3tal`, `./m3tal-api`, `m3tal_*.deb`).

---

## Versioning & Distribution Protocol

- **Synchronized Versioning**:
  - Immediately before committing and pushing on `main`, execute:
    ```bash
    python3 scripts/generate_release_changelog.py --bump auto --apply
    ```
  - Automatically synchronizes:
    - `VERSION`
    - `gui/package.json`
    - `CHANGELOG.md`
- **M3tal-Hub Synchronization**:
  - Ensure the updated version and changelog are synced to M3tal-Hub:
    ```bash
    python3 scripts/sync_hub_manifest.py --manifest /home/m3tal/apps/M3tal-Hub/apps/manifest.yml --version $(cat VERSION)
    ```
- **Automated APT Keyring Ingestion**:
  - Remote CI workflow `.github/workflows/pub-deb.yml` dispatches `publish-deb` to `jakej985-rgb/m3tal-apt-key`.
  - Packages are placed in `pool/main/m3tal/`, cryptographically signed with the central 50-year key (`5F84FE50A40111C981410E11775AD1473BF25102`), and updated in `registry/packages.yml`.
- Every release commit on `main` is tagged with `v[semver]` (e.g. `v0.1.0`, `v0.1.1`).
