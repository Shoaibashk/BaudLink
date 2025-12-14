#!/usr/bin/env bash
set -euo pipefail

PORT=${1:-50055}
RETRY_SECONDS=${2:-10}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="$ROOT_DIR/build"

BIN_SUFFIX=""
if [[ "$(uname -s)" == MINGW* || "$(uname -s)" == MSYS* || "$(uname -s)" == CYGWIN* ]]; then
  BIN_SUFFIX=".exe"
fi

SERVICE="$BUILD_DIR/baudlink-service${BIN_SUFFIX}"
CLI="$BUILD_DIR/baudlink-cli${BIN_SUFFIX}"

if [[ ! -x "$SERVICE" ]]; then
  echo "Service binary not found or not executable: $SERVICE" >&2
  exit 2
fi
if [[ ! -x "$CLI" ]]; then
  echo "CLI binary not found or not executable: $CLI" >&2
  exit 2
fi

ADDR="localhost:$PORT"

"$SERVICE" serve --address "$ADDR" --debug >/tmp/baudlink-service.log 2>&1 &
SERVICE_PID=$!
echo "Started service pid=$SERVICE_PID addr=$ADDR"

trap 'echo "Stopping service pid=$SERVICE_PID"; kill $SERVICE_PID >/dev/null 2>&1 || true; wait $SERVICE_PID 2>/dev/null || true' EXIT

start_time=$(date +%s)
while true; do
  if "$CLI" --address "$ADDR" info --json >/dev/null 2>&1; then
    echo "Service ready"
    break
  fi
  if (( $(date +%s) - start_time > RETRY_SECONDS )); then
    echo "Service did not become ready after ${RETRY_SECONDS}s" >&2
    cat /tmp/baudlink-service.log || true
    exit 3
  fi
  sleep 0.2
done

echo "Running scan (expect JSON array or null):"
SCAN_OUT=$("$CLI" --address "$ADDR" scan --json || true)
if [[ "$SCAN_OUT" == "null" || -z "$SCAN_OUT" ]]; then
  echo "No ports found"
else
  echo "$SCAN_OUT" | jq . >/dev/null 2>&1 || { echo "Scan output not JSON:"; echo "$SCAN_OUT"; exit 4; }
  echo "Scan JSON valid"
fi

echo "Running info (assert grpc_address):"
INFO_OUT=$("$CLI" --address "$ADDR" info --json)
echo "$INFO_OUT" | jq -e ".config.grpc_address == \"$ADDR\"" >/dev/null
echo "Info grpc_address matches"

echo "Smoke test succeeded"
exit 0
