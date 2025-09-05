#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")"/.. && pwd)
BIN_DIR="$ROOT_DIR/bin"
RUN_DIR="$ROOT_DIR/.run"
APP="$BIN_DIR/server"
PID_FILE="$RUN_DIR/server.pid"

mkdir -p "$BIN_DIR" "$RUN_DIR"

echo "Building..."
GO111MODULE=on go build -o "$APP" ./cmd/server

if [[ -f "$PID_FILE" ]]; then
  PID=$(cat "$PID_FILE" || true)
  if [[ -n "$PID" ]] && kill -0 "$PID" 2>/dev/null; then
    echo "Stopping existing process $PID..."
    kill -TERM "$PID"
    wait "$PID" 2>/dev/null || true
  fi
fi

echo "Starting..."
"$APP" &
echo $! > "$PID_FILE"
echo "Started with PID $(cat "$PID_FILE")"

