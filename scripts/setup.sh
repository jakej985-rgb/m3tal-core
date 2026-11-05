#!/bin/bash

# M3TAL Core Setup Script
# Prepares the environment for local development of the Web GUI dashboard.

set -e

echo "🚀 Starting M3TAL Core setup..."

# 1. Dependency checks
command -v go >/dev/null 2>&1 || { echo >&2 "❌ Go is required but not installed. Aborting."; exit 1; }
command -v node >/dev/null 2>&1 || { echo >&2 "❌ Node.js is required for the Web GUI. Aborting."; exit 1; }

# 2. Environment configuration
if [ ! -f ".env" ]; then
    echo "📄 Creating .env from .env.example..."
    cp .env.example .env
    echo "⚠️  Please edit .env and set API_TOKEN before running the server."
else
    echo "✅ .env already exists."
fi

# 3. State directory
mkdir -p ./data

# 4. Build API binary and GUI
echo "🔨 Building M3TAL Core..."
./scripts/build-go.sh
(cd gui && npm ci && npm run build)

echo "✅ Setup complete!"
echo "Run './m3tal-api' and open http://localhost:5050/gui/"
