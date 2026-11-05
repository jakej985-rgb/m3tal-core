# 🧪 Testing & Verification Guide

This guide documents the automated test suites and verification procedures for M3TAL Core.

---

## 🏃 Running Go Tests

```bash
# All packages
go test ./...

# With race detector
go test -race ./...

# Verbose output
go test -v ./...
```

### Package coverage

| Package | Covers |
|---|---|
| `api` | HTTP handlers, auth middleware, plugin/repository/settings handlers. |
| `core/state` | SQLite store, schema migration, API key CRUD/validation. |
| `core/system` | Telemetry collection, metrics aggregator, processes, specs, storage. |
| `core/events` | Event bus pub/sub and wildcards. |
| `pkg/plugins` | Plugin manifest parsing, JSON-RPC protocol, host supervisor lifecycle. |
| `pkg/version` | Version metadata. |

---

## 🖥️ Web GUI Build Verification

```bash
cd gui
npm ci
npm run build    # produces gui/dist, embedded via go:embed
```

---

## 📦 Debian Package (.deb) Verification

Packages are validated inside a minimal Debian container for `GOAMD64=v1` CPU baseline compatibility and zero-cgo stability:

```bash
# Build test package
PACKAGER=dpkg-deb DEB_TARGET=m3tal_test_amd64.deb ./scripts/build-deb.sh

# Run containerized package validation
./scripts/test-pkg.sh m3tal_test_amd64.deb
```

The test script verifies:

1. Package contents (`/usr/bin/m3tal-api`, GUI assets, systemd unit, desktop entry).
2. No `/usr/bin/m3tal` CLI binary is shipped.
3. The systemd unit has no `docker.service` dependency.
4. `m3tal-api` starts and serves `/api/v2/version` under an AVX-less CPU model.
5. Upgrades preserve an existing `/etc/m3tal/.env`.

---

## ✅ Release Checklist

1. `gofmt -l .` returns nothing.
2. `go test ./...` passes.
3. `cd gui && npm run build` succeeds.
4. `./scripts/build-deb.sh` produces a lint-clean `.deb`.
5. `./scripts/test-pkg.sh <deb>` passes.
