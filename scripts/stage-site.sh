#!/bin/sh
# Copy what the static weft.base.al serves into OUT: the page, the installer and
# every real release (test builds, versions ending in -test, are left out), with
# `latest`. A push replaces the whole site, so older releases must be staged
# every time. This staged the static site, which can no longer be published
# (scripts/publish-site-static.sh says why); the Weft-built site stages the
# same files with scripts/stage-releases.sh.
# Usage: scripts/stage-site.sh SITE_DIR OUT_DIR LATEST_VERSION
set -eu
site="$1"
out="$2"
latest="$3"
[ -d "$site/releases/$latest" ] || { echo "stage-site: $site/releases/$latest does not exist" >&2; exit 1; }
cp "$site/index.html" "$out/"
sh "$(dirname "$0")/stage-releases.sh" "$site" "$out" "$latest"
