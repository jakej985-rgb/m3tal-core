# Environment Variables Reference

| [🚀 Overview](../README.md) | [⚙️ Environment](ENVIRONMENT_VARIABLES.md) | [🛠️ Build](BUILD_CONFIGURATION.md) | [🌐 Networking](NETWORKING.md) | [🤖 Architecture](ARCHITECTURE_VISION.md) |
| :---: | :---: | :---: | :---: | :---: |

This document provides a complete reference for all environment variables used by the M3TAL stack.

## Quick Reference

| Variable | Required | Default | Description |
| :--- | :--- | :--- | :--- |
| `BASE_STORAGE_PATH` | **YES** | `./data` | Host path for media + config (mounted to `/mnt` in containers) |
| `DASHBOARD_SECRET` | **YES** | `change_me_immediately` | Session signing key (run `openssl rand -hex 32`) |
| `API_TOKEN` | **YES** | `change_me_api_token` | Dashboard-to-backend auth token |
| `ADMIN_PASSWORD` | **YES** | `admin_pass` | Initial dashboard login password |
| `CF_TUNNEL_TOKEN` | Conditional | - | Required only if using Cloudflare Tunnel for internet access |
| `DASHBOARD_PORT` | No | `8082` | Port for the web dashboard |
| `HTTP_PORT` | No | `8080` | Port for HTTP traffic (Traefik entrypoint) |
| `STATE_DIR` | No | `./state` | Directory for state files |
| `LOG_LEVEL` | No | `info` | Logging level (debug, info, warn, error) |
| `NETWORK_NAME` | No | `m3tal` | Docker network name for services |
| `LOCAL_IP` | No | `127.0.0.1` | Local IP for internal routing |
| `DOMAIN` | No | `localhost` | Root domain for service discovery |
| `VPN_USER` | No | `user` | VPN username (optional) |
| `VPN_PASSWORD` | No | `password` | VPN password (optional) |
| `MEDIA_PATH` | No | `./data/media` | Media storage path (relative to `BASE_STORAGE_PATH`) |
| `CONFIG_PATH` | No | `./data/config` | Configuration storage path |
| `DOWNLOADS_PATH` | No | `./data/downloads` | Downloads storage path |

---

## Detailed Variable Documentation

### Core Configuration

#### `DASHBOARD_PORT`

- **Required**: No
- **Default**: `8082`
- **Description**: The port on which the M3TAL web dashboard will be accessible
- **Example**: `DASHBOARD_PORT=8082`
- **Usage**: In `.env` and `docker-compose.yml`

#### `HTTP_PORT`

- **Required**: No
- **Default**: `8080`
- **Description**: The HTTP port for Traefik entrypoint. **Port 80 must be free** for Traefik to bind.
- **Example**: `HTTP_PORT=80`
- **Usage**: In `.env` and Traefik configuration

#### `STATE_DIR`

- **Required**: No
- **Default**: `./state`
- **Description**: Directory for persisting M3TAL state (metrics, logs)
- **Example**: `STATE_DIR=./state`
- **Usage**: In `.env` and Go daemon

#### `LOG_LEVEL`

- **Required**: No
- **Default**: `info`
- **Description**: Logging verbosity level
- **Valid values**: `debug`, `info`, `warn`, `error`
- **Example**: `LOG_LEVEL=info`
- **Usage**: In `.env`

---

### Authentication

#### `DASHBOARD_SECRET`

- **Required**: No (auto-generated on init)
- **Default**: Auto-generated via `crypto/rand`
- **Description**: Secret key for signing dashboard sessions. **Should be a random hex string**.
- **Example**: `DASHBOARD_SECRET=a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6`
- **Generation**: `openssl rand -hex 32`
- **Usage**: In `.env`

#### `API_TOKEN`

- **Required**: No (auto-generated on init)
- **Default**: Auto-generated
- **Description**: Token used for secure dashboard-to-backend API communication
- **Example**: `API_TOKEN=dashboard-secret-token-12345`
- **Generation**: `openssl rand -hex 16`
- **Usage**: In `.env` and API server

#### `ADMIN_PASSWORD`

- **Required**: No
- **Default**: `admin_pass`
- **Description**: Initial password for the admin user in the dashboard
- **Example**: `ADMIN_PASSWORD=strongpassword123`
- **Usage**: In `.env` and `m3tal dashpass` command

---

### Network Configuration

#### `NETWORK_NAME`

- **Required**: No
- **Default**: `m3tal`
- **Description**: Docker network name used for inter-container communication
- **Example**: `NETWORK_NAME=m3tal`
- **Usage**: In `.env` and docker-compose files

#### `LOCAL_IP`

- **Required**: No
- **Default**: `127.0.0.1`
- **Description**: Local IP address of the host machine (for internal routing)
- **Example**: `LOCAL_IP=192.168.1.100`
- **Usage**: In `.env` and Docker Compose

#### `DOMAIN`

- **Required**: No
- **Default**: `localhost`
- **Description**: Root domain for service discovery and Traefik routing
- **Example**: `DOMAIN=homelab.local`
- **Usage**: In `.env` and Traefik labels

---

### VPN Configuration (Optional)

#### `VPN_USER`

- **Required**: No
- **Default**: `user`
- **Description**: VPN username for authenticated media streaming
- **Example**: `VPN_USER=myvpnuser`
- **Usage**: In `.env`

#### `VPN_PASSWORD`

- **Required**: No
- **Default**: `password`
- **Description**: VPN password for authenticated media streaming
- **Example**: `VPN_PASSWORD=mystrongpassword`
- **Usage**: In `.env`

---

### Storage Configuration

#### `BASE_STORAGE_PATH`

- **Required**: No
- **Default**: `./data`
- **Description**: **Base path for media and configuration storage**. This directory should exist and be writable.
- **Examples**:
  - Linux: `BASE_STORAGE_PATH=/mnt/media`
  - macOS: `BASE_STORAGE_PATH=/Users/username/m3tal-data`
  - Windows: `BASE_STORAGE_PATH=C:\m3tal-data`
- **Usage**: In `.env` and docker-compose volume mounts
- **Important**: Ensure this directory exists and has proper permissions before starting services

#### `MEDIA_PATH`

- **Required**: No
- **Default**: `./data/media`
- **Description**: Path to media files (relative to `BASE_STORAGE_PATH`)
- **Example**: `MEDIA_PATH=./data/media`
- **Usage**: In `.env` and docker-compose

#### `CONFIG_PATH`

- **Required**: No
- **Default**: `./data/config`
- **Description**: Path to configuration files (relative to `BASE_STORAGE_PATH`)
- **Example**: `CONFIG_PATH=./data/config`
- **Usage**: In `.env` and docker-compose

#### `DOWNLOADS_PATH`

- **Required**: No
- **Default**: `./data/downloads`
- **Description**: Path for download directory (relative to `BASE_STORAGE_PATH`)
- **Example**: `DOWNLOADS_PATH=./data/downloads`
- **Usage**: In `.env` and docker-compose

---

### Required for Cloudflare Tunnel

#### `CF_TUNNEL_TOKEN`

- **Required**: Yes (if using Cloudflare tunnel)
- **Default**: None
- **Description**: Cloudflare tunnel token for exposing services to the internet
- **Example**: `CF_TUNNEL_TOKEN=abcd1234-...`
- **Usage**: In `.env` and routing-compose.yml

---

## Environment Variable Usage by Component

### Core Services (.env only)

- `DASHBOARD_PORT`, `HTTP_PORT`, `STATE_DIR`, `LOG_LEVEL`

### Dashboard & API

- `DASHBOARD_SECRET`, `API_TOKEN`, `ADMIN_PASSWORD`

### Networking

- `NETWORK_NAME`, `LOCAL_IP`, `DOMAIN`

### Storage

- `BASE_STORAGE_PATH`, `MEDIA_PATH`, `CONFIG_PATH`, `DOWNLOADS_PATH`

### Cloudflare Tunnel

- `CF_TUNNEL_TOKEN`

### Optional (VPN)

- `VPN_USER`, `VPN_PASSWORD`

---

## Setting Environment Variables

### Option 1: Using the Configuration Wizard

```bash
./m3tal config wizard
```

### Option 2: Manual Edit

1. Copy template: `cp template.env .env`
2. Edit `.env` with your values
3. Run: `./m3tal init`

### Option 3: Environment-Driven

```bash
export DASHBOARD_PORT=9000
export DOMAIN=example.com
./m3tal up
```

---

## Pre-flight Check

Before running `m3tal up`, verify:

1. **Base Storage Path Exists**:

   ```bash
   # Linux/macOS
   ls -la $BASE_STORAGE_PATH

   # Windows PowerShell
   Test-Path $env:BASE_STORAGE_PATH
   ```

2. **Ports Are Free**:

   ```bash
   # Check if ports 80, 443, 8080 are free
   netstat -ano | findstr :80
   netstat -ano | findstr :443
   netstat -ano | findstr :8080
   ```

3. **Docker Permissions**:

   ```bash
   docker ps  # Should work without sudo
   ```

   If it requires sudo, add user to docker group: `sudo usermod -aG docker $USER`
