#!/bin/sh
# Build the playground's web app and run the Go binary that embeds it.
# Usage: scripts/playground.sh [build]   ("build" stops after building)
set -eu
cd "$(dirname "$0")/.."
pnpm install --frozen-lockfile
pnpm --filter @weft/themes build
pnpm --filter @weft/ui build
pnpm --filter playground build
[ "${1:-}" = build ] && exit 0
go run ./playground
