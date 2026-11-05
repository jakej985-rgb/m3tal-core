# Port Configuration Guide

M3TAL Core uses a single TCP port for everything: the REST/WebSocket API and the embedded Web GUI.

## Core Port Assignment

| Port | Service | Protocol | Status | Description | Managed By |
|------|---------|----------|--------|-------------|------------|
| `5050` | M3TAL Core API & Web GUI | TCP | Required | API, WebSocket events, and embedded GUI (`API_PORT`) | **M3TAL Core** |

### Optional Plugin Ports

External plugins bind their own ports independently of Core. For example, the `m3tal-mesh-node` plugin uses `9321` for peer telemetry. See each plugin's documentation for its port requirements.

## Changing the Port

Edit `/etc/m3tal/.env` (or export the variable before starting the binary):

```bash
API_PORT=5051
```

Then restart the service:

```bash
sudo systemctl restart m3tal-api
```

The Web GUI is then available at `http://<host>:5051/gui/`.

## Verifying the Port

```bash
# Check what is listening
ss -tlnp | grep ':5050'

# Smoke test the public version endpoint
curl -s http://localhost:5050/api/v2/version
```

## Troubleshooting

### "Address already in use"

```bash
sudo lsof -i :5050        # find the conflicting process
sudo kill -9 <PID>        # or stop the conflicting service
sudo systemctl restart m3tal-api
```

### GUI loads but API calls fail

The GUI calls the API on the same origin by default. If you access the GUI through a reverse proxy, ensure `/api/v2/*` and `/api/v2/ws/events` (WebSocket) are proxied to `m3tal-api`. In Vite development mode the GUI proxies to `http://localhost:5050` automatically.
