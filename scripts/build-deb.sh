#!/bin/bash
set -e

# 1. Build Web GUI
echo "🎨 Building Web GUI..."
if [ -d "gui" ]; then
    (cd gui && npm run build)
fi

# 2. Get version
VERSION=$(cat VERSION 2>/dev/null | sed 's/^v//' || echo "0.1.0")
export VERSION

# 3. Compile binaries
echo "🚀 Building M3TAL binaries (v${VERSION})..."
go build -ldflags "-X main.Version=${VERSION}" -o m3tal ./cli
go build -ldflags "-X main.Version=${VERSION}" -o m3tal-api ./api/cmd

# 4. Package into .deb
DEB_NAME="m3tal_${VERSION}_amd64.deb"

if command -v nfpm >/dev/null 2>&1; then
    echo "📦 Packaging with nfpm..."
    nfpm pkg --packager deb --target "$DEB_NAME" -f packaging/nfpm.yaml
elif command -v dpkg-deb >/dev/null 2>&1; then
    echo "📦 Packaging with dpkg-deb..."
    STAGE_DIR=$(mktemp -d /tmp/m3tal-deb-stage.XXXXXX)
    trap 'rm -rf "$STAGE_DIR"' EXIT

    mkdir -p "$STAGE_DIR/DEBIAN"
    mkdir -p "$STAGE_DIR/usr/bin"
    mkdir -p "$STAGE_DIR/usr/share/m3tal/stack"
    mkdir -p "$STAGE_DIR/usr/share/m3tal/gui"
    mkdir -p "$STAGE_DIR/etc/m3tal"
    mkdir -p "$STAGE_DIR/lib/systemd/system"
    mkdir -p "$STAGE_DIR/usr/share/applications"
    mkdir -p "$STAGE_DIR/usr/share/pixmaps"
    mkdir -p "$STAGE_DIR/usr/share/icons/hicolor/scalable/apps"

    # Copy binaries & create headless alias
    cp ./m3tal "$STAGE_DIR/usr/bin/"
    cp ./m3tal-api "$STAGE_DIR/usr/bin/"
    chmod 755 "$STAGE_DIR/usr/bin/m3tal" "$STAGE_DIR/usr/bin/m3tal-api"
    ln -sf m3tal "$STAGE_DIR/usr/bin/m3tal-headless"

    # Copy stacks & gui
    if [ -d "deploy/stack" ]; then
        cp -r deploy/stack/* "$STAGE_DIR/usr/share/m3tal/stack/"
    fi
    if [ -d "gui/dist" ]; then
        cp -r gui/dist/* "$STAGE_DIR/usr/share/m3tal/gui/"
    fi

    # Config & systemd
    cp ./.env.example "$STAGE_DIR/etc/m3tal/.env.example"
    cp ./packaging/m3tal-api.service "$STAGE_DIR/lib/systemd/system/"
    cp ./packaging/m3tal.service "$STAGE_DIR/lib/systemd/system/"
    cp ./packaging/m3tal.desktop "$STAGE_DIR/usr/share/applications/"

    # Icons
    if [ -f "docs/goback_logo.svg" ]; then
        cp docs/goback_logo.svg "$STAGE_DIR/usr/share/icons/hicolor/scalable/apps/m3tal.svg"
        cp docs/goback_logo.svg "$STAGE_DIR/usr/share/pixmaps/m3tal.svg"
    fi
    if [ -d "cli/m3tal_icon_pack" ]; then
        for size in 16 24 32 48 64 128 256; do
            icon_file="cli/m3tal_icon_pack/m3tal_icon_${size}x${size}.png"
            if [ -f "$icon_file" ]; then
                mkdir -p "$STAGE_DIR/usr/share/icons/hicolor/${size}x${size}/apps"
                cp "$icon_file" "$STAGE_DIR/usr/share/icons/hicolor/${size}x${size}/apps/m3tal.png"
            fi
        done
    fi

    # Control file
    cat > "$STAGE_DIR/DEBIAN/control" <<EOF
Package: m3tal
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: amd64
Maintainer: Jake Johnson <jake@m3tal.io>
Depends: systemd
Recommends: xdotool
Description: M3TAL Core — DEB-first infrastructure orchestration platform.
 Includes the m3tal CLI and control-plane API with embedded Web GUI.
 Optimized for home servers and homelab automation.
EOF

    # Maintainer scripts
    cp ./scripts/postinstall.sh "$STAGE_DIR/DEBIAN/postinst"
    chmod 755 "$STAGE_DIR/DEBIAN/postinst"
    cp ./scripts/postremove.sh "$STAGE_DIR/DEBIAN/postrm"
    chmod 755 "$STAGE_DIR/DEBIAN/postrm"

    dpkg-deb --build --root-owner-group "$STAGE_DIR" "$DEB_NAME"
else
    echo "❌ Neither nfpm nor dpkg-deb was found."
    exit 1
fi

echo "✅ Package built: $DEB_NAME"
