# Getting Started with M3TAL Core

This guide walks you through installing, configuring, and running **M3TAL Core** — the Web Dashboard and System Monitor with GPU support and a plugin host.

---

## 1. Overview

**M3TAL Core** is a lightweight, bare-metal Go daemon (`m3tal-api`) with an embedded React/Vite Web GUI on port `5050`. It provides:

* Real-time system monitoring: CPU, RAM, NVIDIA GPU (load, VRAM, temperature), thermals, storage mounts, network interfaces, and top processes.
* A settings page for core preferences and Jellyfin-style API key management.
* A plugin host supervisor that discovers, runs, and mounts external plugins (`m3tal-docker`, `m3tal-mesh`, etc.) over JSON-RPC 2.0 stdio.

> [!NOTE]
> **Ecosystem Responsibility Boundary**:  
> M3TAL Core is strictly the dashboard, system monitor, and plugin supervisor. External workloads—Docker container fleet monitoring (`m3tal-docker`), Compose editing (`m3tal-compose`), reverse proxies (`m3tal-traefik`), VPN tunnels (`m3tal-vpn`), local AI inference (`m3tal-ai`), and multi-node clustering (`m3tal-mesh`)—are maintained as **extra plugins** in [m3tal-plugins](https://github.com/jakej985-rgb/m3tal-plugins) or showcased in [M3tal-Hub](https://github.com/jakej985-rgb/M3tal-Hub).

---

## 2. Prerequisites

* **Operating System**: Linux AMD64 (`Debian 12/13`, `Ubuntu 22.04/24.04`, `Linux Mint 21/22`, or compatible x86_64 distributions).
* **Browser**: Any modern browser to access the Web GUI.
* *(Note: Docker is NOT required for Core; it is only needed if you install container-related plugins.)*

---

## 3. Install via APT

```bash
# Recommended one-line bootstrap installer
curl -fsSL https://jakej985-rgb.github.io/m3tal-apt-key/install.sh | sudo bash

# Or configure the repository manually:
curl -fsSL https://jakej985-rgb.github.io/m3tal-apt-key/public.key | sudo gpg --dearmor -o /usr/share/keyrings/m3tal-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/m3tal-archive-keyring.gpg] https://jakej985-rgb.github.io/m3tal-apt-key stable main" | sudo tee /etc/apt/sources.list.d/m3tal.list
sudo apt update && sudo apt install -y m3tal
```

The package installs `m3tal-api`, the embedded GUI, the `m3tal-api.service` systemd unit, and a desktop launcher entry.

---

## 4. Configuration

Configuration lives in `/etc/m3tal/.env` (created from `.env.example` on install):

```bash
sudo nano /etc/m3tal/.env
```

Key variables:

* **`API_PORT`**: Port for the Web GUI and REST API (default `5050`).
* **`API_TOKEN`**: Optional static bootstrap token. You can also generate API keys later in **Settings → API Keys**.
* **`M3TAL_STATE_DB`**: SQLite state path (default `/var/lib/m3tal/state.db`).
* **`M3TAL_PLUGINS_DIR`**: Optional extra plugin search directory.

See [Environment Variables](ENVIRONMENT_VARIABLES.md) for the full list.

---

## 5. Launch the Dashboard

The `m3tal` package enables `m3tal-api.service` automatically on install. Manage it with systemd:

```bash
sudo systemctl status m3tal-api
sudo systemctl restart m3tal-api
journalctl -u m3tal-api -f
```

Then open the Web GUI:

```
http://localhost:5050/gui/
```

For remote access, use `http://<server-ip>:5050/gui/` from any machine on your network.

### Running from source (development)

```bash
# Build GUI assets + API binary
make build

# Start the server
./m3tal-api
```

---

## 6. Using the GUI

* **Dashboard**: CPU, RAM, GPU, disk, network, and event widgets. Use the layout editor to rearrange or hide widgets.
* **System Monitor**: Detailed host telemetry, performance history, system specs, network interfaces, and process management.
* **Plugins**: Discover, install, start/stop, and configure external plugins. Plugin tabs and widgets mount dynamically into the dashboard.
* **Plugin Manager**: Browse the plugin catalog and manage repositories.
* **Settings**: Core settings (node name, telemetry interval, theme), API connection, API keys, and plugin-provided settings.

---

## 7. API Access

All API routes are under `/api/v2` and require authentication via the `X-API-Token` header (or `?token=` query param for WebSocket/downloads). Create keys in **Settings → API Keys**.

Quick smoke test:

```bash
curl -H "X-API-Token: <your-key>" http://localhost:5050/api/v2/system/stats
```

See [Command Reference](COMMAND_REFERENCE.md) (API reference) for the full endpoint list.
