# 📡 M3TAL Core Observability & Telemetry

M3TAL Core exposes real-time host telemetry, health, and event endpoints for monitoring and alerting.

---

## 📊 Telemetry Endpoints

All endpoints are served by `m3tal-api` on `API_PORT` (default `5050`) under `/api/v2` and require an API token (`X-API-Token` header, `Authorization: Bearer`, or the `m3tal_session` cookie):

| Endpoint | Description |
|---|---|
| `GET /api/v2/system/stats` | Current snapshot: CPU, RAM, GPU (util/VRAM/temp/model), disk, network, uptime, hostname. |
| `GET /api/v2/system/metrics` | Sliding-window metrics history (~5 minutes at 5s intervals). |
| `GET /api/v2/system/health` | Host health status for the state database and root filesystem. |
| `GET /api/v2/system/specs` | Static hardware/OS inventory (CPU model, cores, memory, GPU model, NICs). |
| `GET /api/v2/system/network` | Per-interface byte counters. |
| `GET /api/v2/system/processes` | Top processes by CPU/memory. |
| `GET /api/v2/ws/events` | WebSocket endpoint. Core currently exposes the transport but does not publish host/plugin events through it. |
| `GET /api/v2/version` | Version metadata (public). |

Example:

```bash
curl -H "X-API-Token: <key>" http://localhost:5050/api/v2/system/stats
```

## 🖥️ GPU Telemetry

When an NVIDIA GPU is present, `nvidia-smi` is queried for utilization, temperature, VRAM used/total, and model name. These appear in `gpu_*` fields of `/api/v2/system/stats` and in the Dashboard/System Monitor widgets. Non-NVIDIA GPUs are not currently reported by Core.

## 🪵 Logs

Core logs to stdout and is captured by `systemd-journald`:

```bash
journalctl -u m3tal-api -f
```

Plugin stdout/stderr is supervised by the plugin host. Plugin lifecycle and metrics are available through `/api/v2/plugins*` endpoints; the Core WebSocket endpoint is present but currently has no production event producers.

## � Alerting

There is no built-in alerting engine in Core. Consumers should poll `/api/v2/system/health` and `/api/v2/system/stats`. `/api/v2/ws/events` is available for future event producers but should not be treated as a live telemetry feed today. Host-level alerting belongs to plugins or external monitoring.
