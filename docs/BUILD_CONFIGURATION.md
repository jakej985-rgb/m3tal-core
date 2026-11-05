# Build Configuration Guide

This guide explains how to build M3TAL Core from source.

## 📋 Build Requirements

* **Go 1.25+** (`go version` to verify)
* **Node.js + npm** (for the Web GUI assets)
* **Linux AMD64** target for packaged builds

```bash
go mod download
go mod tidy
```

## 🛠️ Build Process

### 1. Clone the repository

```bash
git clone https://github.com/jakej985-rgb/m3tal-core.git
cd m3tal-core
```

### 2. Build the Web GUI (embedded assets)

```bash
cd gui && npm ci && npm run build && cd ..
```

The output in `gui/dist` is embedded into the Go binary via `go:embed`.

### 3. Build the API binary

```bash
# Official script (native profile)
./scripts/build-go.sh

# Debian profile: linux/amd64, GOAMD64=v1 baseline, CGO disabled
VERSION=$(cat VERSION) ./scripts/build-go.sh debian

# Or manually
CGO_ENABLED=0 GOAMD64=v1 go build -trimpath -o m3tal-api ./api/cmd
```

The only binary produced is `m3tal-api` — there is no CLI.

### 4. Build the Debian package

```bash
./scripts/build-deb.sh                 # auto-detects nfpm or dpkg-deb
PACKAGER=dpkg-deb ./scripts/build-deb.sh
```

## 📊 Build Output

```bash
$ make build
🎨 Building Web GUI...
🚀 Building M3TAL Core API daemon...
✅ Build complete.
```

| Artifact | Purpose |
| :--- | :--- |
| `./m3tal-api` | Web GUI + REST/WebSocket API server (~12-14 MB) |
| `gui/dist/` | Embedded frontend assets |
| `m3tal_<version>_amd64.deb` | Installable Debian package |

## 🔍 Common Build Errors

| Error | Fix |
| :--- | :--- |
| `go: command not found` | Install Go 1.25+ and ensure it's on `PATH`. |
| `module not found` | Run `go mod download && go mod tidy`. |
| GUI build fails | Run `cd gui && npm ci` first; check Node version. |
| `permission denied` | `chmod +x scripts/*.sh build.sh`. |

## 🎯 Debian Baseline

Packaged builds use `GOAMD64=v1` and `CGO_ENABLED=0` for compatibility with older x86-64 CPUs (e.g., Intel Core i5-750). See [Debian CPU Baseline](DEBIAN_CPU_BASELINE.md).
