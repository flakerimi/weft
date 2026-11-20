#!/bin/sh
# Build weft.base.al from site-src with Weft and run the Go binary that serves it.
# The installer and every real release in site/releases are staged into
# site-src/public first (git-ignored), so the binary serves them too.
# Usage: scripts/site.sh [build [LATEST_VERSION]]
#   build           stops after building site-src/dist
#   LATEST_VERSION  what /releases/latest names (default: site/releases/latest)
set -eu
cd "$(dirname "$0")/.."
pnpm install --frozen-lockfile
pnpm --filter @weft/themes build
pnpm --filter @weft/ui build
scripts/stage-releases.sh site site-src/public "${2:-}"
pnpm --filter weft-site build
[ "${1:-}" = build ] && exit 0
go run ./site-src
