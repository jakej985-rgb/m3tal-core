# 🐳 M3TAL Stack (Infrastructure)

This repository serves as the **Single Source of Truth** for all Docker Compose configurations in the M3TAL ecosystem.

## 📁 Structure

- `network-compose.yml`: Defines the `proxy` bridge network and core routing.
- `routing-compose.yml`: Configures the Traefik Reverse Proxy and SSL termination.
- `m3tal-compose.yml`: Deploys the core M3TAL components (Backend, Dashboard, Registry).

## 🚀 Usage

These stacks are typically managed via the root `m3tal.py` orchestrator:

```bash
# From the root directory
python m3tal.py up
```

## 🌐 Network Standards

All services in the M3TAL ecosystem must join the **`proxy`** network to enable:
1. Inter-service communication.
2. Traefik automatic discovery.
3. IP isolation from the host.

## 🛠 Adding Services

To add a new service to the stack:
1. Create a `docker/` directory.
2. Add your `*-compose.yml` file.
3. Ensure the service uses the following labels for Traefik:
   ```yaml
   labels:
     - "traefik.enable=true"
     - "traefik.http.routers.myservice.rule=Host(`myservice.local`)"
   ```
