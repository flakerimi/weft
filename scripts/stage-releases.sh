#!/bin/sh
# Copy what the installer and `weft update` fetch into OUT: SITE/install and
# every real release in SITE/releases (test builds, versions ending in -test,
# are left out), with releases/latest naming LATEST. Without LATEST, SITE's own
# releases/latest is copied when it names a release that was staged.
# OUT may hold an earlier staging; a release in it that SITE no longer has is
# refused, never removed, so the site cannot serve a release by mistake.
# Usage: scripts/stage-releases.sh SITE_DIR OUT_DIR [LATEST_VERSION]
set -eu
site="$1"
out="$2"
latest="${3:-}"
fail() {
	echo "stage-releases: $1" >&2
	exit 1
}
real() {
	case "$1" in '' | *-test) return 1 ;; esac
	[ -d "$site/releases/$1" ]
}
if [ -n "$latest" ]; then
	real "$latest" || fail "$site/releases/$latest does not exist or is a test build"
fi
for dir in "$out"/releases/*/; do
	[ -d "$dir" ] || continue
	v=$(basename "$dir")
	real "$v" || fail "$out/releases/$v is not a release in $site/releases; remove it by hand"
done

mkdir -p "$out"
cp "$site/install" "$out/install"
for dir in "$site"/releases/*/; do
	[ -d "$dir" ] || continue
	v=$(basename "$dir")
	real "$v" || continue
	mkdir -p "$out/releases"
	cp -R "$site/releases/$v" "$out/releases/"
done
if [ -z "$latest" ] && [ -f "$site/releases/latest" ]; then
	latest=$(tr -d ' \r\n' <"$site/releases/latest")
	real "$latest" || {
		echo "stage-releases: $site/releases/latest names $latest, which is not staged; leaving latest out" >&2
		latest=
	}
fi
if [ -n "$latest" ]; then
	mkdir -p "$out/releases"
	printf '%s\n' "$latest" >"$out/releases/latest"
elif [ -f "$out/releases/latest" ]; then
	v=$(tr -d ' \r\n' <"$out/releases/latest")
	real "$v" || fail "$out/releases/latest names $v, which is not a release in $site/releases; remove it by hand"
fi
