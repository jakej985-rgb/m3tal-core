#!/bin/bash

# M3TAL Platform Setup Script
# Verifies environment and prepares standardized storage layout.

set -e

echo "🚀 Starting M3TAL Platform Setup..."

# 1. Dependency Checks
command -v docker >/dev/null 2>&1 || { echo >&2 "❌ Docker is required but not installed. Aborting."; exit 1; }
command -v go >/dev/null 2>&1 || { echo >&2 "❌ Go is required but not installed. Aborting."; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo >&2 "❌ Python3 is required but not installed. Aborting."; exit 1; }

# 2. Standardized Storage Setup
echo "🏗️ Setting up standardized storage paths..."
# Read from .env if exists, else defaults
if [ -f ".env" ]; then
    source .env
fi

MEDIA_PATH=${MEDIA_PATH:-"./data/media"}
CONFIG_PATH=${CONFIG_PATH:-"./data/config"}
DOWNLOADS_PATH=${DOWNLOADS_PATH:-"./data/downloads"}

STORAGE_PATHS=("$MEDIA_PATH" "$CONFIG_PATH" "$DOWNLOADS_PATH")

for path in "${STORAGE_PATHS[@]}"; do
    if [ ! -d "$path" ]; then
        echo "Creating $path..."
        mkdir -p "$path"
    else
        echo "✅ $path already exists."
    fi
done

# 3. Environment Configuration
if [ ! -f ".env" ]; then
    echo "📄 Creating .env from .env.example..."
    cp .env.example .env
    echo "⚠️ Please edit .env and set your secrets before running './m3tal up'."
else
    echo "✅ .env already exists."
fi

# 4. State Directory
mkdir -p ./state

# 5. Build Control Plane
echo "🔨 Building M3TAL Control Plane..."
go build -o m3tal ./cmd/m3tal
chmod +x m3tal

echo "✅ Setup complete!"
echo "Run './m3tal up' to launch the ecosystem."
