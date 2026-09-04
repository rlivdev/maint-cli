#!/bin/sh
set -e

BIN_DIR="/usr/local/bin"
TARGET="$BIN_DIR/maint"

IMAGE="rlivdev/maint-cli:latest"
BASE_URL="https://raw.githubusercontent.com/rlivdev/maint-cli/main"

if ! command -v docker >/dev/null 2>&1; then
    echo "Error: docker not found. Install first."
    exit 1
fi

if [ "$(id -u)" != "0" ]; then
    echo "Error: run as root or via sudo to install to $BIN_DIR."
    echo "Try: sudo sh install.sh"
    exit 1
fi

# Pull latest image (always, so updates propagate with :latest)
echo "Pulling $IMAGE..."
if ! docker pull "$IMAGE" >/dev/null 2>&1; then
    echo "Warning: could not pull $IMAGE from registry."
    if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
        echo "Error: no local image and could not pull from registry. Run install.sh from a cloned repo to build locally."
        exit 1
    fi
    echo "Using existing local image."
fi

# Download the maint script from the repo
echo "Downloading maint script..."
curl -fsSL "$BASE_URL/maint" -o "$TARGET"
chmod 0755 "$TARGET"

echo "Installed to $TARGET"
echo "Run: maint --help"
