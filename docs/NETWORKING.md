# 🌐 Networking, Proxy & VPN

> **Moved to plugins.** M3TAL Core does not manage reverse proxies, routes, middleware, VPN tunnels, or container networking. Those features are provided by external plugins.

| Plugin | Provides |
|---|---|
| `m3tal-traefik` | Traefik reverse proxy, TLS certificates, dynamic routing. |
| `m3tal-vpn` | WireGuard/OpenVPN tunnel management and leak checks. |
| `m3tal-mesh` / `m3tal-mesh-node` / `m3tal-mesh-cluster` | Multi-node cluster telemetry federation. |

Core's networking surface is limited to **host network telemetry**: interface counters and IP addresses exposed via `/api/v2/system/network` and rendered by the dashboard's Network widget.

For exposing the Core GUI itself through a reverse proxy, see [Port Configuration](PORT_CONFIGURATION.md) — ensure `/api/v2/*` and the `/api/v2/ws/events` WebSocket are proxied to `m3tal-api`.
