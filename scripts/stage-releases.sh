#!/bin/sh
# Copy what the installer and `weft update` fetch into OUT: SITE/install and
# every real release in SITE/releases (test builds, versions ending in -test,
# are left out), with releases/latest naming LATEST. Without LATEST, SITE's own
# releases/latest is copied when it names a release that was staged.
# OUT may hold an earlier staging; a release in it that SITE no longer has, or
# has with other contents, is refused, never removed or merged into, so the
# site cannot serve a release by mistake.
# With --link-from KEPT, a release KEPT already holds (KEPT/<version>, as
# scripts/publish-site.sh keeps each pushed release once) is hard-linked from
# there instead of copied, so it is kept once however many trees hold it; one
# KEPT holds with other contents than SITE is refused. A release is never
# linked to SITE/releases itself: rebuilding one there would change it in
# every tree.
# A release holding a dot-named file (Finder's .DS_Store) is refused: the
# site's server refuses to start with one.
# With --check, only check, and stage nothing; OUT_DIR is then not given.
# Usage: scripts/stage-releases.sh [--link-from KEPT] [--check] SITE_DIR [OUT_DIR] [LATEST_VERSION]
set -eu
kept=
check=
while :; do
	case "${1:-}" in
	--link-from)
		kept="$2"
		shift 2
		;;
	--check)
		check=1
		shift
		;;
	*) break ;;
	esac
done
site="$1"
if [ -n "$check" ]; then
	out=
	latest="${2:-}"
else
	out="$2"
	latest="${3:-}"
fi
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
for dir in "$site"/releases/*/; do
	[ -d "$dir" ] || continue
	v=$(basename "$dir")
	real "$v" || continue
	dot=$(cd "$dir" && find . -name '.?*' | head -n 1)
	[ -z "$dot" ] ||
		fail "$site/releases/$v holds ${dot#./}, named with a leading dot, which the site does not serve; move it out"
done
for dir in ${out:+"$out"/releases/*/}; do
	[ -d "$dir" ] || continue
	v=$(basename "$dir")
	real "$v" || fail "$out/releases/$v is not a release in $site/releases; remove it by hand"
	diff -r "$site/releases/$v" "$dir" >/dev/null 2>&1 ||
		fail "$out/releases/$v differs from $site/releases/$v, so staging would merge two releases; remove it by hand"
done
if [ -n "$kept" ]; then
	for dir in "$site"/releases/*/; do
		[ -d "$dir" ] || continue
		v=$(basename "$dir")
		real "$v" && [ -d "$kept/$v" ] || continue
		diff -r "$dir" "$kept/$v" >/dev/null 2>&1 ||
			fail "$site/releases/$v differs from $kept/$v, the release $v a verified push served, whose users have its checksums; put $kept/$v back in $site/releases/$v"
	done
fi
[ -z "$check" ] || exit 0

mkdir -p "$out"
cp "$site/install" "$out/install"
for dir in "$site"/releases/*/; do
	[ -d "$dir" ] || continue
	v=$(basename "$dir")
	real "$v" || continue
	# Already staged, and the same (checked above).
	[ ! -d "$out/releases/$v" ] || continue
	mkdir -p "$out/releases"
	if [ -n "$kept" ] && [ -d "$kept/$v" ]; then
		sh "$(dirname "$0")/link-release.sh" "$kept/$v" "$out/releases/$v"
	else
		cp -R "$site/releases/$v" "$out/releases/"
	fi
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
