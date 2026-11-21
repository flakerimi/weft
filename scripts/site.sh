#!/bin/sh
# Build weft.base.al from site-src with Weft and run the Go binary that serves it.
# The binary embeds the pages only. To run it, the installer and every
# published release (site/releases.txt) are staged into site-src/releases (git-ignored), the
# folder it serves them from (WEFT_RELEASES_DIR), as the image has them in
# /data/releases.
# Usage: scripts/site.sh [build]
#   build  stops after building site-src/dist
set -eu
cd "$(dirname "$0")/.."
# Staged here before releases moved out of the binary: the page build would embed them.
for old in site-src/public/install site-src/public/releases; do
	[ ! -e "$old" ] || {
		echo "site: $old is left from an earlier staging, and the page build would embed it in the binary; move it out (releases are staged into site-src/releases now)." >&2
		exit 1
	}
done
pnpm install --frozen-lockfile
pnpm --filter @weft/themes build
pnpm --filter @weft/ui build
pnpm --filter weft-site build
[ "${1:-}" = build ] && exit 0
scripts/stage-releases.sh site site-src/releases
WEFT_RELEASES_DIR=site-src/releases go run ./site-src
