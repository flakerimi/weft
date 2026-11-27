#!/bin/sh
# Make the public GitHub Release v<version> from the files weft.base.al
# serves for it: the archives SHA256SUMS lists, and SHA256SUMS, from DIR
# (site/releases/<version> unless given), so the checksums are the same in
# both places. The tag v<version> must be on GitHub already (gh --verify-tag);
# scripts/publish-site.sh pushes it before the site and runs this after the
# site push, and prints this command when gh could not run. It never replaces
# or deletes a release: one that exists stops it. A version with a hyphen
# (0.1.0-alpha.1) is marked a pre-release. WEFT_GITHUB_REPO names the
# repository (flakerimi/weft unless set).
# Usage: scripts/github-release.sh [--dry-run] <version> [DIR]
#   --dry-run  check what it can without gh and print the gh command
set -eu
dry=
if [ "${1:-}" = --dry-run ]; then
	dry=1
	shift
fi
version="${1:-}"
root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
repo=${WEFT_GITHUB_REPO:-flakerimi/weft}
tag="v$version"

fail() {
	echo "github-release: $1" >&2
	exit 1
}

scripts/version.sh "$version" ||
	fail "give a version like 0.0.1 or 0.1.0-alpha.1; usage: scripts/github-release.sh [--dry-run] <version> [DIR]."
dir="${2:-site/releases/$version}"
[ -f "$dir/SHA256SUMS" ] ||
	fail "$dir/SHA256SUMS is missing, so the files to attach are not known; name the folder of the release weft.base.al serves for $version."

# The files, in SHA256SUMS's order, then SHA256SUMS: exactly what the site serves.
set --
while read -r _ name; do
	[ -n "$name" ] || continue
	name=${name#\*} # shasum's mark of a file read in binary mode
	case "$name" in */* | .*) fail "$dir/SHA256SUMS lists $name, which is not a file of the release's own folder; list only the archives beside it." ;; esac
	[ -f "$dir/$name" ] || fail "$dir/SHA256SUMS lists $name, which is not in $dir; copy the release weft.base.al serves for $version there."
	set -- "$@" "$dir/$name"
done <"$dir/SHA256SUMS"
[ $# -gt 0 ] || fail "$dir/SHA256SUMS lists no file; name the folder of the release weft.base.al serves for $version."
set -- "$@" "$dir/SHA256SUMS"

pre=
case "$version" in *-*) pre=--prerelease ;; esac
notes="weft $version for darwin and linux, on amd64 and arm64: the same files, with the same SHA256SUMS, as https://weft.base.al/releases/$version/. Install with: curl -fsSL https://weft.base.al/install | sh. An app this release makes requires github.com/flakerimi/weft $tag."

if [ -n "$dry" ]; then
	printf 'dry run: would run: gh release create %s --repo %s --verify-tag --title %s --notes '"'"'%s'"'"'%s' "$tag" "$repo" "$tag" "$notes" "${pre:+ $pre}"
	printf ' %s' "$@"
	printf '\n'
	exit 0
fi

command -v gh >/dev/null 2>&1 ||
	fail "gh is not installed, so the GitHub Release $tag is not made; install it from https://cli.github.com, run gh auth login, and run scripts/github-release.sh $version again."
gh auth status >/dev/null 2>&1 ||
	fail "gh is not signed in, so the GitHub Release $tag is not made; run gh auth login, then scripts/github-release.sh $version again."
! gh release view "$tag" --repo "$repo" >/dev/null 2>&1 ||
	fail "the GitHub Release $tag already exists in $repo, and this script never replaces one; leave it, or remove it by hand and run scripts/github-release.sh $version again."
gh release create "$tag" --repo "$repo" --verify-tag --title "$tag" --notes "$notes" ${pre:+"$pre"} "$@" ||
	fail "gh release create $tag did not finish (see above); fix that and run scripts/github-release.sh $version again (a release gh made in part is listed by gh release view $tag --repo $repo)."
echo "made the GitHub Release $tag in $repo, with the files and SHA256SUMS of $dir"
