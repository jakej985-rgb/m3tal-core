# M3TAL Core Development Guide

This guide covers building, testing, developing the Web GUI, and understanding the repository layout.

## Prerequisites

- Go 1.25 or newer
- Node.js 20+ and npm
- Linux AMD64 for packaged builds
- Optional: Docker for `scripts/test-pkg.sh`
- Optional: `nfpm` for package builds; `dpkg-deb` is the fallback

Check versions:

```bash
go version
node --version
npm --version
```

## Repository layout

```text
M3tal-core/
├── api/                  # HTTP server, handlers, routes, WebSocket endpoint
├── core/
│   ├── events/           # event bus
│   ├── state/            # SQLite state and API keys
│   └── system/           # telemetry collectors and health
├── gui/                  # React/Vite embedded dashboard
├── pkg/
│   ├── models/           # shared API models
│   ├── plugins/          # plugin discovery, manifests, JSON-RPC host
│   ├── system/           # process/lock helpers
│   └── version/          # version metadata
├── scripts/              # build, package, release, catalog, review helpers
├── packaging/            # systemd units and package metadata
└── docs/                 # user and operator documentation
```

## Build

Install Go modules:

```bash
go mod download
go mod tidy
```

Build the embedded GUI and API binary:

```bash
make build
# or
./scripts/build-go.sh
```

Debian-profile build:

```bash
VERSION=$(cat VERSION) ./scripts/build-go.sh debian
```

The only runtime binary is `m3tal-api`. There is no `m3tal` CLI in the current codebase.

## Run locally

```bash
./m3tal-api
```

Defaults:

- API/GUI port: `5050`
- GUI: `http://localhost:5050/gui/`
- API docs: `docs/COMMAND_REFERENCE.md`

Use environment variables for local overrides:

```bash
API_PORT=5051 API_TOKEN=dev-token ./m3tal-api
```

## Web GUI development

```bash
cd gui
npm ci
npm run dev
```

The Vite development server proxies API requests to the local Core server. Build production assets with:

```bash
cd gui
npm ci
npm run build
```

`gui/dist` is embedded into the Go binary with `go:embed`.

## Testing

Run Go tests:

```bash
gofmt -l .
go test ./...
go test -race ./...
```

Build the GUI:

```bash
cd gui && npm ci && npm run build
```

Build and validate a Debian package:

```bash
./scripts/build-deb.sh
./scripts/test-pkg.sh <package>.deb
```

`test-pkg.sh` uses Docker/qemu to verify package contents, startup under a baseline CPU model, and upgrade preservation of `/etc/m3tal/.env`.

## Plugin development

Core plugin hosting lives in `pkg/plugins`. See `docs/PLUGIN_AUTHORING.md` for the manifest and JSON-RPC contract.

When changing plugin protocol behavior, update:

- `pkg/plugins/protocol.go`
- `pkg/plugins/host.go`
- `docs/PLUGIN_AUTHORING.md`
- `docs/COMMAND_REFERENCE.md`
- plugin repositories under `m3tal-plugins`

## Documentation workflow

Durable documentation lives in `docs/`. Keep these synchronized when behavior changes:

- API routes and auth: `docs/COMMAND_REFERENCE.md`
- environment variables: `docs/ENVIRONMENT_VARIABLES.md`
- operations: `docs/OPERATIONS.md`
- plugin contract: `docs/PLUGIN_AUTHORING.md`
- release process: `docs/RELEASE.md`

The `.github/workflows/docs-agent.yml` workflow mirrors selected docs into `m3tal-docs`; avoid putting generated or contradictory content in mirrored files.

## Coding conventions

- Keep Core focused on dashboard, telemetry, settings, API keys, and plugin supervision.
- Do not add container orchestration, Compose editing, VPN, proxy, or application-specific workload logic to Core.
- External workloads belong in `m3tal-plugins`.
- Use the standardized API response envelope for `/api/v2` routes.
- Prefer explicit scope checks for new API routes.
- Do not write secrets or tokens to logs.
