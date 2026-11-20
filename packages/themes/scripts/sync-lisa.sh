#!/bin/sh
# Refresh the vendored Lisa token contract. Usage: sh scripts/sync-lisa.sh [path-to-tokens.json]
set -eu
src="${1:-${LISA_TOKENS:-/Volumes/External/Lisa/docs/contract/tokens.json}}"
[ -f "$src" ] || { echo "sync-lisa: $src not found; pass the path to Lisa's docs/contract/tokens.json" >&2; exit 1; }
cp "$src" "$(dirname "$0")/../src/lisa/tokens.json"
echo "copied $src"
