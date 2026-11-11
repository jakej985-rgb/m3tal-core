# M3TAL Core Operations Guide

This guide covers day-to-day operation of `m3tal-api`, its state, configuration, plugin lifecycle, and recovery procedures.

## Service model

M3TAL Core runs as one systemd service:

```bash
sudo systemctl status m3tal-api
sudo systemctl start m3tal-api
sudo systemctl stop m3tal-api
sudo systemctl restart m3tal-api
journalctl -u m3tal-api -f
```

The service loads `/etc/m3tal/.env` when present and listens on `API_PORT` (default `5050`). It serves:

- `/gui/` — embedded dashboard
- `/api/v2/*` — authenticated REST/WebSocket API
- `/plugins/<name>/` — plugin UI bundles

## Configuration locations

| Purpose | Path |
|---|---|
| Environment config | `/etc/m3tal/.env` |
| Environment template | `/etc/m3tal/.env.example` |
| API key database | `M3TAL_STATE_DB`, normally `/var/lib/m3tal/state.db` or `./data/state.db` |
| User settings | `~/.config/m3tal/core_settings.json` |
| Plugin configs | `~/.config/m3tal/plugins/<name>/config.json` |
| Custom repositories | `~/.config/m3tal/repositories.json` |
| API key export for plugins | `~/.config/m3tal/api_keys.json`, fallback `/etc/m3tal/api_keys.json` |

## Plugin discovery

Core scans these directories for `plugin.yaml` or `plugin.yml`:

1. `M3TAL_PLUGINS_DIR` (optional operator override, scanned first)
2. `~/.config/m3tal/plugins`
3. `/var/lib/m3tal/plugins`
4. `/usr/lib/m3tal/plugins`
5. `/etc/m3tal/plugins`

The process working directory is never searched. Set `M3TAL_PLUGINS_AUTOSTART=0` to require explicit plugin starts, or `spec.autostart: false` per plugin.

Discovery runs automatically and also runs on `SIGHUP`. Debian plugin packages use a dpkg trigger to request a SIGHUP after installation.

```bash
sudo kill -HUP $(systemctl show -p MainPID --value m3tal-api)
sudo systemctl restart m3tal-api   # use when SIGHUP is unavailable
```

Core starts discovered plugins automatically unless an operator explicitly stopped them. Stopping a plugin through the API or GUI marks it explicitly stopped and prevents auto-restart on the next discovery cycle.

## Plugin process states

| State | Meaning |
|---|---|
| `stopped` | Not running. |
| `starting` | Process launched; `m3tal.init` handshake in progress. |
| `running` | Init handshake completed and RPC calls are available. |
| `degraded` | Process is running but the init handshake failed. |
| `crashed` | Process exited or failed before/during startup. |

## Plugin lifecycle operations

Use the Web GUI Plugin Manager or the API:

```bash
curl -H "X-API-Token: <key>" http://localhost:5050/api/v2/plugins
curl -X POST -H "X-API-Token: <key>" http://localhost:5050/api/v2/plugins/<name>/start
curl -X POST -H "X-API-Token: <key>" http://localhost:5050/api/v2/plugins/<name>/stop
curl -H "X-API-Token: <key>" "http://localhost:5050/api/v2/plugins/<name>/collect"
curl -X POST -H "X-API-Token: <key>" \
  -H "Content-Type: application/json" \
  -d '{"action":"<action_id>","target":"<target>","args":{}}' \
  http://localhost:5050/api/v2/plugins/<name>/action
```

## Installing plugins

Preferred system installation:

```bash
sudo apt update
sudo apt install m3tal-plugin-<name>
```

Core can also install a discovered `.deb` through `POST /api/v2/plugins/{name}/install`:

- `method: "sudo"` runs `sudo -S dpkg -i` and resolves dependencies with `apt-get install -f -y` when needed.
- `method: "user"` extracts the package into `~/.config/m3tal/plugins/<name>` for per-user testing.

The sudo method accepts an optional password in the request body. Treat this as sensitive and prefer direct `apt install` on untrusted networks.

## State database and backups

The SQLite store persists API keys and related state. Back it up before upgrades or key migration:

```bash
sudo cp /var/lib/m3tal/state.db /var/lib/m3tal/state.db.backup
# or, for a development checkout:
cp ./data/state.db ./data/state.db.backup
```

Removing the `m3tal` package preserves configuration and state. `apt purge` removes package-owned configuration and may remove state according to Debian maintainer scripts.

## Logs and troubleshooting

Start with:

```bash
journalctl -u m3tal-api -f
curl -s http://localhost:5050/api/v2/version
curl -H "X-API-Token: <key>" http://localhost:5050/api/v2/system/health
curl -H "X-API-Token: <key>" http://localhost:5050/api/v2/plugins
```

Common issues:

- **GUI unavailable**: confirm `m3tal-api` is active and open `/gui/`, not `/`.
- **401 responses**: verify the key exists, has the required scope, and was not deleted.
- **403 responses**: API keys are scope-limited; the static `API_TOKEN` bypasses scope checks.
- **Plugin missing**: verify manifest location, YAML syntax, and process name.
- **Plugin degraded/crashed**: inspect `journalctl`, plugin README requirements, and the plugin's own logs or stderr.
- **Plugin UI 404**: the plugin must ship `ui/dist/index.html` or the configured UI entry.

## Upgrade and rollback

```bash
sudo apt update
sudo apt install --only-upgrade m3tal
sudo systemctl restart m3tal-api
```

To roll back, install a retained older package version explicitly:

```bash
sudo apt install m3tal=<version>
sudo systemctl restart m3tal-api
```

APT only offers versions still present in the repository index and subject to retention policy.
