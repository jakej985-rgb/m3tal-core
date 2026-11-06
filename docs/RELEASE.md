# M3TAL Core Release Manual

This is the operator runbook for building, validating, publishing, and verifying a M3TAL Core Debian release.

## Scope and package identity

| Item | Value |
|---|---|
| Source repository | `jakej985-rgb/M3tal-core` |
| Debian package | `m3tal` |
| Binary | `/usr/bin/m3tal-api` |
| Architecture | `amd64` |
| Service | `m3tal-api.service` |
| Default port | `5050` |
| APT repository | `https://jakej985-rgb.github.io/m3tal-apt-key` |
| APT suite/component | `stable main` |

M3TAL Core is the dashboard, system monitor, settings/API-key manager, and plugin host supervisor. Container orchestration, Compose editing, mesh federation, VPN, proxying, storage sharing, AI, and benchmarking belong to external plugins.

## Release principles

- Never reuse a published version or tag.
- Never overwrite GitHub Release assets for an existing tag.
- Keep `VERSION`, `gui/package.json`, `CHANGELOG.md`, and `pkg/version/version.go` synchronized.
- Verify the exact `.deb` bytes before publication.
- Verify the signed APT repository advertises the same version and SHA-256.
- Document breaking API, plugin protocol, authentication, or package changes before release.

## Prerequisites

Local release workstation:

- Go version required by `go.mod`
- Node.js 20+
- `npm`
- `dpkg-deb` or `nfpm`
- Docker for `scripts/test-pkg.sh`
- Git with tags fetched

CI secrets used by `.github/workflows/pub-deb.yml`:

- `GH_PAT` — cross-repository checkout/push/dispatch token
- `GPG_PRIVATE_KEY` — optional plugin catalog signing key
- `GPG_PASSPHRASE` — optional GPG passphrase

The APT repository signing key is managed by `m3tal-apt-key`, not by this repository. Current expected fingerprint:

```text
5F84FE50A40111C981410E11775AD1473BF25102
```

## 1. Prepare the release

Review the working tree and confirm the intended changes are committed:

```bash
git status --short
git log --oneline --decorate -10
git fetch --tags
```

Check the current version:

```bash
cat VERSION
```

Inspect the proposed semantic bump without modifying files:

```bash
./scripts/bump-version.sh check
```

Apply the intended bump:

```bash
./scripts/bump-version.sh auto      # derive from conventional commits
./scripts/bump-version.sh patch     # force patch
./scripts/bump-version.sh minor     # force minor
./scripts/bump-version.sh major     # force major
./scripts/bump-version.sh set 0.25.0
```

The bump updates:

- `VERSION`
- `gui/package.json`
- `CHANGELOG.md`
- `pkg/version/version.go`

Review and commit the version/changelog changes before publishing.

## 2. Pre-flight validation

Run locally:

```bash
gofmt -l .
go mod verify
go test ./...
cd gui && npm ci && npm run build && cd ..
```

Expected result: no output from `gofmt -l`, tests pass, and `gui/dist` builds successfully.

Build the Debian package:

```bash
./scripts/build-deb.sh
```

`build-deb.sh` builds GUI assets, compiles `m3tal-api` with the Debian profile, and packages it as `m3tal_<version>_amd64.deb`.

Inspect package metadata:

```bash
dpkg-deb -I m3tal_<version>_amd64.deb
dpkg-deb -c m3tal_<version>_amd64.deb | head -80
```

Run package validation:

```bash
./scripts/test-pkg.sh m3tal_<version>_amd64.deb
```

This verifies:

- expected package paths
- no `/usr/bin/m3tal` CLI binary
- no Docker service dependency in the systemd unit
- API startup under an AVX-less QEMU CPU model
- `/etc/m3tal/.env` preservation during upgrade

## 3. CI gates

Pull requests and pushes run `.github/workflows/ci-deb.yml`, which validates:

- Go formatting
- Go tests
- `nfpm` and `dpkg-deb` package parity
- package content
- package installation/startup checks

Merging or pushing to `main` triggers `.github/workflows/pub-deb.yml`, which performs the release build and publication.

## 4. Publication pipeline

The publish workflow:

1. Reads `VERSION` and derives tag `v<version>`.
2. Refuses to rebuild if the tag already points to the current commit.
3. Auto-bumps and pushes a new version if the tag exists on another commit.
4. Builds GUI assets.
5. Runs `go mod verify`, `go test ./...`, and formatting checks.
6. Builds `m3tal_<version>_amd64.deb`.
7. Verifies package name, version, and architecture.
8. Runs `scripts/test-pkg.sh`.
9. Creates `m3tal_<version>_amd64.deb.sha256`.
10. Creates a GitHub Release with the `.deb` and checksum.
11. Dispatches `publish-deb` to `jakej985-rgb/m3tal-apt-key`.
12. Syncs `apps/manifest.yml` in `M3tal-Hub`.
13. Regenerates/signs `catalog.json` in `m3tal-plugins`.
14. Polls the public APT repository for up to 20 minutes.
15. Downloads `m3tal=<version>` through APT and verifies its SHA-256 equals the tested artifact.

APT dispatch payload:

```json
{
  "package_id": "m3tal",
  "repo": "jakej985-rgb/M3tal-core",
  "version": "<version>",
  "tag": "v<version>",
  "deb_url": "https://github.com/jakej985-rgb/M3tal-core/releases/download/v<version>/m3tal_<version>_amd64.deb",
  "sha256": "<sha256>",
  "allow_replace": false
}
```

## 5. Manual release verification

After publication, verify on a clean Debian/Ubuntu system:

```bash
curl -fsSL https://jakej985-rgb.github.io/m3tal-apt-key/install.sh | sudo bash
sudo apt update
apt-cache policy m3tal
sudo apt install -y m3tal=<version>
systemctl status m3tal-api
journalctl -u m3tal-api -f
curl -s http://localhost:5050/api/v2/version
```

Open:

```text
http://localhost:5050/gui/
```

Verify plugin management:

```bash
curl -H "X-API-Token: <key>" http://localhost:5050/api/v2/plugins
curl -H "X-API-Token: <key>" http://localhost:5050/api/v2/plugins/catalog
```

Install a plugin package when available:

```bash
sudo apt install m3tal-plugin-<name>
```

Then confirm Core discovers it:

```bash
curl -H "X-API-Token: <key>" http://localhost:5050/api/v2/plugins
```

## 6. APT repository verification

Verify the deployed repository:

```bash
cd /home/m3tal/apps/m3tal-apt-key
python3 scripts/validate_repository.py --strict --skip-container-test
python3 scripts/verify_deployed_repository.py \
  --base-url https://jakej985-rgb.github.io/m3tal-apt-key
```

The live repository currently publishes `stable main` for `amd64`. The authoritative key fingerprint is:

```text
5F84FE50A40111C981410E11775AD1473BF25102
```

## 7. Hub and catalog verification

Check that `M3tal-Hub/apps/manifest.yml` reports the new Core version and that generated Hub pages show the correct install command.

Check that `m3tal-plugins/catalog.json` reflects the current plugin catalog. Core release automation currently calls `scripts/generate_catalog.py`; review that script before release because it maintains a static catalog template and must be kept synchronized with `m3tal-plugins`.

## 8. Rollback

If a release is bad:

1. Do not delete or overwrite the GitHub Release.
2. Publish a corrective version with a higher semantic version.
3. If an older retained APT version is safe, users can pin it:

```bash
sudo apt install m3tal=<older-version>
sudo apt-mark hold m3tal
```

4. To return to normal updates later:

```bash
sudo apt-mark unhold m3tal
sudo apt update
sudo apt install --only-upgrade m3tal
```

Data under `/etc/m3tal`, `~/.config/m3tal`, and `/var/lib/m3tal` is intended to survive package removal; `apt purge` follows the package maintainer scripts and may remove package-owned configuration/state.

## 9. Release checklist

- [ ] Version bumped and synchronized.
- [ ] Changelog updated.
- [ ] `gofmt`, `go mod verify`, and `go test ./...` pass.
- [ ] GUI production build succeeds.
- [ ] `m3tal_<version>_amd64.deb` builds successfully.
- [ ] `scripts/test-pkg.sh` passes.
- [ ] GitHub Release contains `.deb` and `.sha256`.
- [ ] `publish-deb` dispatch succeeded.
- [ ] `m3tal-apt-key` repository validation passes.
- [ ] Live APT index advertises the version.
- [ ] Downloaded APT package SHA-256 matches the release checksum.
- [ ] `M3tal-Hub` manifest/page reflects the release.
- [ ] Plugin catalog remains accurate.
- [ ] Documentation references the correct package, service, port, keyring, and plugin boundaries.
