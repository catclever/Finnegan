#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

DEFAULT_HOME="${FINNEGAN_HOME:-$HOME/Documents/Finnegan}"
OUT_PATH="${1:-$DEFAULT_HOME/.finnegan/plugins/curio.plugin}"

mkdir -p "$(dirname "$OUT_PATH")"

cd "$ROOT_DIR"
CGO_ENABLED=0 go build -o "$OUT_PATH" ./plugins/curio
chmod +x "$OUT_PATH"
echo "Built curio plugin: $OUT_PATH"
