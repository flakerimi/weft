#!/bin/sh
# Roll weft.base.al back to a version published before: push again the tree
# scripts/publish-site.sh kept for it in site/pushed/<version>/ after its push
# was verified, and wait until the site serves it.
# Usage: scripts/rollback-site.sh [--dry-run] <version>
#   --dry-run  check everything, print the tree, and stop before the push
# A push replaces the whole site, so the tree must hold every release
# site/releases.txt lists (scripts/published-releases.sh): a tree kept before a
# release was published would drop that release, and is refused.
set -eu
dry=
if [ "${1:-}" = --dry-run ]; then
	dry=1
	shift
fi
version="${1:-}"
root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

fail() {
	echo "rollback: $1" >&2
	exit 1
}

# --dry-run counts only before the version: one anywhere else, an option this
# script does not know, or a second argument is refused before anything is
# done, so a dry run asked for in the wrong place never pushes.
for arg in "$@"; do
	case "$arg" in -*) fail "$arg is not taken, since no argument but --dry-run may begin with - and --dry-run goes first, before the version; usage: scripts/rollback-site.sh [--dry-run] <version>." ;; esac
done
[ $# -le 1 ] ||
	fail "takes one version and nothing after it; usage: scripts/rollback-site.sh [--dry-run] <version>."

scripts/version.sh "$version" ||
	fail "give a version like 0.0.1 or 0.1.0-alpha.1; usage: scripts/rollback-site.sh [--dry-run] <version>."
tree="$root/site/pushed/$version"
[ -f "$tree/Dockerfile" ] ||
	fail "site/pushed/$version holds no pushed tree; only a version scripts/publish-site.sh published from this checkout can be rolled back to."

scripts/published-releases.sh check site || exit 1
# The releases the tree serves: from site-src/releases, which the image has in
# /data/releases; a tree kept before releases moved out of the binary embeds
# them in its build instead.
served="$tree/site-src/releases/releases"
[ -d "$tree/site-src/releases" ] || served="$tree/site-src/dist/releases"
while IFS= read -r v || [ -n "$v" ]; do
	[ -n "$v" ] || continue
	# Every file, not only the checksums: a tarball changed in place through a link keeps them.
	diff -r "site/releases/$v" "$served/$v" >/dev/null 2>&1 ||
		fail "the tree kept for $version does not serve the published release $v as site/releases has it, so pushing it would drop or change that release; publish a new version with scripts/publish-site.sh instead."
done <site/releases.txt

echo "rollback tree: $tree"
if [ -n "$dry" ]; then
	echo "dry run: not pushed"
	exit 0
fi

lisa server push --name weft "$tree"
scripts/wait-live.sh "$version" || exit 1
echo "rolled weft.base.al back to $version"
