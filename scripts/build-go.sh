#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT_DIR"

VERSION=${VERSION:-$(tr -d '[:space:]' < VERSION | sed 's/^v//')}
if [[ -z "$VERSION" ]]; then
    echo "VERSION is empty" >&2
    exit 1
fi

# Go AMD64 v1 is the x86-64 baseline supported by the i5-750. Disable cgo so
# no host-selected C compiler flags can raise the CPU requirement.
export GOOS=linux
export GOARCH=amd64
export GOAMD64=v1
export CGO_ENABLED=0

go build -trimpath -ldflags "-X main.Version=${VERSION}" -o m3tal ./cli
go build -trimpath -ldflags "-X main.Version=${VERSION}" -o m3tal-api ./api/cmd

chmod 755 m3tal m3tal-api

for binary in m3tal m3tal-api; do
    metadata=$(go version -m "$binary")
    grep -Eq '^[[:space:]]+build[[:space:]]+CGO_ENABLED=0$' <<< "$metadata"
    grep -Eq '^[[:space:]]+build[[:space:]]+GOAMD64=v1$' <<< "$metadata"
    if nm --defined-only "$binary" | awk '$NF == "x_cgo_getstackbound" { found = 1 } END { exit !found }'; then
        echo "$binary unexpectedly contains the cgo stack-bound helper" >&2
        exit 1
    fi
done
