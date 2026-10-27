#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
cd "$ROOT_DIR"

if ! command -v go >/dev/null 2>&1; then
    echo "Go is required. Install the toolchain declared in go.mod, then rerun ./build.sh." >&2
    exit 1
fi

go mod download
./scripts/build-go.sh native
echo "Build complete. Binaries: ./m3tal, ./m3tal-api (native platform)"
