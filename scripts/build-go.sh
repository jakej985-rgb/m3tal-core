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
        ;;
    debian)
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

go build -trimpath -ldflags "-X main.Version=${VERSION}" -o m3tal-api ./api/cmd
chmod 755 m3tal-api

if [[ "$BUILD_PROFILE" == debian ]]; then
    metadata=$(go version -m m3tal-api)
    grep -Eq '^[[:space:]]+build[[:space:]]+CGO_ENABLED=0$' <<< "$metadata"
    grep -Eq '^[[:space:]]+build[[:space:]]+GOAMD64=v1$' <<< "$metadata"
    if nm --defined-only m3tal-api | awk '$NF == "x_cgo_getstackbound" { found = 1 } END { exit !found }'; then
        echo "m3tal-api unexpectedly contains the cgo stack-bound helper" >&2
        exit 1
    fi
fi
