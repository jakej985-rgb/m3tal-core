# M3TAL Networking Guide

| [🚀 Overview](../README.md) | [⚙️ Environment](ENVIRONMENT_VARIABLES.md) | [🛠️ Build](BUILD_CONFIGURATION.md) | [🌐 Networking](NETWORKING.md) | [🤖 Architecture](ARCHITECTURE_VISION.md) |
| :---: | :---: | :---: | :---: | :---: |

This guide explains the M3TAL network architecture, Docker Compose networking, and how to debug network issues.

---

## 📋 Overview

M3TAL uses a multi-network Docker architecture to separate concerns:

| Network | Purpose | Isolation Level |
| :--- | :--- | :--- |
| `m3tal` | Internal control plane | All M3TAL services |
| `proxy` | External traffic routing | Traefik + services |
| `api_internal` | Backend API communication | Dashboard + API server |

---

## 🚦 Ingress & Traefik Configuration

The `m3tal` binary automatically orchestrates the Traefik gateway.

### Configuration Loading

* **Static Config**: Managed via `deploy/stack/routing-compose.yml`.
* **Dynamic Config**: The orchestrator scans `deploy/stack/traefik/` for any `.yml` or `.toml` files and mounts them as dynamic configurations.
* **Provider**: Uses the Docker Provider to automatically discover services with the `traefik.enable=true` label.
* **Dependency**: Traefik relies on the `proxy` Docker network. If this network is not initialized correctly by the orchestrator, ingress routing will fail.

### Connectivity Check

* If you encounter a 404 or 503 error:
    1. Verify the container is on the `proxy` network: `docker inspect <container> | grep proxy`.
    2. Check the Traefik dashboard at `traefik.localhost:8080` (Internal only).

---

## 🏗️ Architecture

```text
[ Internet ]
     ↓
[ Cloudflare Tunnel ] (cloudflared)
     ↓
[ Traefik ] (Port 80, 443)
     ↓
[ Services on `proxy` network ]
```

Internal services communicate over the `m3tal` network:

```text
[ Dashboard (m3tal) ] ←→ [ API Server (m3tal) ]
     ↓
[ Container Orchestrator (m3tal) ]
```

---

## 🔧 Docker Compose Network Configuration

### Default Networks

M3TAL creates the following Docker networks automatically:

| Network Name | Driver | Compose File |
| :--- | :--- | :--- |
| `m3tal` | bridge | network-compose.yml |
| `proxy` | bridge | routing-compose.yml |
| `api_internal` | bridge | api.yml |

### Network Names in Docker Compose

Each docker-compose.yml file uses these networks:

**network-compose.yml**:

```yaml
networks:
  m3tal:
    driver: bridge
```

**routing-compose.yml**:

```yaml
networks:
  proxy:
    external: true
```

**api.yml**:

```yaml
networks:
  api_internal:
    driver: bridge
```

---

## 🛠️ Troubleshooting Network Issues

### 1. Check Docker Networks

List all networks:

```bash
docker network ls
```

Expected networks:

* `m3tal` (internal services)
* `proxy` (Traefik routing)

### 2. Check Network Connectivity

Test if a container is on the correct network:

```bash
# Check Radarr's networks
docker inspect radarr | grep -A5 '"Networks"'

# Check Traefik's networks
docker inspect traefik | grep -A5 '"Networks"'
```

### 3. Debug Traefik Routing

Traefik requires services to be on the `proxy` network. Check logs:

```bash
# Check Traefik logs for network errors
docker logs traefik --tail 100 | grep -E "(IP address|network|404)"
```

### 4. Test Service Accessibility

From the host, test if a service is reachable:

```bash
# Test Radarr on proxy network
curl -H "Host: radarr.${DOMAIN}" http://radarr:7878

# Test Traefik directly
curl -H "Host: ${DOMAIN}" http://localhost:8080
```

### 5. Inspect Container Network Settings

```bash
# Get container IP on each network
docker inspect <container_name> --format='{{range $key, $value := .NetworkSettings.Networks}}{{$key}}={{$value.IPAddress}} {{end}}'
```

---

## 🐛 Common Network Errors

### Error: "unable to find IP address"

**Cause**: Container is not on the `proxy` network or has no IP assigned.

**Solution**:

1. Ensure the service is attached to the `proxy` network in docker-compose.yml
2. Restart the container: `docker-compose restart <service>`

### Error: "server is ignored" in Traefik logs

**Cause**: Traefik cannot reach the container on the specified port.

**Solution**:

1. Verify the service port matches the Traefik label: `traefik.http.services.<name>.loadbalancer.server.port=<port>`
2. Ensure the service exposes the port in docker-compose

### Error: 404 Page Not Found

**Cause**: Traefik routing rule doesn't match the Host header.

**Solution**:

1. Check your `DOMAIN` environment variable
2. Verify the Host header: `curl -H "Host: yourdomain.com" http://localhost`

---

## 🔐 Internal Network Communication

The dashboard talks to the backend via the `m3tal` network:

```text
[ Dashboard ] (port 8082) 
     ↓ HTTP (internal)
[ API Server ] (port 8090) on `m3tal` network
     ↓ 
[ Orchestrator ] on `m3tal` network
```

### Debugging Internal API Communication

```bash
# Test API server is reachable from dashboard
docker exec -it m3tal-dashboard curl -s http://api:8090/health

# Test orchestrator is reachable
docker exec -it m3tal-dashboard curl -s http://orchestrator:8091/status
```

---

## 🚀 Network Configuration Reference

### Port Mappings

| Host Port | Container Port | Service | Network |
| :--- | :--- | :--- | :--- |
| 80 | 80 | Traefik | proxy |
| 443 | 443 | Traefik (HTTPS) | proxy |
| 8080 | 8080 | Traefik Dashboard | proxy |
| 8082 | 8082 | M3TAL Dashboard | m3tal |
| 8090 | 8090 | API Server | m3tal |
| 8091 | 8091 | Orchestrator | m3tal |

---

## ✅ Verification Checklist

Run these commands after starting M3TAL:

```bash
# 1. Check all networks exist
docker network ls | grep -E "(m3tal|proxy|api_internal)"

# 2. Verify Traefik is on proxy network
docker inspect traefik | grep -A5 '"proxy"'

# 3. Test dashboard can reach API
docker exec -it m3tal-dashboard curl -s http://api:8090/health

# 4. Test routing works
curl http://localhost
```

---

## 📝 Network Security Notes

- The `m3tal` network is isolated from `proxy` - internal services cannot be accessed externally
- The API server (8090) is only accessible from within the `m3tal` network
- Traefik (8080) is exposed to the host but not the internet unless configured

---

## 🎯 Objective

Ensure **ALL externally routed services** are reachable by Traefik via a **single shared Docker network (`proxy`)**, eliminating:

* `unable to find IP address`
* `unable to find the IP address`
* `server is ignored`
* Traefik 404 errors

---

## 🧠 Root Cause

Docker Compose creates **default networks per stack**:

* `media_default`
* `control-plane_default`
* `app_default`

Even when `proxy` is added, services remain attached to these defaults, causing:

* Traefik selecting wrong network OR
* No usable IP on `proxy`

---

## 🟢 Phase 1 — Define Global Proxy Network

### Create (or verify) external proxy network

```bash
docker network create proxy
```

---

## 🔵 Phase 2 — Enforce Proxy Network in ALL Compose Files

### 🔴 REQUIRED: Add to EVERY compose file

```yaml
networks:
  proxy:
    external: true
```

---

### 🔴 For EVERY routed service (Radarr, Sonarr, etc.)

```yaml
services:
  radarr:
    networks:
      - proxy
```

---

### 🔴 Add REQUIRED Traefik label

```yaml
labels:
  - traefik.enable=true
  - traefik.docker.network=proxy
  - traefik.http.services.radarr.loadbalancer.server.port=7878
```

---

## 🟡 Phase 3 — Remove Default Network Leakage

### ❌ Problem

Docker auto-attaches `default` network

---

### ✅ Solution (explicit override)

Add at bottom of compose:

```yaml
networks:
  default:
    name: none
```

OR ensure ONLY `proxy` is referenced.

---

## 🔴 Phase 4 — Internal Services Isolation

For DB/internal-only services:

```yaml
networks:
  app_internal:
    driver: bridge
```

```yaml
services:
  db:
    networks:
      - app_internal
```

---

### 🔥 Rule

| Service Type     | Networks             |
| ---------------- | -------------------- |
| Public (Traefik) | proxy                |
| Internal only    | app_internal         |
| Hybrid (API)     | proxy + app_internal |

---

## 🟢 Phase 5 — Rebuild Infrastructure

```bash
docker compose down
docker compose up -d
```

---

## 🧪 Phase 6 — Verification

---

### ✅ Network Validation

```bash
docker inspect radarr | grep -A5 Networks
```

**Expected:**

```text
"proxy": {
  "IPAddress": "..."
}
```

---

* `media_default`
* `control-plane_default`

---

### ✅ Traefik Log Cleanliness

```bash
docker logs traefik
```

**No more:**

```text
unable to find the IP address
server is ignored
```

---

### ✅ Routing Truth Test

```bash
curl -H "Host: radarr.${DOMAIN}" http://localhost
```

**Expected:**

* HTML response OR redirect
* NOT `404 page not found`

---

## 🧱 Phase 7 — Optional (Recommended Hardening)

---

### Add Audit Rule (future improvement)

In `audit.py`:

* FAIL if service:

  * is NOT on proxy
  * OR has NO IP on proxy

---

### Add Check

```python
if "proxy" not in container_networks:
    CRITICAL
```

---

## 💥 Final Architecture

```text
[ Cloudflare ]
        ↓
[ cloudflared ]
        ↓
[ Traefik ]  ← proxy network
        ↓
[ ALL SERVICES ] ← proxy network
```

---

## 🧠 One-Line Rule

> If Traefik cannot see the container on `proxy`, it does not exist.

---

## ✅ Definition of Done

* All routed containers attached to `proxy`
* No Traefik IP errors
* `curl Host` test passes
* Domain resolves externally
