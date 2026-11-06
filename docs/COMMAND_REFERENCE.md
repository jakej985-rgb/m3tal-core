# M3TAL Core API Reference

M3TAL Core is a single `m3tal-api` binary. It serves the embedded React Web GUI, the REST API under `/api/v2`, and the plugin host supervisor. There is no separate `m3tal` CLI in the current package.

## Running the service

```bash
./m3tal-api                 # uses API_PORT (default 5050) and API_TOKEN env vars
sudo systemctl status m3tal-api
sudo systemctl restart m3tal-api
journalctl -u m3tal-api -f
```

## Authentication

Every `/api/v2/*` route requires a token. Core accepts the token in this order:

1. `X-API-Token: <token>` header
2. `Authorization: Bearer <token>` header
3. `?token=<token>` query parameter

The query parameter is primarily for the embedded GUI and browser-based plugin UI loading. Prefer headers for scripts and integrations because URLs can be logged by proxies and browsers.

Accepted credentials are:

- `API_TOKEN` from the environment or `/etc/m3tal/.env`, when configured. This static token has full access.
- API keys created under **Settings → API Keys** or `POST /api/v2/auth/keys`.
- The primary local key synced to `~/.config/m3tal/api_keys.json` for plugin access.

## Response envelope

Successful responses use:

```json
{
  "status": "success",
  "data": {},
  "meta": {},
  "error": null
}
```

Errors use the same envelope with `status: "error"` and an `error` object:

```json
{
  "status": "error",
  "data": null,
  "meta": {},
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Unauthorized",
    "details": null
  }
}
```

## Public routes

| Method | Path | Auth | Description |
|---|---|---:|---|
| GET | `/` | No | Redirects to `/gui/`. |
| GET/HEAD | `/gui/*` | No | Embedded Web GUI assets. |
| GET | `/plugins/{name}` | No | Redirects to `/plugins/{name}/`. |
| GET | `/plugins/{name}/*` | No | Static plugin UI files. |
| GET | `/version` | No | Core version metadata. |
| GET | `/api/v2/version` | No | Core version metadata. |

## API key management

All key-management routes require the `keys:manage` scope unless the request uses the static `API_TOKEN`.

| Method | Path | Scope | Description |
|---|---|---|---|
| GET | `/api/v2/auth/keys` | `keys:manage` | List API keys. |
| POST | `/api/v2/auth/keys` | `keys:manage` | Create a key. |
| DELETE | `/api/v2/auth/keys/{id}` | `keys:manage` | Delete a key. |

### Create key request

```json
{
  "name": "Home Assistant",
  "permissions": ["system:read", "plugins:read"],
  "direction": "local",
  "target": ""
}
```

Fields:

- `name` — required display/application name.
- `permissions` — optional list of scopes. Empty means full access for backward compatibility.
- `direction` — `local` or `remote`. `local` is the default and creates a server-generated key for inbound access to this Core instance. `remote` records a credential issued by another machine or application.
- `token` / `key` — required only for `direction: "remote"`; local keys cannot supply a custom token.
- `target` — optional label for the remote target.

### Permission scopes

| Scope | Grants |
|---|---|
| `*` | Full access. |
| `system:read` | Telemetry, metrics, health, specs, network, process list, settings, and repositories reads. |
| `system:write` | Process termination. Also implies `system:read`. |
| `plugins:read` | Plugin list/catalog/tabs/changelog/download/collect/settings reads. Also implies read access for write operations. |
| `plugins:write` | Plugin install, uninstall, start, stop, action, and settings writes. Implies `plugins:read`. |
| `settings:write` | Core settings and repository management. |
| `keys:manage` | API key list/create/delete. |
| `mesh:access` | Reserved for mesh/federation integrations. |

## Host telemetry and system monitor

| Method | Path | Scope | Description |
|---|---|---|---|
| GET | `/api/v2/system/stats` | `system:read` | Current telemetry snapshot: CPU, RAM, GPU, disk, network, uptime, hostname. |
| GET | `/api/v2/system/metrics` | `system:read` | Sliding-window metrics history. |
| GET | `/api/v2/system/health` | `system:read` | Host health checks. Current checks cover the state database and root filesystem; it does not report plugin-manager health. |
| GET | `/api/v2/system/specs` | `system:read` | OS, kernel, CPU, memory, GPU, and NIC inventory. |
| GET | `/api/v2/system/processes?limit=N` | `system:read` | Top host processes. `limit` defaults to 150. |
| POST | `/api/v2/system/processes/{pid}/kill` | `system:write` | Terminate a host process. |
| GET | `/api/v2/system/network` | `system:read` | Network interface counters. |
| GET | `/api/v2/ws/events` | `system:read` | WebSocket event stream endpoint. The current production code exposes the endpoint, but Core does not yet publish host/plugin events through it. |

## Plugins

Plugin routes use the manifest `metadata.name` or a supported alias such as the canonical process name.

| Method | Path | Scope | Description |
|---|---|---|---|
| GET | `/api/v2/plugins` | `plugins:read` | List discovered and managed plugins. |
| GET | `/api/v2/plugins/catalog` | `plugins:read` | Plugin catalog from built-in defaults plus `catalog.json` overrides. |
| GET | `/api/v2/plugins/tabs` | `plugins:read` | Plugin-contributed dashboard tabs. |
| GET | `/api/v2/plugins/{name}/changelog` | `plugins:read` | Plugin changelog when available. |
| GET | `/api/v2/plugins/{name}/download` | `plugins:read` | Download a discovered plugin `.deb`. |
| POST | `/api/v2/plugins/{name}/install` | `plugins:write` | Install a plugin package. |
| POST | `/api/v2/plugins/{name}/uninstall` | `plugins:write` | Uninstall a plugin package. |
| POST | `/api/v2/plugins/{name}/start` | `plugins:write` | Start a plugin process. |
| POST | `/api/v2/plugins/{name}/stop` | `plugins:write` | Stop a plugin process. |
| GET | `/api/v2/plugins/{name}/collect` | `plugins:read` | Request the latest plugin-collected metrics/tables. |
| POST | `/api/v2/plugins/{name}/action` | `plugins:write` | Execute a plugin action. |
| GET | `/api/v2/plugins/{name}/settings` | `plugins:read` | Read plugin settings. |
| POST | `/api/v2/plugins/{name}/settings` | `plugins:write` | Save plugin settings. |

### Plugin install request

```json
{
  "method": "sudo",
  "password": "optional sudo password",
  "mode": "optional package mode",
  "package": "optional explicit package name"
}
```

- `method: "sudo"` runs `sudo -S dpkg -i <deb>` and can resolve dependencies with `apt-get install -f -y`.
- `method: "user"` extracts the package into `~/.config/m3tal/plugins/<name>` without system installation.
- `mode` and `package` help select alternate artifacts such as mesh node versus mesh cluster packages.

Security note: the sudo method accepts a password over the authenticated API. Use it only on trusted networks or behind a trusted reverse proxy, and prefer `user` mode or direct `apt install` when possible.

### Plugin action request

```json
{
  "action": "restart_container",
  "target": "container-name-or-id",
  "args": {}
}
```

The exact `action` and `target` values are plugin-specific. Use the plugin's README, `m3tal.init` action list, or `/api/v2/plugins/{name}/collect` output to discover supported actions.

## Settings and repositories

| Method | Path | Scope | Description |
|---|---|---|---|
| GET | `/api/v2/settings` | `system:read` | Core settings, plugin setting sections, categories, and current plugin config values. |
| POST | `/api/v2/settings/core` | `settings:write` | Save core settings. |
| GET | `/api/v2/repositories` | `system:read` | Official and custom plugin repositories plus discovered APT packages. |
| POST | `/api/v2/repositories` | `settings:write` | Add or update a custom repository. |
| POST | `/api/v2/repositories/refresh` | `settings:write` | Refresh repository metadata. |
| DELETE | `/api/v2/repositories/{id}` | `settings:write` | Remove a custom repository. |

Core settings currently contain `node_name`, `poll_interval`, and `theme` and are stored in `~/.config/m3tal/core_settings.json`. `poll_interval` is persisted for compatibility but is not currently consumed by the fixed host telemetry scheduler.

Plugin settings are stored under `~/.config/m3tal/plugins/<plugin>/config.json`.

## Plugin protocol

Core supervises external plugins as subprocesses and communicates over JSON-RPC 2.0 on stdin/stdout. The implemented methods are:

- `m3tal.init`
- `m3tal.collect`
- `m3tal.action`

`m3tal.shutdown` is not currently implemented by the Core host. Core stops plugins by closing stdin, sending `SIGINT`, and then terminating the process if needed.

Core injects these environment variables into plugin processes:

- `M3TAL_CORE_VERSION`
- `M3TAL_PLUGIN_DIR`
- `M3TAL_PROCESS_NAME`
- `M3TAL_API_KEY`
- `M3TAL_API_TOKEN`

Core resolves the injected API key in this order:

1. `M3TAL_API_KEY`
2. `M3TAL_API_TOKEN`
3. `API_TOKEN`
4. `~/.config/m3tal/api_keys.json` primary token
5. `/etc/m3tal/api_keys.json` primary token fallback

Plugin search paths are:

1. `./plugins`
2. `~/.config/m3tal/plugins`
3. `/var/lib/m3tal/plugins`
4. `/usr/lib/m3tal/plugins`
5. `/etc/m3tal/plugins`

`M3TAL_PLUGINS_DIR` is not currently implemented; do not rely on it.
