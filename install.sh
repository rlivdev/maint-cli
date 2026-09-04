#!/bin/sh
set -e

BIN_DIR="/usr/local/bin"
TARGET="$BIN_DIR/maint"
SOURCE="$(dirname "$0")/maint"

IMAGE="rlivdev/maint-cli:latest"

if ! command -v docker >/dev/null 2>&1; then
    echo "Error: docker not found. Install first."
    exit 1
fi

# Pull latest image (always, so updates propagate with :latest)
echo "Pulling $IMAGE..."
if ! docker pull "$IMAGE" >/dev/null 2>&1; then
    echo "Warning: could not pull $IMAGE from registry."
    if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
        [ -f "$SOURCE" ] || { echo "Error: no local image and 'maint' script not found next to install.sh."; exit 1; }
        echo "Building local image..."
        (cd "$(dirname "$0")" && docker build -t "$IMAGE" .)
    else
        echo "Using existing local image."
    fi
fi

install -m 0755 "$SOURCE" "$TARGET"

echo "Installed to $TARGET"
echo "Run: maint --help"
