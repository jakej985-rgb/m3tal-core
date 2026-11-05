# M3TAL Core Environment Variables Reference

This document lists the environment variables used by **M3TAL Core** itself. All variables are read from `/etc/m3tal/.env` (or a local `.env` in development) and can also be set directly in the process environment.

> Variables for workload features (Traefik ports, VPN credentials, storage paths, PUID/PGID, etc.) belong to external plugins and are documented by those plugins, not by Core.

## Quick Reference

| Variable | Default | Description |
|---|---|---|
| `API_PORT` | `5050` | Port for the embedded Web GUI and REST API. |
| `API_TOKEN` | *(empty)* | Bearer/`X-API-Token` credential for API and plugin access. API keys generated in Settings are also accepted. |
| `M3TAL_STATE_DB` | `/var/lib/m3tal/state.db` (or `./data/state.db`) | SQLite state database path (API keys). |
| `M3TAL_PLUGINS_DIR` | *(unset)* | Extra plugin search directory, searched in addition to the default paths. |

## Details

### `API_PORT`

TCP port the `m3tal-api` binary listens on. Serves both the REST/WebSocket API (`/api/v2/*`) and the embedded Web GUI (`/gui/`).

```bash
API_PORT=5050 ./m3tal-api
```

### `API_TOKEN`

Static bootstrap token accepted alongside server-generated API keys. Leave empty to rely solely on keys created under **Settings → API Keys** in the Web GUI. The primary key is synced to `~/.config/m3tal/api_keys.json` for plugin discovery.

### `M3TAL_STATE_DB`

Overrides the default SQLite path. If unset, Core uses `/var/lib/m3tal/state.db` when that directory exists, otherwise `./data/state.db`.

### `M3TAL_PLUGINS_DIR`

Adds an extra directory to the plugin search path. Default search paths are `./plugins`, `$HOME/.config/m3tal/plugins`, `/var/lib/m3tal/plugins`, `/usr/lib/m3tal/plugins`, and `/etc/m3tal/plugins`.
