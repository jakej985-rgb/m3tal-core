# ⬆️ Upgrading M3TAL Core

M3TAL Core is distributed as a system-wide Debian package (`.deb`). This guide covers upgrades and configuration migration.

## 📁 Key Paths

| Component | Path |
| :--- | :--- |
| **Binary** | `/usr/bin/m3tal-api` |
| **Config** | `/etc/m3tal/.env` |
| **State (API keys)** | `~/.config/m3tal/api_keys.json`, `/var/lib/m3tal/state.db`, or `./data/state.db` depending on runtime mode |
| **GUI assets** | embedded in `/usr/bin/m3tal-api` |
| **Systemd unit** | `/lib/systemd/system/m3tal-api.service` |
| **Plugins** | `./plugins`, `~/.config/m3tal/plugins`, `/var/lib/m3tal/plugins`, `/usr/lib/m3tal/plugins`, `/etc/m3tal/plugins` |

## 🚀 Upgrade Steps

### 1. Back up configuration

```bash
sudo cp /etc/m3tal/.env /etc/m3tal/.env.bak
```

### 2. Install the new package

```bash
sudo apt update && sudo apt install -y --only-upgrade m3tal
```

or install the `.deb` directly:

```bash
sudo dpkg -i m3tal_<version>_amd64.deb
sudo apt-get install -f   # resolve dependencies if needed
```

The package postinst preserves an existing `/etc/m3tal/.env`; it only seeds `.env.example` when no `.env` exists.

### 3. Restart the service

```bash
sudo systemctl restart m3tal-api
journalctl -u m3tal-api -f
```

## 🔄 Migrating from older versions

* The legacy `m3tal` CLI, TUI, tray, Docker/Compose/Traefik/VPN orchestration, and agent/daemon code have been **removed from Core**. Those features now live in external plugins under [m3tal-plugins](https://github.com/jakej985-rgb/m3tal-plugins).
* Legacy `/etc/m3tal/.env` keys for removed features (VPN credentials, Traefik ports, storage paths, etc.) are ignored by Core. Plugin-specific variables should be configured by the corresponding plugins.
* Existing `state.db` files keep working; unused legacy tables are left in place but no longer written to.
* API keys are preserved across upgrades.
* The APT repository previously published a separate `1.x` package lineage. The current package stream is `0.x`; because Debian version ordering treats `0.x` as older than `1.x`, APT will not automatically replace a `1.x` installation with `0.x`. If you have an old `1.x` package installed, remove it explicitly before installing the current `m3tal` package.

## ❓ FAQ

**Q: Where did `m3tal ps`, `m3tal compose`, `m3tal vpn`, etc. go?**
A: They were removed with the CLI. Equivalent functionality ships as external plugins (`m3tal-docker`, `m3tal-compose`, `m3tal-vpn`, …) managed from the Web GUI's Plugin Manager.

**Q: Do I need Docker to run Core?**
A: No. Core has no container runtime dependency. Docker is only needed by plugins that manage containers.
