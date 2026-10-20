#!/bin/bash
set -e

echo "[m3tal] Removing M3TAL Core..."

# 1. Stop and disable services
if command -v systemctl >/dev/null 2>&1; then
    if [ -f /lib/systemd/system/m3tal-api.service ]; then
        echo "[m3tal] Stopping M3TAL API service"
        systemctl stop m3tal-api.service >/dev/null 2>&1 || true
        systemctl disable m3tal-api.service >/dev/null 2>&1 || true
    fi
    if [ -f /lib/systemd/system/m3tal.service ]; then
        echo "[m3tal] Stopping M3TAL Agents service"
        systemctl stop m3tal.service >/dev/null 2>&1 || true
        systemctl disable m3tal.service >/dev/null 2>&1 || true
    fi
    systemctl daemon-reload >/dev/null 2>&1 || true
fi

# 4. Clean up /docker symlink
if [ -L /docker ]; then
    echo "[m3tal] Removing /docker symlink"
    rm /docker
fi

# 5. Handle Purge (remove data and config)
if [ "$1" = "purge" ]; then
    echo "[m3tal] Purging M3TAL data and configuration..."

    # Define default stack files and directories to selectively clean up
    DEFAULT_FILES=(
        "README.md"
        "ai-compose.yml"
        "cloudflared-config.yml"
        "m3tal-compose.local.yml"
        "m3tal-compose.traefik.yml"
        "m3tal-compose.yml"
        "routing-compose.yml"
        "traefik.yml"
        "users.example.json"
    )
    DEFAULT_DIRS=(
        "dynamic"
        "dash"
    )

    # Clean default stack files from /opt/m3tal/stack
    if [ -d /opt/m3tal/stack ]; then
        echo "[m3tal] Cleaning default stack files from /opt/m3tal/stack..."
        for file in "${DEFAULT_FILES[@]}"; do
            rm -f "/opt/m3tal/stack/$file"
        done
        for dir in "${DEFAULT_DIRS[@]}"; do
            rm -rf "/opt/m3tal/stack/$dir"
        done
        # Delete /opt/m3tal/stack if empty, otherwise preserve custom files
        rmdir /opt/m3tal/stack 2>/dev/null || echo "[m3tal] Custom files exist in /opt/m3tal/stack; preserving them."
    fi

    # Clean default stack files from /docker if it is a directory and not a symlink
    if [ -d /docker ] && [ ! -L /docker ]; then
        echo "[m3tal] Cleaning default stack files from /docker..."
        for file in "${DEFAULT_FILES[@]}"; do
            rm -f "/docker/$file"
        done
        for dir in "${DEFAULT_DIRS[@]}"; do
            rm -rf "/docker/$dir"
        done
        # Delete /docker if empty, otherwise preserve custom files
        rmdir /docker 2>/dev/null || echo "[m3tal] Custom files exist in /docker; preserving folder."
    fi

    # Clean other system configuration and data
    rm -rf /etc/m3tal
    rm -rf /var/lib/m3tal
    
    # Remove /opt/m3tal if empty (no custom files remain)
    rmdir /opt/m3tal 2>/dev/null || echo "[m3tal] Custom files/folders remain under /opt/m3tal; preserving folder."

    echo "[m3tal] Purge complete."
else
    echo "[m3tal] Removed."
    echo "[m3tal] NOTE: Configuration (/etc/m3tal) and Data (/var/lib/m3tal) have been preserved."
    echo "[m3tal] NOTE: Stack files (/opt/m3tal/stack) have also been preserved (including custom compose files)."
fi