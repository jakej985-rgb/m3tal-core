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
- `m3tal_session` cookie (HttpOnly, SameSite=Strict; `Secure` when TLS is configured)

`?token=` query parameters are **not** accepted — tokens in URLs leak via browser history, proxy logs, and referrer headers. The bundled GUI keeps its token in memory only (never web storage) and relies on the session cookie for subsequent requests; seed a new browser session once via `?token=` on the GUI page URL.

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

Core injects `M3TAL_API_KEY`/`M3TAL_API_TOKEN` into plugin environments. When the SQLite state store is active, plugins receive a dedicated scoped credential (`m3tal-plugin-runtime`: `system:read`, `plugins:read`, `plugins:write`) instead of the primary full-access key; `M3TAL_API_KEY`/`M3TAL_API_TOKEN`/`API_TOKEN` environment overrides and the `api_keys.json` fallbacks still take precedence for operators who need them. Plugins must not log or expose these credentials. Note that plugins run with the service user's privileges and can read the same credential files, so the scoped key limits accidental exposure and remote blast radius — it is not a sandbox.

## Plugin installation risk

`POST /api/v2/plugins/{name}/install` with `method: "system"` runs `dpkg -i` directly because the API service executes as root — no sudo password is accepted over the API. The resolved `.deb` must live inside a trusted package directory, is re-validated and staged under a private directory, and package-manager argv contains only literal paths. Consider:

- keep the API on localhost or behind a trusted reverse proxy;
- prefer `method: "user"` for testing;
- prefer direct `apt install` for routine administration;
- avoid exposing the API beyond localhost unless authentication and transport are strong.

## WebSocket origin behavior

The WebSocket handler accepts same-origin requests and origins listed in `M3TAL_WS_ALLOWED_ORIGINS`; other cross-origin connections are rejected. Non-browser clients without an `Origin` header are still allowed — if you expose Core remotely, enforce origin restrictions at your reverse proxy or VPN boundary as well.

## Transport security

The API listener serves plaintext HTTP unless `M3TAL_TLS_CERT` and `M3TAL_TLS_KEY` are both set (or a TLS-terminating reverse proxy fronts it). Plain HTTP exposes `X-API-Token` headers and session cookies to network observers — do not expose the port remotely without TLS.

## Filesystem permissions

Review permissions for:

- `/etc/m3tal/.env`
- `~/.config/m3tal/`
- `/var/lib/m3tal/state.db`
- plugin directories
- `/usr/lib/m3tal/plugins`

Plugin UI assets at `/plugins/{name}/` require API authentication (header, Bearer, or the `m3tal_session`/`m3tal_ui_token` cookies set after the first authenticated request); do not place secrets in plugin UI bundles.

## Reporting issues

Report suspected vulnerabilities privately to the repository maintainer rather than opening a public issue with exploit details.
