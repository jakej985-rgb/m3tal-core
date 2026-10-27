#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT_DIR"

BUILD_PROFILE=${1:-native}
VERSION=${VERSION:-$(cat VERSION)}
VERSION=$(printf '%s' "$VERSION" | tr -d '[:space:]')
VERSION=${VERSION#v}
if [[ -z "$VERSION" ]]; then
    echo "VERSION is empty" >&2
    exit 1
fi

case "$BUILD_PROFILE" in
    native)
        # Preserve Go's normal native build behavior for local development.
        ;;
    debian)
        # Go AMD64 v1 is the x86-64 baseline supported by the i5-750. Disable
        # cgo so no host-selected C compiler flags can raise the CPU requirement.
        export GOOS=linux
        export GOARCH=amd64
        export GOAMD64=v1
        export CGO_ENABLED=0
        ;;
    *)
        echo "Unknown build profile '$BUILD_PROFILE'; use native or debian" >&2
        exit 2
        ;;
esac

go build -trimpath -ldflags "-X main.Version=${VERSION}" -o m3tal ./cli
go build -trimpath -ldflags "-X main.Version=${VERSION}" -o m3tal-api ./api/cmd

chmod 755 m3tal m3tal-api

if [[ "$BUILD_PROFILE" == debian ]]; then
    for binary in m3tal m3tal-api; do
        metadata=$(go version -m "$binary")
        grep -Eq '^[[:space:]]+build[[:space:]]+CGO_ENABLED=0$' <<< "$metadata"
        grep -Eq '^[[:space:]]+build[[:space:]]+GOAMD64=v1$' <<< "$metadata"
        if nm --defined-only "$binary" | awk '$NF == "x_cgo_getstackbound" { found = 1 } END { exit !found }'; then
            echo "$binary unexpectedly contains the cgo stack-bound helper" >&2
            exit 1
        fi
    done
fi
