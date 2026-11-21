#!/bin/sh
# Copy what the installer and `weft update` fetch into OUT: SITE/install and
# the releases the site has published, those SITE/releases.txt lists, from
# SITE/releases, with LATEST, the one being published, beside them and named
# by releases/latest. Nothing else in SITE/releases is staged: not a dry run's
# build, not a folder put there by hand, not a test build (a version ending in
# -test). A listed release SITE/releases lacks is reported and left out
# (scripts/published-releases.sh check refuses that before a publish). Without
# LATEST, SITE's own releases/latest is copied when it names a release that
# was staged.
# OUT may hold an earlier staging; a release in it that is not one to stage,
# or that SITE has with other contents, is refused, never removed or merged
# into, so the site cannot serve a release by mistake.
# With --link-from KEPT, a release KEPT already holds (KEPT/<version>, as
# scripts/publish-site.sh keeps each pushed release once) is hard-linked from
# there instead of copied, so it is kept once however many trees hold it; one
# KEPT holds with other contents than SITE is refused. A release is never
# linked to SITE/releases itself: rebuilding one there would change it in
# every tree.
# A release holding a dot-named file (Finder's .DS_Store) is refused: the
# site's server refuses to start with one.
# With --live LIVE, the release the site serves now is staged too, though
# SITE/releases.txt may not list it (its publish went live, but was not
# verified in time to be listed): its users keep it. It must be in
# SITE/releases.
# With --check, only check, and stage nothing; OUT_DIR is then not given.
# Usage: scripts/stage-releases.sh [--link-from KEPT] [--live LIVE] [--check] SITE_DIR [OUT_DIR] [LATEST_VERSION]
set -eu
kept=
check=
live=
while :; do
	case "${1:-}" in
	--link-from)
		kept="$2"
		shift 2
		;;
	--live)
		live="$2"
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
if [ -n "$live" ]; then
	real "$live" || fail "$site/releases/$live, the release the site serves now, does not exist or is a test build"
fi
# The releases to stage: the published ones, the one the site serves now,
# then the one being published. No other folder in SITE/releases.
list="$site/releases.txt"
[ -f "$list" ] || fail "$list does not exist; it lists the releases the site has published, the ones staged"
versions=
while IFS= read -r v || [ -n "$v" ]; do
	v=$(printf '%s' "$v" | tr -d ' \r')
	[ -n "$v" ] || continue
	case " $versions " in *" $v "*) continue ;; esac
	if real "$v"; then
		versions="$versions $v"
	else
		echo "stage-releases: $list lists $v, which $site/releases does not have (or is a test build); not staged" >&2
	fi
done <"$list"
for v in $live $latest; do
	case " $versions " in *" $v "*) ;; *) versions="$versions $v" ;; esac
done
staged() {
	case " $versions " in *" $1 "*) return 0 ;; esac
	return 1
}
for v in $versions; do
	dir="$site/releases/$v"
	dot=$(cd "$dir" && find . -name '.?*' | head -n 1)
	[ -z "$dot" ] ||
		fail "$site/releases/$v holds ${dot#./}, named with a leading dot, which the site does not serve; move it out"
done
for dir in ${out:+"$out"/releases/*/}; do
	[ -d "$dir" ] || continue
	v=$(basename "$dir")
	staged "$v" || fail "$out/releases/$v is not a release to stage ($list does not list it, and it is neither live nor the one being published); remove it by hand"
	diff -r "$site/releases/$v" "$dir" >/dev/null 2>&1 ||
		fail "$out/releases/$v differs from $site/releases/$v, so staging would merge two releases; remove it by hand"
done
if [ -n "$kept" ]; then
	for v in $versions; do
		dir="$site/releases/$v"
		[ -d "$kept/$v" ] || continue
		diff -r "$dir" "$kept/$v" >/dev/null 2>&1 ||
			fail "$site/releases/$v differs from $kept/$v, the release $v a verified push served, whose users have its checksums; put $kept/$v back in $site/releases/$v"
	done
fi
[ -z "$check" ] || exit 0

mkdir -p "$out"
cp "$site/install" "$out/install"
for v in $versions; do
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
	staged "$latest" || {
		echo "stage-releases: $site/releases/latest names $latest, which is not staged; leaving latest out" >&2
		latest=
	}
fi
if [ -n "$latest" ]; then
	mkdir -p "$out/releases"
	printf '%s\n' "$latest" >"$out/releases/latest"
elif [ -f "$out/releases/latest" ]; then
	v=$(tr -d ' \r\n' <"$out/releases/latest")
	staged "$v" || fail "$out/releases/latest names $v, which is not a release to stage; remove it by hand"
fi
