#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="$ROOT_DIR/../build"
OUT_DIR="$BUILD_DIR/packages"
GOARCH=$(go env GOARCH)
PKG_DIR="$OUT_DIR/baudlink_linux_${GOARCH}"

mkdir -p "$PKG_DIR/bin" "$PKG_DIR/systemd"

# Build service binary
GOOS=linux GOARCH=$GOARCH go build -o "$PKG_DIR/bin/baudlink-service" ./cmd/service

# Copy systemd unit
cp "$ROOT_DIR/linux/baudlink.service" "$PKG_DIR/systemd/baudlink.service"
# Copy installer
cp "$ROOT_DIR/linux/install.sh" "$PKG_DIR/install.sh"
chmod +x "$PKG_DIR/install.sh"

mkdir -p "$OUT_DIR"
tar -czf "$OUT_DIR/baudlink_linux_${GOARCH}.tar.gz" -C "$OUT_DIR" "baudlink_linux_${GOARCH}"

echo "Created package: $OUT_DIR/baudlink_linux_${GOARCH}.tar.gz"