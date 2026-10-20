# ⬆️ Upgrading to M3TAL Core (System Install)

M3TAL has transitioned from a repository-based execution model to a system-wide installation model (`.deb`). This guide explains how to migrate your existing installation to the new structure.

## 📁 Key Path Changes

| Component | Old Path (Repo) | New Path (System) |
| :--- | :--- | :--- |
| **Binaries** | `./m3tal` | `/usr/bin/m3tal` |
| **API** | `./m3tal-api` | `/usr/bin/m3tal-api` |
| **Config** | `./.env` | `/etc/m3tal/config.yaml` |
| **Stacks** | `./deploy/stack/` | `/opt/m3tal/stack/` |
| **Data** | `./data/` | `/var/lib/m3tal/` |

## 🚀 Migration Steps

### 1. Back up your configuration
If you have an existing `.env` file in your repository:
```bash
cp .env .env.bak
```

### 2. Install the system package
Follow the instructions in the main [README.md](../README.md) to add the repository and install `m3tal-core`.

### 3. Migrate configuration
The new system uses `/etc/m3tal/config.yaml` by default. You can run the initialization wizard to re-enter your settings, or manually map your `.env` values to the new location.

```bash
sudo m3tal init
```

### 4. Transition Services
If you were running M3TAL manually or via a custom service:
1. Stop your existing processes.
2. The `m3tal-core` package includes systemd units. Enable them:
```bash
sudo systemctl enable --now m3tal m3tal-api
```

## ❓ FAQ

**Q: Where are my Docker Compose files now?**
A: They are located in `/opt/m3tal/stack/`. You should not modify these directly; use `m3tal config` to update parameters.

**Q: Can I still run M3TAL from the repository?**
A: Yes, but it is not recommended for production. The `m3tal` CLI will look for local `.env` and `deploy/stack/` if system paths are missing.
