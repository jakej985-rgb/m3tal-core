# ⚡ M3TAL Core

**M3TAL Core** is a high-performance, Go-native **Linux Desktop Application** and homelab container orchestrator. Designed specifically for Linux desktops and home servers, M3TAL provides an integrated control center for monitoring hardware telemetry, orchestrating Docker containers, and managing Traefik reverse proxies—with zero CLI TUI overhead.

---

## 🚀 Key Features

* **🖥️ Dedicated Desktop Application**: Launch `m3tal` or open from your desktop app menu. Launches a sleek, borderless desktop window (`--app` mode with isolated profile and window class `m3tal`) powered by an embedded React/Vite GUI.
* **🌐 Headless Home Server Ready**: Run on headless home servers with `m3tal headless [port]` or `m3tal --headless` to serve the full Web GUI on your local network without requiring a display.
* **📊 Comprehensive Hardware Telemetry**:
  * Real-time CPU & System RAM utilization.
  * Dedicated NVIDIA GPU monitoring (Core load, VRAM used/total, temperatures).
  * System thermals sensor polling.
  * Storage mounts monitoring (including `/mnt/disk1`, `/mnt/disk2`, and root filesystem).
* **⚡ Instant Terminal Telemetry**: Run `m3tal stats` for a fast, clean terminal summary of host metrics, GPU status, temperatures, and storage mounts.
* **🐳 Smart Container Orchestrator**: Multi-stack Docker Compose V2 management (`/docker` / `/opt/m3tal/stack`), health monitoring, container logs streaming, and automated state reconciliation.
* **🔒 Traefik Reverse Proxy & VPN**: Built-in routing configuration, automatic route discovery, SSL termination, and Gluetun VPN leak detection.
* **📦 Fully Self-Contained**: Statically compiled Go binary with embedded Web GUI assets—no Python or Flask runtime required.

---

## 📥 Installation

### Debian / Ubuntu / Linux Mint (.deb)

Download and install the official Debian package:

```bash
# Install deb package
sudo dpkg -i m3tal_0.1.0_amd64.deb

# Or install from APT repository
curl -fsSL https://jakej985-rgb.github.io/m3tal-apt-key/public.key | sudo gpg --dearmor -o /usr/share/keyrings/m3tal-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/m3tal-archive-keyring.gpg] https://jakej985-rgb.github.io/m3tal-apt-key stable main" | sudo tee /etc/apt/sources.list.d/m3tal.list
sudo apt update && sudo apt install -y m3tal
```

### Prerequisites
* **Docker Engine** & **Docker Compose V2** (`docker compose version`)
* **Google Chrome / Chromium** (for standalone desktop app window mode)

---

## 🖥️ Running M3TAL

### 1. Linux Desktop Application
Simply run `m3tal` or click the **M3TAL Control Center** icon in your application launcher:

```bash
m3tal
# or explicitly:
m3tal ui
```

This starts the embedded API backend, serves the Web GUI locally, and opens a dedicated application window with no browser address bar or tabs.

### 2. Headless Home Server Mode
For home servers running without a display (or over SSH):

```bash
# Start headless on default port 5050
m3tal headless

# Start headless on a custom port
m3tal headless 8080

# Or using the flag
m3tal --port 5050 --headless
```

Access the Web GUI from any computer on your network at `http://<server-ip>:5050`.

### 3. Instant Terminal Stats
Check your server or desktop hardware status in the terminal without opening a browser:

```bash
m3tal stats
```

Output includes CPU load, memory utilization, NVIDIA GPU engine load, VRAM, sensor temperatures, and mounted drive capacities.

---

## 🛠️ CLI Command Overview

| Command | Description |
| :--- | :--- |
| `m3tal` / `m3tal ui` | Launch M3TAL Linux Desktop Application window. |
| `m3tal headless [port]` | Start M3TAL in headless mode for home servers. |
| `m3tal stats` | Print comprehensive system, GPU, thermals, and storage metrics to stdout. |
| `m3tal up` | Initialize and start all Docker Compose stacks in `/docker/`. |
| `m3tal down` | Stop and tear down all M3TAL-managed Docker Compose stacks. |
| `m3tal ps` / `m3tal list` | List all running Docker containers and health states. |
| `m3tal logs [container]` | View live container logs. |
| `m3tal compose` | Smart Docker Compose editor & syntax linter. |
| `m3tal config wizard` | Interactive environment setup wizard (`/etc/m3tal/.env`). |
| `m3tal proxy` | Manage reverse proxy routes, auto-discovery, and SSL. |
| `m3tal tray` | Run the background system tray status indicator. |
| `m3tal doctor` | Run comprehensive pre-flight health diagnostics. |

---

## 🗂️ Filesystem Layout

| Path | Purpose |
| :--- | :--- |
| `/usr/bin/m3tal` | Unified M3TAL Desktop App and CLI binary. |
| `/usr/share/applications/m3tal.desktop` | Desktop launcher menu entry. |
| `/etc/m3tal/.env` | Primary configuration file. |
| `/var/lib/m3tal/state.db` | SQLite persistent state database. |
| `/opt/m3tal/stack/` | Canonical Docker Compose stacks and Traefik dynamic configurations. |
| `/docker` | User-facing symlink to `/opt/m3tal/stack/`. |
| `~/.config/m3tal/browser-profile` | Isolated browser profile for the desktop window. |

---

## ⚙️ Background Service (systemd)

To run the M3TAL API daemon continuously in the background on a server:

```bash
# Check service status
systemctl status m3tal-api

# Start / Restart service
sudo systemctl restart m3tal-api

# View live service logs
journalctl -u m3tal-api -f
```

---

## 📚 Documentation

For deeper configuration and architectural guides:
* [Command Reference](docs/COMMAND_REFERENCE.md)
* [Getting Started Guide](docs/GET_STARTED.md)
* [Architecture Vision](docs/ARCHITECTURE_VISION.md)
* [Environment Variables](docs/ENVIRONMENT_VARIABLES.md)
* [Networking & Reverse Proxy](docs/NETWORKING.md)