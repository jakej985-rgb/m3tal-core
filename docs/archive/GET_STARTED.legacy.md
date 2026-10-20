# 📗 Getting Started with M3TAL

Welcome to **M3TAL Media Server**! This guide will help you get your autonomous media server running in 10 minutes, even if you are new to Linux or Docker.

---

## 🏗️ What is M3TAL?

Think of M3TAL as your server's **robot brain**. Instead of you checking every day if your movies are downloading or if a container has crashed, M3TAL's "Agents" watch the server for you 24/7 and fix problems automatically.

---

## ⚠️ Pre-flight Checklist

Before you begin, please verify the following to ensure a successful installation:

### ✅ System Requirements
- **OS**: Linux (Ubuntu 22.04+ recommended) or macOS
- **Docker**: Docker Engine 20.10+ or Docker Desktop
- **Memory**: At least 4GB RAM (8GB recommended)
- **Disk Space**: 20GB+ available for data and configurations

### ✅ Port Availability

M3TAL requires the following ports to be **free** on your host:

| Port | Service | Purpose |
|------|---------|---------|
| `80` | Traefik | HTTP entrypoint (required) |
| `443` | Traefik | HTTPS entrypoint (optional, for Cloudflare tunnel) |
| `8080` | Traefik Dashboard | Internal networking |
| `8082` | M3TAL Dashboard | User web interface |

**Check if ports are available** (Linux/macOS):
```bash
# Check port 80
lsof -i :80

# Check port 443
lsof -i :443

# Check port 8080
lsof -i :8080
```

**If a port is in use**, either stop the conflicting service or configure M3TAL to use different ports via `m3tal config wizard`.

### ✅ Path Requirements

**`/mnt` Directory**: M3TAL expects a data directory at `/mnt` by default. If you're on macOS or Windows, or want to use a different path:

1. Set `BASE_STORAGE_PATH` in your `.env` file
2. Ensure the directory exists and is writable:
   ```bash
   mkdir -p /mnt/media
   chmod 755 /mnt
   ```

### ✅ Docker Permissions

M3TAL requires Docker API access. Ensure your user can run Docker commands:

```bash
# Test Docker access
docker ps

# If this requires sudo, add your user to the docker group:
sudo usermod -aG docker $USER
# Then log out and back in for changes to take effect
```

For more information, see [Environment Variables](ENVIRONMENT_VARIABLES.md) or [Port Configuration](PORT_CONFIGURATION.md).

---

## 🛠️ Step 1: Preparation

You will need a Linux server (Ubuntu is recommended) and basic command line access (SSH).

### 1. Update your system

Before starting, ensure your server is up to date:

```bash
sudo apt update && sudo apt upgrade -y
```

### 2. Download M3TAL

Clone the repository to your server:

```bash
git clone https://github.com/jakej985-rgb/M3tal-Media-Server.git
cd M3tal-Media-Server
```

---

## 🚀 Step 2: Installation

M3TAL comes with an **Interactive Setup Wizard** that installs everything for you (Docker, Python, and all agents).

1. **Run the installer**:

   ```bash
   chmod +x install.sh
   ./install.sh
   ```

2. **Follow the Prompts**:

   * **Data Directory**: Typically `/mnt`. This is where your movies and shows will live.
   * **Auto-Install**: Type `y` to let M3TAL install Docker for you.
   * **Auto-Start**: Type `y` to launch the dashboard immediately.

---

## 🖥️ Step 3: The Dashboard

Once installed, M3TAL provides a web interface to see what's happening.

1. **Open your browser** and go to `http://YOUR_SERVER_IP:8080`.

2. **Log in** with the admin credentials you created during setup:

   * **Username**: `admin`
   * **Password**: the admin password you chose during the interactive setup

3. **Recover or rotate your password if needed**: Run `python scripts/config/manage_users.py --reset-admin` from an interactive terminal.

---

## 办 Step 4: Adding Your Services

M3TAL looks inside the `docker/` folder for your apps.

1. **Explore the folders**: Inside `docker/media/`, you will find a `docker-compose.yml`. This file defines apps like Radarr (Movies) and Sonarr (TV).

2. **Customizing**: You can edit these files to add your own apps. M3TAL will automatically detect them and start monitoring them within 60 seconds.

---

## 🚑 Step 5: Self-Healing in Action

You don't need to do anything! If a container crashes:

1. The **Monitor** will see it's offline.
2. The **Decision Engine** will plan a restart.
3. The **Reconciler** will bring it back to life.
4. You can check the **Logs** tab in the dashboard to see exactly when and why M3TAL fixed the issue.

---

## 💾 Step 6: Backups

To keep your configuration safe, run the backup script once a week:

```bash
bash scripts/maintenance/backup.sh
```

This saves your settings to `/mnt/backups`. If anything breaks, you can use `bash scripts/maintenance/restore.sh` to get everything back.

---

## 💡 Pro Tips for Beginners

* **Paths**: Always use `/mnt` for your hard drives. M3TAL expects this structure to keep your media organized.
* **Logs**: If something isn't working, check `control-plane/state/logs/`. Every agent leaves a detailed trail of what it's thinking.
* **Resources**: Use the Dashboard's "Metrics" tab to see which app is using too much RAM or CPU.

---

**Welcome to the future of homelabbing!** 🚀

If you have questions, check the [Full Documentation](docs/) or open an issue on GitHub.
