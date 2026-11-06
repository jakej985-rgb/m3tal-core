# 🔍 Diagnostics & Troubleshooting Guide

This guide covers diagnostic procedures and error resolution for M3TAL Core.

---

## 🩺 Health Endpoints

Quick programmatic checks against the running service:

```bash
# Version (public, no auth)
curl -s http://localhost:5050/api/v2/version

# Host health (database and root filesystem; requires token)
curl -H "X-API-Token: <key>" http://localhost:5050/api/v2/system/health

# Live telemetry snapshot
curl -H "X-API-Token: <key>" http://localhost:5050/api/v2/system/stats
```

---

## 🪵 Service Logs & systemd

M3TAL Core runs as the native systemd service `m3tal-api.service`.

```bash
# Runtime status
systemctl status m3tal-api

# Live logs
journalctl -u m3tal-api -f

# Logs since last boot
journalctl -u m3tal-api -b

# Restart after config changes
sudo systemctl restart m3tal-api
```

Common log signals:

* `� State database:` — SQLite store opened successfully.
* `🚀 M3TAL Core API starting` — HTTP listener is up.
* `🔄 Received SIGHUP: refreshing plugins` — plugin rediscovery triggered (e.g., after `apt install m3tal-plugin-*`).
* `⚠️ Could not open state database` — check `/var/lib/m3tal` permissions and `M3TAL_STATE_DB`.

---

## 🧩 Plugin Diagnostics

* **Plugin not appearing**: confirm the plugin directory is under a search path (`./plugins`, `~/.config/m3tal/plugins`, `/var/lib/m3tal/plugins`, `/usr/lib/m3tal/plugins`, or `/etc/m3tal/plugins`) and contains a valid `plugin.yaml` or `plugin.yml`. Core rediscovers every 5 seconds and also refreshes on `SIGHUP`; restart `m3tal-api` if discovery still fails.
* **Plugin UI 404 at `/plugins/<name>/`**: the plugin must ship `ui/dist` (or `ui/index.html`) inside its directory.
* **Plugin auth failures**: verify the API key file exists at `~/.config/m3tal/api_keys.json` (or `/etc/m3tal/api_keys.json` as a fallback) and that the plugin receives `M3TAL_API_KEY` or `M3TAL_API_TOKEN`.

---

## 🌐 GUI / Networking Issues

* **GUI won't load**: verify the service is listening (`ss -tlnp | grep :5050`) and open `http://localhost:5050/gui/` (note the `/gui/` path — `/` redirects there).
* **Port conflict**: see [Port Configuration](PORT_CONFIGURATION.md).
* **WebSocket events not streaming**: ensure the reverse proxy (if any) upgrades WebSocket connections for `/api/v2/ws/events`.

---

## 🐞 Development Debugging

```bash
# Run tests
go test ./...

# Build everything (GUI + binary)
make build

# Run the API directly (there is no LOG_LEVEL variable)
./m3tal-api
```
