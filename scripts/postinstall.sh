#!/bin/bash
set -e

if [ "$1" = "triggered" ]; then
    echo "[m3tal] Trigger received for plugins ($2)..."
    if command -v systemctl >/dev/null 2>&1; then
        systemctl try-restart m3tal.service >/dev/null 2>&1 || true
        systemctl try-restart m3tal-api.service >/dev/null 2>&1 || true
    fi
    exit 0
fi

echo "[m3tal] Running post-installation setup..."

# 1. Create required directories
mkdir -p /etc/m3tal
mkdir -p /var/lib/m3tal
mkdir -p /var/log/m3tal

# 2. Config Initialization (do not overwrite existing config)
if [ ! -f /etc/m3tal/.env ]; then
    if [ -f /etc/m3tal/.env.example ]; then
        echo "[m3tal] Initializing default configuration at /etc/m3tal/.env"
        cp /etc/m3tal/.env.example /etc/m3tal/.env
        chmod 600 /etc/m3tal/.env
    else
        echo "⚠️  /etc/m3tal/.env.example not found, skipping config init"
    fi
fi

# 3. Permissions & Group Management
echo "[m3tal] Hardening permissions for sudo-less operation..."
if ! getent group m3tal >/dev/null; then
    groupadd -r m3tal
fi

# Ensure all core directories are owned by the group
chown -R root:m3tal /etc/m3tal /var/lib/m3tal /var/log/m3tal 2>/dev/null || true
chmod -R g+rwX /etc/m3tal /var/lib/m3tal /var/log/m3tal 2>/dev/null || true

# 4. Systemd Service Integration
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload >/dev/null 2>&1 || true

    # Clean up legacy duplicate m3tal.service if previously installed
    if systemctl is-active --quiet m3tal.service 2>/dev/null; then
        echo "[m3tal] Stopping legacy m3tal.service..."
        systemctl stop m3tal.service >/dev/null 2>&1 || true
    fi
    systemctl disable m3tal.service >/dev/null 2>&1 || true

    # Enable and restart primary API daemon
    if [ -f /lib/systemd/system/m3tal-api.service ]; then
        systemctl enable m3tal-api.service >/dev/null 2>&1 || true
        systemctl restart m3tal-api.service >/dev/null 2>&1 || true
        echo "[m3tal] M3TAL API service enabled and started"
    fi
fi

# 5. Desktop & Icon Cache Update
if command -v update-desktop-database >/dev/null 2>&1; then
    echo "[m3tal] Updating desktop database..."
    update-desktop-database -q /usr/share/applications || true
fi

if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    echo "[m3tal] Updating GTK icon cache..."
    gtk-update-icon-cache -f -t /usr/share/icons/hicolor >/dev/null 2>&1 || true
fi

# Detect primary IP
LAN_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
[ -z "$LAN_IP" ] && LAN_IP="127.0.0.1"

echo "[m3tal] ✅ Installation complete."
echo "[m3tal] 🌐 Headless Web GUI server is enabled and running!"
echo "[m3tal]   Local:    http://localhost:5050/gui/"
echo "[m3tal]   Network:  http://${LAN_IP}:5050/gui/"
echo ""
echo "[m3tal] Management:"
echo "  • Status:        sudo systemctl status m3tal-api"
echo "  • Restart:       sudo systemctl restart m3tal-api"
echo "  • CLI Headless:  m3tal headless [port]"
echo "  • CLI Stats:     m3tal stats"
echo "  • Change Port:   Edit API_PORT in /etc/m3tal/.env"