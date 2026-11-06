# M3TAL Core Security Notes

This document describes the current security boundaries and operational risks of M3TAL Core. It is not a substitute for a full security audit.

## Service exposure

`m3tal-api` serves HTTP on `API_PORT` (default `5050`) with no built-in TLS. For remote access, place it behind a trusted reverse proxy or VPN and terminate TLS there.

The GUI and plugin UI routes are public at the HTTP layer:

- `/`
- `/gui/*`
- `/plugins/{name}/*`
- `/version`
- `/api/v2/version`

All `/api/v2/*` routes require authentication except `/api/v2/version`.

## Authentication

Core accepts tokens through:

- `X-API-Token`
- `Authorization: Bearer`
- `?token=`

Prefer headers. Query tokens can appear in browser history, proxy logs, and referrer headers.

The static `API_TOKEN` has full access and bypasses permission scopes. Server-generated API keys can be restricted with scopes:

- `system:read` / `system:write`
- `plugins:read` / `plugins:write`
- `settings:write`
- `keys:manage`
- `mesh:access`
- `*`

Write scopes imply their corresponding read scopes.

## API key storage

API keys are persisted in the SQLite state database and exported for plugin use in:

1. `~/.config/m3tal/api_keys.json`
2. `/etc/m3tal/api_keys.json` fallback

Protect these files and the state database with filesystem permissions. Do not commit them.

## Plugin trust boundary

Plugins are subprocesses running with the permissions of `m3tal-api`. In the packaged service model, the daemon runs as root, so plugins can perform privileged operations.

Only install plugins from trusted sources. Review plugin manifests, source code, Debian package contents, and required host permissions before enabling them.

Core injects `M3TAL_API_KEY`/`M3TAL_API_TOKEN` into plugin environments. Plugins must not log or expose these credentials.

## Plugin installation risk

`POST /api/v2/plugins/{name}/install` supports a sudo mode that accepts a sudo password in the request body. This is convenient but sensitive:

- use it only over trusted networks;
- prefer `method: "user"` for testing;
- prefer direct `apt install` for routine administration;
- avoid exposing the API beyond localhost unless authentication and transport are strong.

## WebSocket origin behavior

The WebSocket handler currently allows cross-origin connections. If you expose Core remotely, enforce origin restrictions at your reverse proxy or VPN boundary.

## Filesystem permissions

Review permissions for:

- `/etc/m3tal/.env`
- `~/.config/m3tal/`
- `/var/lib/m3tal/state.db`
- plugin directories
- `/usr/lib/m3tal/plugins`

Plugin UI assets are served without API authentication at `/plugins/{name}/`; do not place secrets in plugin UI bundles.

## Reporting issues

Report suspected vulnerabilities privately to the repository maintainer rather than opening a public issue with exploit details.
