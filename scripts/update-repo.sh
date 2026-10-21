#!/bin/bash
set -e

REPO_DIR="${REPO_DIR:-repo}"
KEY_ID="${KEY_ID:-5F84FE50A40111C981410E11775AD1473BF25102}"

echo "🚀 Updating APT Repository in '$REPO_DIR'..."

# If m3tal-apt-key ingest script is present in the repo, delegate to it
if [ -f "$REPO_DIR/scripts/ingest_deb.py" ]; then
    echo "📦 Using central ingest_deb.py from $REPO_DIR..."
    for deb in *.deb; do
        if [ -f "$deb" ]; then
            python3 "$REPO_DIR/scripts/ingest_deb.py" "$deb" --allow-replace
        fi
    done
    exit 0
fi

# 1. Setup structured pool and dists structure
mkdir -p "$REPO_DIR/pool/main/m3tal"
mkdir -p "$REPO_DIR/dists/stable/main/binary-amd64"

# 2. Copy .deb files to pool
cp *.deb "$REPO_DIR/pool/main/m3tal/" 2>/dev/null || true

# 3. Generate Packages file
echo "[repo] Generating Packages..."
cd "$REPO_DIR"
apt-ftparchive packages pool/main > dists/stable/main/binary-amd64/Packages
gzip -c dists/stable/main/binary-amd64/Packages > dists/stable/main/binary-amd64/Packages.gz

# 4. Generate Release file
echo "[repo] Generating Release..."
apt-ftparchive \
    -o "APT::FTPArchive::Release::Origin=M3TAL" \
    -o "APT::FTPArchive::Release::Label=M3TAL" \
    -o "APT::FTPArchive::Release::Suite=stable" \
    -o "APT::FTPArchive::Release::Codename=stable" \
    -o "APT::FTPArchive::Release::Components=main" \
    -o "APT::FTPArchive::Release::Architectures=amd64" \
    -o "APT::FTPArchive::Release::Description=M3TAL Core Repository" \
    release dists/stable > dists/stable/Release

# 5. Sign Release file
echo "[repo] Signing Release with GPG ($KEY_ID)..."
GPG_OPTS="-u $KEY_ID"
if [ -n "$GPG_PASSPHRASE" ]; then
    GPG_OPTS="$GPG_OPTS --batch --yes --pinentry-mode loopback --passphrase $GPG_PASSPHRASE"
fi

rm -f dists/stable/InRelease dists/stable/Release.gpg
gpg --batch --no-tty $GPG_OPTS --clearsign -o dists/stable/InRelease dists/stable/Release
gpg --batch --no-tty $GPG_OPTS -abs -o dists/stable/Release.gpg dists/stable/Release

# 6. Export Public Key
gpg --batch --no-tty --yes --armor --export "$KEY_ID" > public.key
cp public.key KEY.gpg

echo "✅ APT Repository updated."
echo "👉 Users can add it with:"
echo "   curl -fsSL https://jakej985-rgb.github.io/m3tal-apt-key/public.key | sudo gpg --dearmor -o /usr/share/keyrings/m3tal-archive-keyring.gpg"
echo "   echo 'deb [signed-by=/usr/share/keyrings/m3tal-archive-keyring.gpg] https://jakej985-rgb.github.io/m3tal-apt-key stable main' | sudo tee /etc/apt/sources.list.d/m3tal.list"
