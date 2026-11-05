# ⚡ M3TAL Core — Production Architecture Vision (v0.24+)

| [🚀 Overview](../README.md) | [⚙️ Environment](ENVIRONMENT_VARIABLES.md) | [🛠️ Build](BUILD_CONFIGURATION.md) | [🌐 Networking](NETWORKING.md) | [🐳 Docker](DOCKER.md) |
| :---: | :---: | :---: | :---: | :---: |

## 🎯 Architecture Vision & Guiding Principles

M3TAL Core (`m3tal-core`) is a **high-performance, Go-native Dashboard and System Monitor with extra plugins** for Linux desktops and servers.

Distributed strictly as a native Debian package (`.deb`), M3TAL Core runs directly on bare metal as the `m3tal-api` systemd daemon (`/usr/bin/m3tal-api`). It serves as the single source of truth for host hardware telemetry, system metrics, and the host supervisor for modular external plugins.

> **Ecosystem Responsibility Boundary**:  
> M3TAL Core contains zero container management, compose orchestration, proxy routing, or application database logic. All external workloads—including Docker container fleet monitoring (`m3tal-docker`), Compose editing (`m3tal-compose`), reverse proxying (`m3tal-traefik`), VPN tunnels (`m3tal-vpn`), local LLM inference (`m3tal-ai`), and multi-node clustering (`m3tal-mesh`)—are maintained as **extra plugins** in [m3tal-plugins](https://github.com/jakej985-rgb/m3tal-plugins) or showcased as application templates in [M3tal-Hub](https://github.com/jakej985-rgb/M3tal-Hub).

---

## 🏛️ Foundational Architectural Locks

### 1. Zero Container Footprint for Core
* **Native Host Execution**: M3TAL Core has **no Docker image**. It runs directly on the host operating system with minimal memory overhead, zero container layer virtualization lag, and direct access to Linux hardware metrics (NVIDIA GPUs, system thermals, storage mounts, top processes).
* **Embedded Web GUI**: The React/Vite web control center is compiled into static production assets and embedded directly into the Go executable via `go:embed`. There is no separate web container, Flask runtime, or Node.js server.
* **Canonical API Port**: The embedded Web GUI and REST API daemon serve concurrently on port `5050` (`API_PORT=5050`).

### 2. External Plugin Host Supervisor
* **Zero Core Footprint**: `m3tal-core` contains zero plugin-specific UI components, API models, or bespoke database schemas. It acts solely as the Host Supervisor and Dynamic UI mount point (`/plugins/<name>/`).
* **JSON-RPC 2.0 over Stdio**: External plugins run as independent subprocesses supervised by Core, communicating via standardized JSON-RPC 2.0 over stdin/stdout pipes.
* **SIGHUP Rediscovery via Dpkg Triggers**: When new plugin packages are installed via `apt install m3tal-plugin-<name>`, Debian postinst triggers issue a `SIGHUP` to `m3tal-api`, initiating immediate, zero-downtime plugin discovery.
* **Jellyfin-Style API Key Inheritance**: M3TAL Core generates cryptographically secure API keys and persists the active key to `/etc/m3tal/api_keys.json`. Core injects this key into all plugin subprocesses via `M3TAL_API_KEY`.

### 3. Workload Delegation to Plugins & Hub
* **Strict Separation of Concerns**: Core does not bundle or manage Docker containers. Operators who desire container fleet monitoring or Compose deployment install the dedicated `m3tal-docker` and `m3tal-compose` plugins from `m3tal-plugins`.
* **Hub Distribution**: Pre-packaged application stacks, catalog metadata, and Debian packaging specifications are maintained centrally in `M3tal-Hub` and distributed via `m3tal-apt-key`.

### 4. Atomic Persistent State (`state.db`)
* **SQLite Backend**: All runtime metadata, persistent settings, and API key registries are stored in `/var/lib/m3tal/state.db`.
* **Idempotent Migrations**: Database schemas migrate atomically on daemon startup.

---

## 🛡️ Security & Reliability Model

| Feature | Architectural Implementation |
| :--- | :--- |
| **Debian Baseline** | Compiled with `GOAMD64=v1` and zero cgo dependencies for broad x86-64 CPU compatibility. |
| **Process Isolation** | Plugin crashes or memory leaks in external modules do not crash the primary `m3tal-api` daemon. |
| **Server-Side Key Generation** | Custom client-submitted API key tokens are prohibited; keys are strictly generated server-side. |
| **Sudo-less Host Access** | User group `m3tal` allows non-root management of system state and configurations. |

---

## 🏁 Definition of Done for Core Releases

A release of M3TAL Core is verified when:
1. **Packaging**: Both `dpkg-deb` and `nfpm` package builders produce identical, lint-clean Debian packages (`m3tal_<version>_amd64.deb`).
2. **CPU Baseline Verification**: Binaries execute cleanly without `SIGILL` on Nehalem-era baselines.
3. **Verification**: `go test ./...` passes and the packaged `m3tal-api` serves `/api/v2/system/health` successfully.
4. **Integration Testing**: All Go package test suites (`go test -v ./...`) pass with zero regressions.
