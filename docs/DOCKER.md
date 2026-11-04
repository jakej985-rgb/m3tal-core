# M3TAL Docker and Container Guide

| [🚀 Overview](../README.md) | [⚙️ Environment](ENVIRONMENT_VARIABLES.md) | [🛠️ Commands](COMMAND_REFERENCE.md) | [🌐 Networking](NETWORKING.md) | [🐳 Docker](DOCKER.md) |
| :---: | :---: | :---: | :---: | :---: |

This guide documents the M3TAL container architecture, Docker Compose stack orchestration, default container fleet, and container lifecycle operations.

---

## 📋 Overview and Prerequisites

M3TAL uses **Docker Engine** and **Docker Compose V2** as its foundational container orchestration runtime.

### Key Requirements
- **Docker Engine**: Installed on the host system with the Docker daemon active (`systemctl status docker`).
- **Docker Compose V2**: Modern compose plugin accessible via `docker compose`.
- **Socket Permissions**: Non-root operations access the Docker daemon via `unix:///var/run/docker.sock` requiring membership in the host `docker` group (`sudo usermod -aG docker $USER`).
- **Core Orchestrator**: M3TAL interacts with Docker through internal stack managers, standard Docker Compose execution, and the official `m3tal-docker` plugin.

---

## ⚙️ How Docker Works in M3TAL

M3TAL works with Docker by orchestrating multi-container compose stacks via Docker Compose V2 and the `m3tal-docker` plugin. Here is how container workflows function:

1. **Stack Declarations**: Stacks define services in `*-compose.yml` files stored in `/docker`.
2. **Container Supervision**: The M3TAL CLI runs `m3tal up` to launch containers in detached mode, while the daemon monitors container health.
3. **Ingress Routing**: Traefik automatically discovers running containers on the `proxy` network through container labels.
4. **Plugin Telemetry**: The `m3tal-docker` plugin connects to the Docker socket at `/var/run/docker.sock` to collect live resource stats and stream logs.

---

## 🏗️ Stacks and the `/docker` Directory

M3TAL organizes all deployed container workloads into declarative Compose stacks located in `/docker` (symlinked from `/opt/m3tal/stack`).

### Directory Layout
```text
/docker/
├── m3tal-compose.yml          # Core system stack (dashboard, metrics, internal agents)
├── m3tal-compose.local.yml    # Local port override mapping
├── m3tal-compose.traefik.yml  # Reverse proxy label override
├── routing-compose.yml        # Traefik ingress & cloudflared tunnel stack
├── media-compose.yml          # Media services (e.g., Jellyfin)
└── ai-compose.yml             # Local AI/LLM stack (Ollama)
```

Each stack file follows standard Docker Compose specifications with labeled Traefik routing rules and volume mounts.

---

## 🚦 Ingress & Traefik Setup

Traefik serves as the primary reverse proxy and SSL terminator for M3TAL container workloads.

### Default Port Mappings
- **HTTP**: Port `80` (configurable via `TRAEFIK_WEB_PORT`, default: `80`)
- **HTTPS**: Port `443` (configurable via `TRAEFIK_WEBHTTPS_PORT`, default: `443`)
- **Traefik Dashboard**: `127.0.0.1:8081:8080` (Internal administrative dashboard)

### Automatic Service Discovery
Services join the `proxy` Docker bridge network and declare Traefik labels:
```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.jellyfin.rule=Host(`media.localhost`)"
  - "traefik.http.services.jellyfin.loadbalancer.server.port=8096"
```

---

## 📦 Default Container Fleet

M3TAL ships with predefined container stacks tailored for self-hosted infrastructure:

| Container | Image | Stack | Role / Purpose |
| :--- | :--- | :--- | :--- |
| `traefik` | `traefik:latest` | `routing` | Ingress proxy, router, and SSL terminator |
| `cloudflared` | `cloudflare/cloudflared:latest` | `routing` | Cloudflare Zero Trust tunnel for remote access |
| `ollama` | `ollama/ollama:latest` | `ai` | Local neural network and LLM inference engine |
| `jellyfin` | `jellyfin/jellyfin:latest` | `media` | Open-source media streaming and transcoding server |

---

## ⚡ Container Management Commands

### Whole-Fleet Management
- **Start All Stacks**:
  ```bash
  m3tal up
  # Direct fallback:
  docker compose -f /docker/*.yml up -d
  ```
- **Stop All Stacks**:
  ```bash
  m3tal down
  # Direct fallback:
  docker compose -f /docker/*.yml down
  ```
- **View Aggregated Logs**:
  ```bash
  m3tal logs
  ```

### Single Container Operations
- **Inspect Running Containers**:
  ```bash
  # Query M3TAL container fleet:
  m3tal containers
  # Or through assistant:
  go run ./cmd/m3tal-ai "list containers"
  ```
- **View Container Logs**:
  ```bash
  m3tal logs <container-name>
  ```
- **Lifecycle Control**:
  ```bash
  m3tal restart container <container-name>
  m3tal stop container <container-name>
  m3tal start container <container-name>
  ```

---

## 🔌 The `m3tal-docker` Plugin

The official **M3TAL Docker Plugin** (`/home/m3tal/apps/m3tal-plugins/m3tal-docker`) provides live telemetry and container fleet management via JSON-RPC 2.0 stdio without embedding container runtime dependencies into `m3tal-core`.

### Architecture & IPC
- **Socket**: Interacts with the Docker Engine API via `unix:///var/run/docker.sock`.
- **Telemetry Collection**: Emits real-time CPU, RAM, network I/O, and container health states.
- **Actions**: Handles `start`, `stop`, `restart`, `remove`, and `prune_images`.
- **Dashboard UI**: Mounts a dedicated micro-frontend web console at `/containers`.

---

## 🩺 Preflight Diagnostics & Health Checks

Use the preflight diagnostic command to verify Docker Engine readiness:
```bash
m3tal doctor
```

The doctor command checks:
1. Docker daemon socket availability (`/var/run/docker.sock`).
2. Docker group membership for current user.
3. Bridge network health (`m3tal`, `proxy`).
4. Port availability on ports `80`, `443`, and `5050`.
