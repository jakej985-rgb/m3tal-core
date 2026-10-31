#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT_DIR"

VERSION=${VERSION:-$(cat VERSION)}
VERSION=$(printf '%s' "$VERSION" | tr -d '[:space:]')
VERSION=${VERSION#v}
DEB_NAME="m3tal_${VERSION}_amd64.deb"
DEB_TARGET=${DEB_TARGET:-$DEB_NAME}
PACKAGER=${PACKAGER:-auto}

if [[ -z "$VERSION" ]]; then
    echo "VERSION is empty" >&2
    exit 1
fi

# Remove outputs first so a failed compile can never package stale binaries.
rm -f m3tal m3tal-api "$DEB_TARGET"

if [[ -d gui ]]; then
    (cd gui && npm ci && npm run build)
fi

VERSION="$VERSION" ./scripts/build-go.sh debian

if [[ "$PACKAGER" == "auto" ]]; then
    if command -v nfpm >/dev/null 2>&1; then
        PACKAGER=nfpm
    else
        PACKAGER=dpkg-deb
    fi
fi

case "$PACKAGER" in
    nfpm)
        command -v nfpm >/dev/null 2>&1 || { echo "nfpm is required for PACKAGER=nfpm" >&2; exit 1; }
        VERSION="$VERSION" nfpm pkg --packager deb --target "$DEB_TARGET" -f packaging/nfpm.yaml
        ;;
    dpkg-deb)
        command -v dpkg-deb >/dev/null 2>&1 || { echo "dpkg-deb is required for PACKAGER=dpkg-deb" >&2; exit 1; }
        STAGE_DIR=$(mktemp -d)
        trap 'rm -rf "$STAGE_DIR"' EXIT
        install -d "$STAGE_DIR/DEBIAN" \
            "$STAGE_DIR/usr/bin" \
            "$STAGE_DIR/usr/share/m3tal/gui" \
            "$STAGE_DIR/etc/m3tal" \
            "$STAGE_DIR/lib/systemd/system" \
            "$STAGE_DIR/usr/share/applications" \
            "$STAGE_DIR/usr/share/pixmaps" \
            "$STAGE_DIR/usr/share/icons/hicolor/scalable/apps"

        install -m 755 m3tal m3tal-api "$STAGE_DIR/usr/bin/"
        for alias in m3tal-core m3tal-gui m3tal-headless m3tal-web; do
            ln -s /usr/bin/m3tal "$STAGE_DIR/usr/bin/$alias"
        done

        if [[ -d gui/dist ]]; then
            cp -a gui/dist/. "$STAGE_DIR/usr/share/m3tal/gui/"
        fi
        install -m 644 .env.example "$STAGE_DIR/etc/m3tal/.env.example"
        install -m 644 VERSION "$STAGE_DIR/etc/m3tal/VERSION"
        install -m 644 VERSION "$STAGE_DIR/usr/share/m3tal/VERSION"
        install -m 644 packaging/m3tal-api.service "$STAGE_DIR/lib/systemd/system/m3tal-api.service"
        install -m 644 packaging/m3tal.desktop "$STAGE_DIR/usr/share/applications/m3tal.desktop"
        install -m 644 docs/goback_logo.svg "$STAGE_DIR/usr/share/icons/hicolor/scalable/apps/m3tal.svg"
        install -m 644 docs/goback_logo.svg "$STAGE_DIR/usr/share/pixmaps/m3tal.svg"
        for size in 16 24 32 48 64 128 256; do
            icon="cli/m3tal_icon_pack/m3tal_icon_${size}x${size}.png"
            if [[ -f "$icon" ]]; then
                install -D -m 644 "$icon" "$STAGE_DIR/usr/share/icons/hicolor/${size}x${size}/apps/m3tal.png"
            fi
        done

        cat > "$STAGE_DIR/DEBIAN/control" <<EOF
Package: m3tal
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: amd64
Maintainer: Jake Johnson <jake@m3tal.io>
Recommends: xdotool
Description: M3TAL Core — DEB-first infrastructure orchestration platform.
 Includes the m3tal CLI and control-plane API.
 Optimized for Traefik ecosystems and home lab automation.
EOF
        printf '%s\n' \
            /etc/m3tal/.env.example \
            /etc/m3tal/VERSION \
            /lib/systemd/system/m3tal-api.service \
            > "$STAGE_DIR/DEBIAN/conffiles"
        install -m 644 packaging/deb/DEBIAN/triggers "$STAGE_DIR/DEBIAN/triggers"
        install -m 755 scripts/postinstall.sh "$STAGE_DIR/DEBIAN/postinst"
        install -m 755 scripts/postremove.sh "$STAGE_DIR/DEBIAN/postrm"
        dpkg-deb --build --root-owner-group "$STAGE_DIR" "$DEB_TARGET"
        ;;
    *)
        echo "Unknown PACKAGER '$PACKAGER'; use auto, nfpm, or dpkg-deb" >&2
        exit 2
        ;;
esac

test -s "$DEB_TARGET"
echo "Package built: $DEB_TARGET"
