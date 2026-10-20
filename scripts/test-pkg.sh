#!/bin/bash
set -e

DEB_FILE=$(ls m3tal-core_*.deb | head -n 1)

if [ -z "$DEB_FILE" ]; then
    echo "❌ No .deb file found. Run build-deb.sh first."
    exit 1
fi

echo "🧪 Testing $DEB_FILE in clean Docker container..."

# Run a temporary container
docker run --rm -v "$(pwd)/$DEB_FILE:/tmp/pkg.deb" debian:bookworm-slim bash -c "
    set -e
    echo '[test] Installing dependencies...'
    apt-get update >/dev/null
    apt-get install -y ./tmp/pkg.deb >/dev/null
    
    echo '[test] Verifying binaries...'
    which m3tal
    which m3tal-api
    m3tal --help >/dev/null
    
    echo '[test] Verifying directories...'
    ls -d /etc/m3tal
    ls -d /opt/m3tal/stack
    ls -d /var/lib/m3tal
    
    echo '[test] Verifying stack manifests...'
    ls /opt/m3tal/stack/m3tal-compose.yml
    
    echo '✅ Fresh install test PASSED'
"

echo ""
echo "🧪 Testing configuration preservation..."

docker run --rm -v "$(pwd)/$DEB_FILE:/tmp/pkg.deb" debian:bookworm-slim bash -c "
    set -e
    mkdir -p /etc/m3tal
    echo 'PRESERVED=true' > /etc/m3tal/.env
    
    apt-get update >/dev/null
    apt-get install -y ./tmp/pkg.deb >/dev/null
    
    grep 'PRESERVED=true' /etc/m3tal/.env >/dev/null
    echo '✅ Configuration preservation test PASSED'
"

echo ""
echo "🎉 All package tests PASSED!"
