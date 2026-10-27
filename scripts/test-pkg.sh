#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
DEB_FILE=${1:-}

if [[ -z "$DEB_FILE" ]]; then
    shopt -s nullglob
    candidates=("$ROOT_DIR"/m3tal_*.deb)
    if (( ${#candidates[@]} != 1 )); then
        echo "Pass the exact .deb path (found ${#candidates[@]} matching artifacts)." >&2
        exit 1
    fi
    DEB_FILE=${candidates[0]}
elif [[ "$DEB_FILE" != /* ]]; then
    DEB_FILE="$ROOT_DIR/$DEB_FILE"
fi

[[ -s "$DEB_FILE" ]] || { echo "Package not found: $DEB_FILE" >&2; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "Docker is required to run package tests." >&2; exit 1; }

echo "Testing Debian package: $DEB_FILE"
docker run --rm \
    -v "$DEB_FILE:/tmp/m3tal.deb:ro" \
    debian:bookworm-slim bash -euc '
        export DEBIAN_FRONTEND=noninteractive
        apt-get update -qq
        apt-get install -y -qq --no-install-recommends curl qemu-user >/dev/null

        expected=(
            /usr/bin/m3tal
            /usr/bin/m3tal-api
            /usr/bin/m3tal-core
            /usr/bin/m3tal-gui
            /usr/bin/m3tal-headless
            /usr/bin/m3tal-web
            /etc/m3tal/.env.example
            /etc/m3tal/config.yaml
            /etc/m3tal/VERSION
            /usr/share/m3tal/VERSION
            /lib/systemd/system/m3tal-api.service
            /usr/share/applications/m3tal.desktop
            /usr/share/m3tal/gui/index.html
            /usr/share/icons/hicolor/scalable/apps/m3tal.svg
            /usr/share/pixmaps/m3tal.svg
            /usr/share/icons/hicolor/256x256/apps/m3tal.png
        )

        echo "[test] Fresh install and package contents"
        apt-get install -y -qq /tmp/m3tal.deb >/dev/null
        for path in "${expected[@]}"; do
            test -e "$path" || { echo "Missing package path: $path" >&2; exit 1; }
        done
        test -x /usr/bin/m3tal
        test -x /usr/bin/m3tal-api
        grep -Fq "StartLimitIntervalSec=60" /lib/systemd/system/m3tal-api.service
        grep -Fq "StartLimitBurst=5" /lib/systemd/system/m3tal-api.service
        m3tal --help >/dev/null
        test "$(dpkg-deb -f /tmp/m3tal.deb Package)" = m3tal
        test "$(dpkg-deb -f /tmp/m3tal.deb Architecture)" = amd64

        echo "[test] API startup with an AVX-less Nehalem CPU model"
        API_PORT=15050 API_TOKEN=cpu-compat-test \
            qemu-x86_64 -cpu Nehalem /usr/bin/m3tal-api >/tmp/m3tal-api.log 2>&1 &
        api_pid=$!
        cleanup() {
            kill "$api_pid" 2>/dev/null || true
            wait "$api_pid" 2>/dev/null || true
        }
        trap cleanup EXIT
        healthy=false
        for _ in $(seq 1 40); do
            if curl --silent --show-error --fail \
                -H "Authorization: Bearer cpu-compat-test" \
                http://127.0.0.1:15050/api/health >/tmp/health.json 2>/dev/null; then
                healthy=true
                break
            fi
            if ! kill -0 "$api_pid" 2>/dev/null; then
                cat /tmp/m3tal-api.log >&2
                exit 1
            fi
            sleep 0.5
        done
        if [[ "$healthy" != true ]]; then
            cat /tmp/m3tal-api.log >&2
            echo "API health endpoint did not become ready" >&2
            exit 1
        fi
        cleanup
        trap - EXIT

        echo "[test] Upgrade preserves existing /etc/m3tal/.env"
        current_version=$(dpkg-deb -f /tmp/m3tal.deb Version)
        mkdir -p /tmp/old
        dpkg-deb -R /tmp/m3tal.deb /tmp/old
        sed -i "s/^Version: .*/Version: 0.0.1/" /tmp/old/DEBIAN/control
        dpkg-deb --build --root-owner-group /tmp/old /tmp/m3tal-old.deb >/dev/null
        dpkg -i /tmp/m3tal-old.deb >/dev/null
        printf "PRESERVED_BY_UPGRADE=yes\n" >/etc/m3tal/.env
        dpkg -i /tmp/m3tal.deb >/dev/null
        grep -Fxq "PRESERVED_BY_UPGRADE=yes" /etc/m3tal/.env
        test "$(dpkg-query -W -f="\${Version}" m3tal)" = "$current_version"
        test -d /var/lib/m3tal
        echo "Package install, API startup, and upgrade tests passed."
    '
