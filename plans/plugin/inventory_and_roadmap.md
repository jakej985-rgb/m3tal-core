# M3TAL Plugin Catalog: Inventory & Roadmap

**Status**: Active Architectural Plan  
**Target Architecture**: Self-Contained Plugins (`all code for said plugin lives with the plugin`)  
**Directory**: `plans/plugin/`

---

## 1. Executive Summary

M3TAL Core is designed as a high-performance, lightweight Linux system supervisor and control plane. Rather than bundling third-party tools, container orchestrators, and AI engines into the core repository, all non-core capabilities reside in the centralized **`m3tal-plugins`** repository (`jakej985-rgb/m3tal-plugins`).

When a plugin is developed and installed:
1. All plugin code lives directly in `m3tal-plugins/<plugin-name>/` (no intermediate wrapper folders).
2. Debian packages (`m3tal-plugin-<name>_<version>_amd64.deb`) are built in `m3tal-plugins` and dispatched to `jakej985-rgb/m3tal-apt-key` for official GPG signing and APT repository ingestion.
3. The core supervisor executes the plugin as a dedicated canonical process (`m3tal-<plugin-name>`).
4. The Web GUI dynamically mounts the plugin's tab and UI without requiring hardcoded components in the core frontend.
5. If a plugin is not installed, it consumes zero background processes and does not appear in the sidebar.

---

## 2. Existing Plugins (Current Inventory to be Externalized)

These components currently exist across `plugins/`, `deploy/plugins/`, or hardcoded GUI/API routes, and are being cataloged for standardization and externalization:

### 2.1. `docker-monitor` (Container Fleet Manager)
- **Role**: Real-time Docker container management, live log streaming, container lifecycle controls, image pruning.
- **Current Locations**:
  - `plugins/docker-monitor/` (`main.go`, `plugin.yaml`, compiled `docker-plugin`)
  - `deploy/plugins/services/docker-monitor.yml`
  - Hardcoded GUI page: `gui/src/pages/Containers.jsx`
  - Hardcoded API routes: `/api/v2/services`, `/api/v2/containers/*`, `/api/v2/docker/*`
- **Standardized Target**:
  - Standalone repository / package: `m3tal-plugin-docker`
  - Process Name: `m3tal-docker`
  - Capabilities: `collector`, `actions`, `ui_view`, `websocket_stream`
  - UI: Self-contained frontend bundle mounted dynamically at `/containers`.

---

### 2.2. `stack-orchestrator` (Compose Multi-Service Stacks)
- **Role**: Declarative multi-container stack deployment, live health checks, and lifecycle management (up/down/restart) using Docker Compose specifications.
- **Current Locations**:
  - `deploy/plugins/stacks/` (`m3tal.yml`, `network.yml`, `maintenance.yml`, `routing.yml`, `ai.yml`)
  - Hardcoded GUI page: `gui/src/pages/Stacks.jsx`
  - Hardcoded API routes: `/api/v2/stacks/*`
- **Standardized Target**:
  - Standalone package: `m3tal-plugin-stacks`
  - Process Name: `m3tal-stacks`
  - Capabilities: `orchestrator`, `actions`, `ui_view`
  - UI: Self-contained frontend bundle mounted dynamically at `/stacks`.

---

### 2.3. `ai-engine` (Modular Local AI & LLM Runner)
- **Role**: Local LLM and inference server manager (Ollama, vLLM, LocalAI), background inference job queue, model downloading and cataloging.
- **Current Locations**:
  - `deploy/plugins/ai/` (`ai-chat.env`, `ai-local.env`, `ai-coder.env`, `routes.js`, `adapter.js`, `config.yaml`)
  - Hardcoded GUI page: `gui/src/pages/AI.jsx`
  - Hardcoded API routes: `/api/v2/ai/*`, `/api/v2/queue/*`
- **Standardized Target**:
  - Standalone package: `m3tal-plugin-ai`
  - Process Name: `m3tal-ai`
  - Capabilities: `inference_runner`, `job_queue`, `ui_view`
  - UI: Self-contained frontend bundle mounted dynamically at `/ai`.

---

### 2.4. `traefik-gateway` (Ingress & Reverse Proxy)
- **Role**: Traefik v3 reverse proxy orchestration, automated Let's Encrypt SSL/TLS certificates, domain routing, and middleware pipelines.
- **Current Locations**:
  - `deploy/plugins/traefik/` (`traefik.yml`, `traefik-gateway.yml`)
  - `core/plugins/traefik_gen.go`
  - Hardcoded API routes: `/api/v2/routes`, `/api/v2/config/cloudflared`
- **Standardized Target**:
  - Standalone package: `m3tal-plugin-traefik`
  - Process Name: `m3tal-traefik`
  - Capabilities: `proxy_generator`, `route_provider`

---

### 2.5. `gluetun-vpn` (Network Privacy & Tunnel Egress)
- **Role**: Secure VPN tunnel connectivity (WireGuard/OpenVPN) with container egress routing and live public IP/leak status checking.
- **Current Locations**:
  - `core/networking/vpn/`
  - Hardcoded API routes: `/api/v2/vpn/*`
  - UI component: `NetworkSummary.jsx` (VPN tunnel widget)
- **Standardized Target**:
  - Standalone package: `m3tal-plugin-vpn`
  - Process Name: `m3tal-vpn`
  - Capabilities: `network_tunnel`, `status_checker`

---

### 2.6. `http-middleware` (Security & Auth Presets)
- **Role**: HTTP traffic security middlewares: Basic Authentication, Rate Limiting, Security Headers, and Authelia Single-Sign-On.
- **Current Locations**:
  - `deploy/plugins/middleware/` (`basicauth.yml`, `ratelimit.yml`, `security-headers.yml`)
  - `deploy/plugins/services/authelia.yml`
- **Standardized Target**:
  - Bundled middleware package or modular config packs under `m3tal-plugin-traefik`.

---

## 3. Planned Plugins (Roadmap)

The following plugins are scheduled for development under the new standardized plugin architecture:

| Plugin Name | Category | Process Name | Description | Key Capabilities |
|---|---|---|---|---|
| **`smart-disk`** | Hardware Storage | `m3tal-smart` | Drive health telemetry, NVMe wear life, S.M.A.R.T. self-tests, and temperature history. | `collector`, `ui_widget`, `alerts` |
| **`samba-manager`** | File Sharing | `m3tal-samba` | Graphical Samba/NFS share creator, guest permissions, user credentials, and active connections. | `actions`, `ui_view`, `system_admin` |
| **`system-bench`** | Diagnostics | `m3tal-bench` | One-click benchmarking for CPU multi-threading, GPU compute, and disk read/write throughput. | `benchmark_runner`, `ui_view` |
| **`tailscale-mesh`** | Networking | `m3tal-tailscale` | Tailscale VPN node connector, exit node selector, and MagicDNS routing status. | `network_mesh`, `status_checker` |
| **`backup-sync`** | Maintenance | `m3tal-backup` | Scheduled snapshot and offsite backup manager utilizing Borg, Rclone, or local block mirroring. | `cron_runner`, `storage_sync`, `ui_view` |
| **`service-media`** | Applications | `m3tal-media` | Automated deployment and health telemetry for Jellyfin, Plex, Radarr, and Sonarr instances. | `app_template`, `status_checker` |
| **`log-watcher`** | Security | `m3tal-logwatch` | Real-time systemd journal filter, failed SSH login detector, and incident alerting. | `log_collector`, `alerts` |

---

## 4. Packaging, Signing & Release Pipeline
 
1. **Centralized Plugin Monorepo**: All plugins live in `jakej985-rgb/m3tal-plugins/<plugin-name>/` with their own manifests, JSON-RPC 2.0 handlers, static UI bundles, and embedded `.agents/agents/plugin-maintainer/agent.md`.
2. **Automated Debian Builds**: `m3tal-plugins/scripts/build-plugin-deb.sh <plugin-name>` packages the plugin into `m3tal-plugin-<name>_<version>_amd64.deb`.
3. **Official GPG Signing**: Built `.deb` packages are dispatched via GitHub Actions to `jakej985-rgb/m3tal-apt-key` (`publish-deb`), signed with the central M3TAL repository key, and published to the APT repository for one-command installation:
   ```bash
   sudo apt update && sudo apt install m3tal-plugin-<name>
   ```
