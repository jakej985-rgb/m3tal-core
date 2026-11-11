# M3TAL Core Environment Variables Reference

This document lists environment variables consumed directly by the `m3tal-api` process. Variables are read from `/etc/m3tal/.env` when packaged, a local `.env` in development, or the process environment.

> Variables for plugin-specific workloads (Docker endpoints, mesh ports, share paths, VPN credentials, etc.) belong to the plugins and are documented by those plugins.

## Quick reference

| Variable | Default | Description |
|---|---:|---|
| `API_PORT` | `5050` | Port for the embedded Web GUI and REST/WebSocket API. |
| `API_TOKEN` | empty | Optional static bootstrap token. When set, it has full API access. API keys created in Settings are also accepted. |
| `M3TAL_STATE_DB` | `/var/lib/m3tal/state.db` or `./data/state.db` | SQLite database path for persisted API keys. |
| `XDG_RUNTIME_DIR` | systemd/user runtime dir | Influences where temporary canonical plugin executable symlinks are created when needed. |

## Details

### `API_PORT`

TCP port used by `m3tal-api` for both:

- `/api/v2/*` REST and WebSocket routes
- `/gui/` embedded Web GUI
- `/plugins/<name>/` plugin UI assets

```bash
API_PORT=5050 ./m3tal-api
```

### `API_TOKEN`

Static bootstrap credential accepted by `X-API-Token`, `Authorization: Bearer`, or the `m3tal_session` cookie. If empty, authentication relies on server-generated API keys.

The primary key material used for plugin injection is resolved in this order:

1. `M3TAL_API_KEY`
2. `M3TAL_API_TOKEN`
3. `API_TOKEN`
4. `~/.config/m3tal/api_keys.json` primary token
5. `/etc/m3tal/api_keys.json` primary token fallback

### `M3TAL_STATE_DB`

Overrides the SQLite path used for persisted API keys. If unset, Core uses `/var/lib/m3tal/state.db` when that directory exists; otherwise it uses `./data/state.db`.

```bash
M3TAL_STATE_DB=/var/lib/m3tal/state.db ./m3tal-api
```

### `M3TAL_TLS_CERT` / `M3TAL_TLS_KEY`

Optional paths to a TLS certificate and private key. When both are set, the API listener serves HTTPS directly via `ListenAndServeTLS` (and the session cookie gains the `Secure` flag). When unset, Core serves plaintext HTTP and logs a startup warning — tokens and session cookies then traverse the network unencrypted, so front the API with a TLS-terminating proxy or set these variables before remote exposure.

### `M3TAL_PLUGINS_DIR`

Optional additional directory scanned for plugins before the standard locations (`~/.config/m3tal/plugins`, `/var/lib/m3tal/plugins`, `/usr/lib/m3tal/plugins`, `/etc/m3tal/plugins`). The process working directory is never searched.

### `M3TAL_PLUGINS_AUTOSTART`

Optional. Set to `0`, `false`, `no`, or `off` to disable automatic startup of discovered plugins — plugins remain listed and can be started explicitly via the API/GUI. Defaults to enabled.

### `M3TAL_WS_ALLOWED_ORIGINS`

Comma-separated extra `Origin` values allowed to open the `/api/v2/ws/events` WebSocket besides same-origin requests (for example `https://dashboard.example.com`). Requests with no `Origin` header (non-browser clients) are always allowed.

## Plugin-injected environment

Core launches plugin subprocesses with a filtered environment: only allowlisted variables (`PATH`, `HOME`, locale/`XDG_*`/`RUNTIME_DIRECTORY` settings, and non-secret `M3TAL_*` variables) are inherited — unrelated host secrets are not passed through. Core additionally sets:

| Variable | Purpose |
|---|---|
| `M3TAL_CORE_VERSION` | Core version string. |
| `M3TAL_PLUGIN_DIR` | Absolute plugin directory containing `plugin.yaml`. |
| `M3TAL_PROCESS_NAME` | Canonical process name such as `m3tal-docker`. |
| `M3TAL_API_KEY` | Scoped API credential injected for Core API access (dedicated `m3tal-plugin-runtime` key: `system:read`, `plugins:read`, `plugins:write`; env overrides and file fallbacks take precedence when no store is active). |
| `M3TAL_API_TOKEN` | Compatibility alias for the same credential. |

## Other environment notes

- There is no `LOG_LEVEL` variable in the current Core binary. Use `journalctl -u m3tal-api -f` for service logs.
- `poll_interval` in core settings is persisted but is not currently consumed by host telemetry collection.
