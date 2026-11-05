# 🐳 Docker & Container Workloads

> **Moved to plugins.** M3TAL Core no longer contains any Docker, container, or Compose management code. Core is only the Web GUI dashboard, system monitor (including GPU telemetry), settings/API keys, and the plugin host supervisor.

Container fleet monitoring, Compose editing, image management, and stack lifecycle are provided by external plugins maintained in [m3tal-plugins](https://github.com/jakej985-rgb/m3tal-plugins):

| Plugin | Provides |
|---|---|
| `m3tal-docker` | Container fleet monitor, logs, lifecycle controls, image pruning. |
| `m3tal-compose` | Compose visual editor, validation, stack lifecycle. |
| `m3tal-stacks` | Curated Compose template catalog. |

Install them from the Web GUI's **Plugin Manager** (`/plugins` catalog) or via `apt install m3tal-plugin-<name>`.

Core itself has **no Docker dependency** — the systemd service does not require `docker.service`, and no container runtime is needed to run the dashboard.
