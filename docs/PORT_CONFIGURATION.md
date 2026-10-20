# Port Configuration Guide

This guide explains how M3TAL uses ports, how to resolve conflicts, and how to configure custom port mappings.

## Default Port Assignments

| Port | Service | Protocol | Status | Description |
|------|---------|----------|--------|-------------|
| `80` | Traefik HTTP | TCP | Required | HTTP entrypoint for all services |
| `443` | Traefik HTTPS | TCP | Required | HTTPS entrypoint (Cloudflare tunnel) |
| `8080` | Traefik Dashboard | TCP | Required | Traefik web UI (internal only) |
| `8082` | M3TAL Dashboard | TCP | Required | User web interface |
| `8090` | Local API | TCP | Internal | Go API server (internal network only) |
| `9000` | Docker Socket | TCP | Optional | Docker API (remote management) |

## Port Conflict Resolution

### Step 1: Identify the Conflict

Run the following command to check which services are using the required ports:

```bash
# Linux/macOS
lsof -i :80
lsof -i :443
lsof -i :8080

# Windows PowerShell
netstat -ano | findstr ":80"
netstat -ano | findstr ":443"
netstat -ano | findstr ":8080"
```

### Step 2: Options for Resolution

#### Option A: Stop the Conflicting Service

Identify the process and stop it:

```bash
# Find the PID from lsof or netstat output
sudo kill -9 <PID>

# Or find by process name
ps aux | grep <process_name>
```

#### Option B: Reconfigure M3TAL Ports

Edit your `.env` file to use different ports:

```bash
# In .env, change:
DASHBOARD_PORT=8083   # Instead of 8082
HTTP_PORT=8081        # Instead of 80 (if not root)
```

**Note**: Changing `HTTP_PORT` from 80 requires Traefik to run without root privileges.

#### Option C: Use Docker Network Proxy

If you cannot free port 80, consider using a reverse proxy like Nginx or Traefik in front of M3TAL.

## Port 80 Special Considerations

Port 80 is a **privileged port** on Linux/macOS (ports < 1024). This means:

- **Linux/macOS**: Requires root/sudo to bind, OR use a non-privileged port (e.g., 8080)
- **Windows**: No privileged port restrictions
- **Docker**: You can publish a non-privileged port to container port 80

### Recommended Approaches for Port 80 Conflicts

1. **Use Docker Port Mapping**:
   ```yaml
   # In docker-compose.yml
   ports:
     - "8080:80"  # Expose container port 80 as host port 8080
   ```

2. **Configure Traefik to use non-privileged port**:
   ```bash
   # In .env
   HTTP_PORT=8080
   ```
   Then update Traefik configuration to listen on 8080.

3. **Use a Reverse Proxy**:
   ```bash
   # Nginx example
   server {
       listen 80;
       location / {
           proxy_pass http://localhost:8080;
       }
   }
   ```

## Verifying Port Configuration

After configuring ports, verify everything is working:

```bash
# Check if ports are listening
ss -tlnp | grep -E ':(80|443|8080|8082|8090|8091)'

# Or use netstat (older systems)
netstat -tuln | grep -E ':(80|443|8080|8082|8090|8091)'
```

## Troubleshooting

### "Port already in use" Error

1. **Check what's using the port**:
   ```bash
   lsof -i :80
   ```

2. **Stop the conflicting process**:
   ```bash
   sudo kill -9 <PID>
   ```

3. **Restart M3TAL**:
   ```bash
   ./m3tal down
   ./m3tal up
   ```

### Services Not Accessible

1. **Check firewall settings**:
   ```bash
   sudo ufw status
   sudo ufw allow 80/tcp
   sudo ufw allow 443/tcp
   ```

2. **Verify Docker container is running**:
   ```bash
   docker ps
   docker logs <container_name>
   ```

3. **Check Traefik logs**:
   ```bash
   docker logs traefik
   ```

## Platform-Specific Notes

### Linux (Ubuntu/Debian)
- Ports < 1024 require root or CAP_NET_BIND_SERVICE capability
- Use `sudo usermod -aG docker $USER` to avoid sudo for Docker

### macOS
- Similar to Linux, ports < 1024 are privileged
- Docker Desktop handles port mapping transparently

### Windows
- No privileged port restrictions
- Docker Desktop is the recommended approach

### Docker Compose
- All services are on the same Docker network by default
- Internal communication uses service names (not ports)
- Only Traefik ports need to be exposed to the host

## Quick Port Test

After starting M3TAL, test all ports are accessible:

```bash
#!/bin/bash
echo "Testing M3TAL ports..."

ports=(80 443 8080 8082 8090)

for port in "${ports[@]}"; do
    if nc -z localhost $port 2>/dev/null; then
        echo "✓ Port $port is open"
    else
        echo "✗ Port $port is closed"
    fi
done