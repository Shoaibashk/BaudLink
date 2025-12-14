#!/usr/bin/env bash
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  echo "This installer must be run as root. Use sudo." >&2
  exit 1
fi

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
BIN_DST=/usr/local/bin
UNIT_DST=/etc/systemd/system

echo "Installing BaudLink to $BIN_DST and systemd unit to $UNIT_DST"

cp "$ROOT_DIR/bin/baudlink-service" "$BIN_DST/baudlink-service"
chmod +x "$BIN_DST/baudlink-service"

cp "$ROOT_DIR/systemd/baudlink.service" "$UNIT_DST/baudlink.service"

systemctl daemon-reload
systemctl enable baudlink.service
systemctl start baudlink.service

echo "Installation complete. Use 'systemctl status baudlink' to check service status."