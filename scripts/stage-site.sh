#!/bin/sh
# Copy what weft.base.al serves into OUT: the page, the installer and every real
# release (test builds, versions ending in -test, are left out), with `latest`.
# A push replaces the whole site, so older releases must be staged every time.
# Usage: scripts/stage-site.sh SITE_DIR OUT_DIR LATEST_VERSION
set -eu
site="$1"
out="$2"
latest="$3"
[ -d "$site/releases/$latest" ] || { echo "stage-site: $site/releases/$latest does not exist" >&2; exit 1; }
cp "$site/index.html" "$site/install" "$out/"
mkdir -p "$out/releases"
for dir in "$site"/releases/*/; do
	v=$(basename "$dir")
	case "$v" in *-test) continue ;; esac
	cp -R "$site/releases/$v" "$out/releases/"
done
printf '%s\n' "$latest" >"$out/releases/latest"
