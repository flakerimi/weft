#!/bin/sh
# SITE/releases.txt lists every version weft.base.al has published, one per
# line. A push replaces the whole site, so each must be in SITE/releases when
# the site is published, or WEFT_VERSION pins to it stop working.
# Usage: scripts/published-releases.sh check SITE_DIR
#          fail, in one sentence, when a listed version is not in SITE/releases
#        scripts/published-releases.sh add SITE_DIR VERSION
#          list VERSION (once); run only after a push is verified live
set -eu
cmd="${1:-}"
site="${2:-}"
list="$site/releases.txt"
fail() {
	echo "published-releases: $1" >&2
	exit 1
}
[ -n "$site" ] && [ -f "$list" ] || fail "$list does not exist; usage: published-releases.sh check|add SITE_DIR [VERSION]"
case "$cmd" in
check)
	missing=
	while IFS= read -r v || [ -n "$v" ]; do
		[ -n "$v" ] || continue
		[ -f "$site/releases/$v/SHA256SUMS" ] || missing="$missing $v"
	done <"$list"
	[ -z "$missing" ] ||
		fail "$site/releases lacks the published release(s)$missing, which a push would drop from the site; fetch each from https://weft.base.al/releases/<version>/ (SHA256SUMS and the files it lists) first."
	;;
add)
	version="${3:-}"
	sh "$(dirname "$0")/version.sh" "$version" || fail "give a version like 0.0.1 or 0.1.0-alpha.1 to add."
	grep -qxF "$version" "$list" || printf '%s\n' "$version" >>"$list"
	;;
*)
	fail "usage: published-releases.sh check|add SITE_DIR [VERSION]"
	;;
esac
