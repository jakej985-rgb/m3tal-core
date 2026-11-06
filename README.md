# ⚡ M3TAL Core

**M3TAL Core** is a high-performance, Go-native **Web Dashboard and System Monitor** for Linux desktops and servers. It serves an embedded React GUI for hardware telemetry, system performance, settings, and API key management—plus a plugin host supervisor for everything else.

---

## 🚀 Key Features

* **🖥️ Embedded Web GUI**: A sleek React/Vite dashboard served directly by the Go API binary on port `5050` (`/gui/`). No separate frontend server or Node.js runtime required.
* **🌐 Headless Server Ready**: Runs as a native systemd service (`m3tal-api.service`) to serve the full Web GUI across your local network without requiring a display.
* **📊 Comprehensive System Monitoring**:
  * Real-time CPU & System RAM utilization with history graphs.
  * Dedicated NVIDIA GPU monitoring (core load, VRAM used/total, temperatures, model).
  * System thermals and motherboard sensor polling.
  * Storage mounts monitoring (physical disks, removable media, network shares).
  * Host network interface statistics and top process inspector.
* **⚙️ Settings & API Keys**: Core settings (node identity, telemetry interval, theme) plus Jellyfin-style server-generated API keys for programmatic access.
* **🔌 Extensible Plugin Host Supervisor**: Discovers and supervises external plugins over standard JSON-RPC 2.0 `stdio`, automatically injecting API authentication (`M3TAL_API_KEY`) and mounting frontend UI tabs dynamically into the dashboard (`/plugins/<name>/`).
* **📦 Fully Self-Contained**: Statically compiled Go binary with embedded Web GUI assets—no Python, Node.js, container runtime, or CLI required to run the core dashboard and system monitor.

---

> [!NOTE]
> **🧩 Extra Plugins & Ecosystem Repositories**:  
> M3TAL Core strictly limits its scope to the system dashboard, telemetry monitor, settings, and plugin supervisor. External workloads—such as Docker container fleet monitoring (`m3tal-docker`), Compose editing (`m3tal-compose`), reverse proxies (`m3tal-traefik`), VPN tunnels (`m3tal-vpn`), local AI inference (`m3tal-ai`), and multi-node clustering (`m3tal-mesh`)—are maintained as **extra plugins** in [m3tal-plugins](https://github.com/jakej985-rgb/m3tal-plugins). Application templates, repository distribution, and packaging standards belong to [M3tal-Hub](https://github.com/jakej985-rgb/M3tal-Hub).

---

## 📥 Installation

The Linux AMD64 Debian package targets the Go `GOAMD64=v1` baseline, including older
x86-64 CPUs such as the Intel Core i5-750. The packaged Go binary is built with
cgo disabled. See [the Debian CPU baseline and release checklist](docs/DEBIAN_CPU_BASELINE.md).

### Debian / Ubuntu / Linux Mint (.deb)

Install via the official M3tal APT repository:

```bash
# Universal one-line bootstrap installer (recommended)
curl -fsSL https://jakej985-rgb.github.io/m3tal-apt-key/install.sh | sudo bash

# Or configure the APT repository manually:
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://jakej985-rgb.github.io/m3tal-apt-key/public.key | sudo gpg --dearmor -o /etc/apt/keyrings/m3tal-archive-keyring.gpg
echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/m3tal-archive-keyring.gpg] https://jakej985-rgb.github.io/m3tal-apt-key stable main" | sudo tee /etc/apt/sources.list.d/m3tal.list
sudo apt update && sudo apt install -y m3tal
```

### Prerequisites
* **Linux AMD64** (`Debian 12/13`, `Ubuntu 22.04/24.04`, `Linux Mint 21/22`, or compatible)
* **Google Chrome / Chromium** (optional, only if you want a desktop window wrapper)
* *(Note: Docker Engine is only needed if installing container-related plugins like `m3tal-docker`)*

---

## 🖥️ Running M3TAL Core

M3TAL Core is a single binary: `m3tal-api`. It serves both the REST/WebSocket API and the embedded Web GUI.

### 1. Direct run (development)

```bash
# Build the GUI assets and API binary
make build

# Start the server (default port 5050)
./m3tal-api
```

Open `http://localhost:5050/gui/` in your browser. Set `API_TOKEN` and `API_PORT` via environment or `/etc/m3tal/.env`.

### 2. systemd service

```bash
# Check service status
systemctl status m3tal-api

# Start / Restart service
sudo systemctl restart m3tal-api

# View live service logs
journalctl -u m3tal-api -f
```

The service has no Docker dependency; it only needs host access for telemetry and plugin supervision.

---

## 🌐 API Surface

All API routes live under `/api/v2` and require an API token (`X-API-Token` or `Authorization: Bearer` header; `?token=` is also accepted for GUI/plugin UI use):

| Endpoint | Description |
| :--- | :--- |
| `GET /api/v2/version` | Core version metadata (public). |
| `GET /api/v2/system/stats` | Current host telemetry snapshot (CPU, RAM, GPU, disk, network). |
| `GET /api/v2/system/metrics` | Sliding-window metrics history. |
| `GET /api/v2/system/health` | Host health (database and filesystem). |
| `GET /api/v2/system/processes` | Top host processes. |
| `POST /api/v2/system/processes/{pid}/kill` | Terminate a host process. |
| `GET /api/v2/system/specs` | OS, kernel, CPU, memory, GPU, and NIC inventory. |
| `GET /api/v2/system/network` | Network interface counters. |
| `GET /api/v2/ws/events` | WebSocket endpoint (no production event producers yet). |
| `GET/POST/DELETE /api/v2/auth/keys` | API key management. |
| `GET /api/v2/plugins*` | Plugin discovery, lifecycle, collect, action, settings. |
| `GET/POST /api/v2/settings*` | Core and plugin settings. |
| `GET/POST/DELETE /api/v2/repositories*` | Plugin repositories. |

Plugin UIs are served at `/plugins/{name}/`.

---

## 🗂️ Filesystem Layout

| Path | Purpose |
| :--- | :--- |
| `/usr/bin/m3tal-api` | M3TAL Core Web GUI + API binary. |
| `/usr/share/applications/m3tal.desktop` | Desktop launcher menu entry. |
| `/etc/m3tal/.env` | Primary configuration file. |
| `/etc/m3tal/.env.example` | Default configuration template. |
| `/var/lib/m3tal/state.db` | SQLite state database when `/var/lib/m3tal` is present; otherwise `./data/state.db`. |
| `./plugins`, `~/.config/m3tal/plugins`, `/var/lib/m3tal/plugins`, `/usr/lib/m3tal/plugins`, `/etc/m3tal/plugins` | Plugin search paths. |

---

## 📚 Documentation

* [Getting Started Guide](docs/GET_STARTED.md)
* [Command & API Reference](docs/COMMAND_REFERENCE.md)
* [Environment Variables Reference](docs/ENVIRONMENT_VARIABLES.md)
* [Port Configuration Guide](docs/PORT_CONFIGURATION.md)
* [Diagnostics & Troubleshooting](docs/DIAGNOSTICS.md)
* [Observability & Metrics](docs/OBSERVABILITY.md)
* [Build Configuration](docs/BUILD_CONFIGURATION.md)
* [Debian CPU Baseline](docs/DEBIAN_CPU_BASELINE.md)
* [Testing & Verification](docs/TESTING.md)
* [Command & API Reference](docs/COMMAND_REFERENCE.md)
* [Operations Guide](docs/OPERATIONS.md)
* [Plugin Authoring](docs/PLUGIN_AUTHORING.md)
* [Security Notes](docs/SECURITY.md)
* [Development Guide](docs/DEVELOPMENT.md)
* [Release Manual](docs/RELEASE.md)
* [Architecture Vision](docs/ARCHITECTURE_VISION.md)
* [System Upgrade Guide](docs/UPGRADE.md)
