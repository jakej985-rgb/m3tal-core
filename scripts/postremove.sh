#!/bin/bash
set -e

echo "[m3tal] Removing M3TAL Core..."

# 1. Stop and disable services
if command -v systemctl >/dev/null 2>&1; then
    if [ -f /lib/systemd/system/m3tal-api.service ] || systemctl is-active --quiet m3tal-api.service 2>/dev/null; then
        echo "[m3tal] Stopping M3TAL API service..."
        systemctl stop m3tal-api.service >/dev/null 2>&1 || true
        systemctl disable m3tal-api.service >/dev/null 2>&1 || true
    fi
    if systemctl is-active --quiet m3tal.service 2>/dev/null; then
        echo "[m3tal] Stopping legacy m3tal.service..."
        systemctl stop m3tal.service >/dev/null 2>&1 || true
        systemctl disable m3tal.service >/dev/null 2>&1 || true
    fi
    systemctl daemon-reload >/dev/null 2>&1 || true
fi

# 2. Clean up legacy /docker symlink if it was created as a symlink
if [ -L /docker ]; then
    rm -f /docker
fi

# 3. Handle Purge (remove data and config)
if [ "$1" = "purge" ]; then
    echo "[m3tal] Purging M3TAL Core data and configuration..."
    rm -rf /etc/m3tal
    rm -rf /var/lib/m3tal
    rm -rf /var/log/m3tal
    echo "[m3tal] Purge complete."
else
    echo "[m3tal] Removed."
    echo "[m3tal] NOTE: Configuration (/etc/m3tal) and Data (/var/lib/m3tal) have been preserved."
fi