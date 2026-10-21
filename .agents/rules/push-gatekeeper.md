# Push Gatekeeper Rule

This repository uses a strict validation gate for pushes.

**M3TAL Core** officially targets **Linux (Debian/Ubuntu x86_64/arm64), systemd services, daemon control-plane API, embedded Web GUI, and Docker**.

---

## Mandatory Behavior

- **Local Execution First**: Before any push, the Push Gatekeeper MUST execute all tests, static analysis, code formatting checks, binary builds, and packaging in the working tree. Never push assuming remote CI will catch or pass untested changes.
- **Complete Test & CI Suite**:
  1. `go mod verify`
  2. `if [ "$(gofmt -s -l . | grep -v vendor | wc -l)" -gt 0 ]; then exit 1; fi`
  3. `go test -v ./...`
  4. `(cd gui && npm ci && npm run build)` -> ensures `gui/dist/index.html` exists
  5. `go build -v -o /tmp/m3tal-test ./cli && go build -v -o /tmp/m3tal-api-test ./api/cmd && rm -f /tmp/m3tal-test /tmp/m3tal-api-test`
  6. `./scripts/build-deb.sh` -> packages `m3tal_${VERSION}_amd64.deb`
  7. `./scripts/test-pkg.sh` -> validates package installation & configuration preservation
- **Only the Push Gatekeeper is authorized to execute `git push`**.
- Validation results are tied to the exact working-tree/commit state that was tested.
- Any code, packaging script, UI, dependency, or workflow change after validation invalidates the previous result.
- **Active Diagnostics & Auto-Repair**: If any check fails (locally or remotely), do not simply halt and say "PUSH BLOCKED". Actively diagnose the error, repair it (or invoke repair agent), and retry validation up to 3 repair cycles.
- **Remote CI/Release Monitoring**: After pushing, quickly verify workflow kickoff via non-blocking check (`gh run list -L 3`). If remote jobs fail, inspect the failure log (`gh run view --log-failed`), apply the fix locally, re-run all local validation, and ensure the release is green.
- Never force-push unless explicitly authorized.
- Never commit secrets, credentials, GPG private keys, or generated build artifacts (`./m3tal`, `./m3tal-api`, `m3tal_*.deb`).

---

## Versioning & Distribution Protocol

- **Synchronized Versioning**:
  - Immediately before committing and pushing, execute:
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
- Every release commit is tagged with `v[semver]` (e.g. `v0.1.0`, `v0.1.1`).
