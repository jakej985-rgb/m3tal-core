# M3TAL Core API Reference

M3TAL Core has no CLI. It is a single `m3tal-api` binary serving the embedded Web GUI and a REST/WebSocket API. This document is the reference for that API and the systemd service.

## Running the Service

```bash
./m3tal-api                 # uses API_PORT (default 5050) and API_TOKEN env vars
sudo systemctl status m3tal-api
sudo systemctl restart m3tal-api
journalctl -u m3tal-api -f
```

## Authentication

All `/api/v2/*` routes require a token, supplied as:

* `X-API-Token: <token>` header, or
* `?token=<token>` query parameter (used by the GUI and WebSocket).

Tokens are managed in **Settings → API Keys** (`/api/v2/auth/keys`) or via the `API_TOKEN` env var.

## Endpoints

### Public

| Method | Path | Description |
|---|---|---|
| GET | `/` | Redirects to `/gui/`. |
| GET | `/gui/*` | Embedded Web GUI assets. |
| GET | `/plugins/{name}/*` | Plugin UI static files. |
| GET | `/version`, `/api/v2/version` | Core version metadata. |

### Host telemetry & system monitor

| Method | Path | Description |
|---|---|---|
| GET | `/api/v2/system/stats` | Current telemetry snapshot (CPU, RAM, GPU, disk, network, uptime). |
| GET | `/api/v2/system/metrics` | Sliding-window metrics history. |
| GET | `/api/v2/system/health` | Host and plugin-manager health. |
| GET | `/api/v2/system/specs` | OS, kernel, CPU, memory, swap, GPU, NIC inventory. |
| GET | `/api/v2/system/processes?limit=N` | Top host processes. |
| POST | `/api/v2/system/processes/{pid}/kill` | Terminate a host process. |
| GET | `/api/v2/system/network` | Network interface counters. |
| GET | `/api/v2/ws/events` | WebSocket event stream. |

### API keys

| Method | Path | Description |
|---|---|---|
| GET | `/api/v2/auth/keys` | List API keys. |
| POST | `/api/v2/auth/keys` | Create a key (`{"name": "..."}`; token is server-generated). |
| DELETE | `/api/v2/auth/keys/{id}` | Delete a key. |

### Plugins

| Method | Path | Description |
|---|---|---|
| GET | `/api/v2/plugins` | List discovered/managed plugins. |
| GET | `/api/v2/plugins/catalog` | Plugin catalog (official + custom repositories). |
| GET | `/api/v2/plugins/tabs` | Plugin-contributed dashboard tabs. |
| GET | `/api/v2/plugins/{name}/changelog` | Plugin changelog. |
| GET | `/api/v2/plugins/{name}/download` | Download plugin `.deb`. |
| POST | `/api/v2/plugins/{name}/install` | Install a plugin. |
| POST | `/api/v2/plugins/{name}/uninstall` | Uninstall a plugin. |
| POST | `/api/v2/plugins/{name}/start` | Start a plugin process. |
| POST | `/api/v2/plugins/{name}/stop` | Stop a plugin process. |
| GET | `/api/v2/plugins/{name}/collect` | Latest plugin-collected metrics. |
| POST | `/api/v2/plugins/{name}/action` | Execute a plugin action (`{"action","args"}`). |
| GET/POST | `/api/v2/plugins/{name}/settings` | Read/save plugin settings. |

### Settings & repositories

| Method | Path | Description |
|---|---|---|
| GET | `/api/v2/settings` | All settings (core + plugin sections). |
| POST | `/api/v2/settings/core` | Save core settings. |
| GET | `/api/v2/repositories` | List plugin repositories. |
| POST | `/api/v2/repositories` | Add/update a repository. |
| POST | `/api/v2/repositories/refresh` | Refresh repository metadata. |
| DELETE | `/api/v2/repositories/{id}` | Remove a repository. |

## Response Format

All API responses use the standardized envelope:

```json
{ "status": "success", "data": { ... }, "meta": null, "error": null }
```

## Plugin Integration

Plugins are external processes speaking JSON-RPC 2.0 over stdio (`m3tal.init`, `m3tal.collect`, `m3tal.action`, `m3tal.shutdown`). Core injects `M3TAL_API_KEY` into plugin environments. See `pkg/plugins/protocol.go` for the protocol definition.
