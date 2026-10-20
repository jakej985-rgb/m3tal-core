# M3TAL CLI Command Reference

This document provides a comprehensive cheat-sheet for the M3TAL command-line interface (CLI).

## Table of Contents

1.  [Core Commands](#core-commands)
2.  [Configuration Management](#configuration-management)
3.  [Dashboard Management](#dashboard-management)
4.  [System Management](#system-management)
5.  [Systemd Service Management](#systemd-service-management)
6.  [Direct Docker Compose Commands](#direct-docker-compose-commands)
7.  [APT Installation](#apt-installation)

---

## Core Commands

These are the primary commands for interacting with the M3TAL ecosystem.

### `m3tal` (or `m3tal ui`)

Launches the M3TAL Linux Desktop Application. Opens a dedicated, borderless application window (using Chrome/Chromium `--app` mode with isolated profile and window class `m3tal`) with real-time system metrics, container orchestration, and storage monitoring.

**Usage Example:**
```bash
m3tal
# or explicitly:
m3tal ui
```

### `m3tal headless [port]` (or `m3tal --headless`)

Starts M3TAL in headless mode without opening a graphical window. Perfect for headless home servers, remote servers, or background daemon operation. Serves the Web GUI at `http://<host>:<port>` (default port: `5050`).

**Usage Example:**
```bash
# Start headless on default port 5050
m3tal headless

# Start headless on custom port 8080
m3tal headless 8080

# Or via flag
m3tal --headless --port 5050
```

### `m3tal stats`

Displays instant host system telemetry in the terminal: CPU load, Memory usage, NVIDIA GPU & VRAM utilization, system temperatures, and mounted drive usage (such as `/mnt/disk1`, `/mnt/disk2`).

**Usage Example:**
```bash
m3tal stats
```

### `m3tal tray`

Launches the M3TAL background system tray monitor with live metric tooltips and quick actions.

**Usage Example:**
```bash
m3tal tray
```

---

## Configuration Management

Manage M3TAL's environment variables, which control its behavior and integrate with your system.

### `m3tal init`

Generates the default `/etc/m3tal/.env` file. Use this command on first installation to create the base configuration.

**Usage Example:**
```bash
m3tal init
```

### `m3tal config wizard`

Launches an interactive wizard to guide you through configuring `/etc/m3tal/.env`. This is the recommended method for initial setup or significant configuration changes.

**Usage Example:**
```bash
m3tal config wizard
```

### `m3tal config set KEY VALUE`

Sets a single environment variable in the `/etc/m3tal/.env` file.

**Usage Example:**
```bash
m3tal config set DOMAIN mym3tal.local
```

### `m3tal config get KEY`

Reads and displays the value of a single environment variable from `/etc/m3tal/.env`.

**Usage Example:**
```bash
m3tal config get DASHBOARD_PORT
```

### `m3tal config scan`

Lists all environment variables across all managed stacks, including their current values.

**Usage Example:**
```bash
m3tal config scan
```

### `m3tal config list`

Displays the current contents of the `/etc/m3tal/.env` file.

**Usage Example:**
```bash
m3tal config list
```

---

## Dashboard Management

Commands specifically for managing the M3TAL Dashboard container.

### `m3tal dashpass [username] [password]`

Updates the password for a dashboard user. If `username` and `password` are omitted, the command will prompt interactively.

**Usage Example (interactive):**
```bash
m3tal dashpass
```

**Usage Example (with arguments):**
```bash
m3tal dashpass admin newsecurepassword123
```

### `m3tal dash up`

Pulls the latest dashboard Docker Compose configuration from GitHub and then starts the dashboard container. This command respects the `DASHBOARD_EXPOSE_MODE` setting in your `.env` file.

**Usage Example:**
```bash
m3tal dash up
```

### `m3tal dash down`

Stops and removes the M3TAL Dashboard container.

**Usage Example:**
```bash
m3tal dash down
```

### `m3tal dash restart`

Restarts the M3TAL Dashboard container.

**Usage Example:**
```bash
m3tal dash restart
```

### `m3tal dash logs`

Streams the logs from the M3TAL Dashboard container in real-time.

**Usage Example:**
```bash
m3tal dash logs
```

### `m3tal dash status`

Shows the current status of the M3TAL Dashboard container.

**Usage Example:**
```bash
m3tal dash status
```

---

## System Management

Commands for managing all running M3TAL stacks.

### `m3tal up`

Runs `docker compose up` across all `*-compose.yml` files found in the `/docker/` directory. This command starts all your deployed services.

**Usage Example:**
```bash
m3tal up
```

### `m3tal down`

Runs `docker compose down` for all stacks managed by M3TAL. This command stops and removes all containers, networks, and volumes defined in your `*-compose.yml` files.

**Usage Example:**
```bash
m3tal down
```

### `m3tal logs`

Streams aggregated logs from all running M3TAL stacks. This provides a centralized view of your system's activity.

**Usage Example:**
```bash
m3tal logs
```

---

## Systemd Service Management

The M3TAL API daemon is managed by `systemd`. You can interact with it using `systemctl` and `journalctl`.

### `systemctl status m3tal-api`

Checks the status of the `m3tal-api.service`.

**Usage Example:**
```bash
systemctl status m3tal-api
```

### `journalctl -u m3tal-api -f`

Streams the logs for the `m3tal-api.service` in real-time, similar to `m3tal dash logs` but for the API daemon.

**Usage Example:**
```bash
journalctl -u m3tal-api -f
```

---

## Direct Docker Compose Commands

As a fallback or for advanced debugging, you can use Docker Compose commands directly. M3TAL manages its stacks within the `/docker` directory.

### Running all stacks:

**Start all services:**
```bash
docker compose -f /docker/*.yml up -d
```

**Stop all services:**
```bash
docker compose -f /docker/*.yml down
```

### Managing the dashboard specifically:

**Start the dashboard (respecting local or traefik mode):**
```bash
# Ensure you have the correct override file downloaded and in place.
# For local mode:
docker compose -f /docker/m3tal-compose.yml -f /docker/m3tal-compose.local.yml up -d m3tal-dashboard

# For traefik mode:
docker compose -f /docker/m3tal-compose.yml -f /docker/m3tal-compose.traefik.yml up -d m3tal-dashboard
```

**Stop the dashboard:**
```bash
docker compose -f /docker/m3tal-compose.yml stop m3tal-dashboard
```

**Restart the dashboard:**
```bash
docker compose -f /docker/m3tal-compose.yml restart m3tal-dashboard
```

**View dashboard logs:**
```bash
docker compose -f /docker/m3tal-compose.yml logs -f m3tal-dashboard
```

---

## APT Installation

This section outlines the commands to install or update M3TAL on your system via APT.

**Usage Example:**
```bash
# 1. Add the GPG signing key
curl -fsSL https://jakej985-rgb.github.io/m3tal-apt-key/public.key | sudo gpg --dearmor -o /usr/share/keyrings/m3tal-archive-keyring.gpg

# 2. Add the APT repository
echo "deb [signed-by=/usr/share/keyrings/m3tal-archive-keyring.gpg] https://jakej985-rgb.github.io/m3tal-apt-key stable main" | sudo tee /etc/apt/sources.list.d/m3tal.list

# 3. Update package list and install M3TAL
sudo apt update && sudo apt install -y m3tal
```