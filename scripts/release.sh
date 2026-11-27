#!/bin/sh
# Build release binaries into RELEASES/<version>/, and name it in
# RELEASES/latest; RELEASES is site/releases unless given (a dry run builds
# elsewhere, leaving site/releases as it was). Does not tag, publish or list it
# in site/releases.txt (scripts/publish-site.sh tags it and makes its GitHub
# Release). Each binary carries the bundle (the packages, the catalogue and
# the app template; not the Go package, which a release's apps require at the
# tag v<version> from GitHub): scripts/bundle.sh stages it
# into cmd/weft/bundle for go:embed, and afterwards, failed or not, it is moved
# out to a temporary folder, so the folder holds only its README.md again.
# WEFT_TARGETS names the systems to build, space-separated os/arch pairs; it
# defaults to all six systems the site serves (darwin/arm64 darwin/amd64
# linux/arm64 linux/amd64 windows/arm64 windows/amd64), which
# scripts/publish-site.sh builds. A darwin or linux
# target is weft_<os>_<arch>.tar.gz holding weft; a windows one is
# weft_windows_<arch>.zip holding weft.exe, made with zip. SHA256SUMS lists
# every archive. .github/workflows/release.yml builds all six on a tag.
# Usage: [WEFT_TARGETS="os/arch ..."] scripts/release.sh <version> [RELEASES]
set -eu

version="${1:-}"
root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
scripts/version.sh "$version" || {
	echo "release: give a version like 0.0.1 or 0.1.0-alpha.1" >&2
	exit 2
}
targets=${WEFT_TARGETS:-darwin/arm64 darwin/amd64 linux/arm64 linux/amd64 windows/arm64 windows/amd64}
# $targets is split on spaces but never globbed, so a * is refused as itself;
# restore and the SHA256SUMS line below glob, and turn it back on for themselves.
set -f
# shellcheck disable=SC2086
[ -n "$(printf '%s' $targets)" ] || {
	echo "release: WEFT_TARGETS names no target; name os/arch pairs such as \"linux/amd64 windows/arm64\", or unset it for the default six." >&2
	exit 2
}
for target in $targets; do
	case "$target" in
	darwin/amd64 | darwin/arm64 | linux/amd64 | linux/arm64) ;;
	windows/amd64 | windows/arm64)
		command -v zip >/dev/null || {
			echo "release: $target is archived with zip, which is not on PATH; install zip, or leave windows out of WEFT_TARGETS." >&2
			exit 2
		}
		;;
	*)
		echo "release: WEFT_TARGETS names $target; name only darwin, linux or windows, each as amd64 or arm64, such as \"linux/amd64 windows/arm64\"." >&2
		exit 2
		;;
	esac
done
# The Go the release ships: the root package (Version), warp and every
# package under it, and the CLI; the playground needs a web build and is not
# shipped. -short: the full CLI tests install apps from the registry and build
# a release themselves, so they run in the suite and in CI, not here.
go test -short . ./warp/... ./cmd/weft

bundle="$root/cmd/weft/bundle"
# restore moves what bundle.sh staged out of cmd/weft/bundle; nothing is
# deleted. Without bundle.sh's mark the folder is not its output, and is left.
restore() {
	[ -e "$bundle/.weft-bundle" ] || return 0
	set +f
	kept=$(mktemp -d)
	for f in "$bundle"/* "$bundle"/.[!.]* "$bundle"/..?*; do
		[ -e "$f" ] || continue
		[ "$(basename "$f")" = README.md ] || mv "$f" "$kept/"
	done
	echo "release: moved the staged bundle out of cmd/weft/bundle into $kept" >&2
}
trap restore EXIT
trap 'exit 1' HUP INT TERM
WEFT_VERSION="$version" scripts/bundle.sh "$bundle"

releases="${2:-site/releases}"
out="$releases/$version"
mkdir -p "$out"
for target in $targets; do
	os=${target%/*}
	arch=${target#*/}
	exe=weft
	[ "$os" != windows ] || exe=weft.exe
	work=$(mktemp -d)
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
		-ldflags "-s -w -X github.com/flakerimi/weft.Version=$version" \
		-o "$work/$exe" ./cmd/weft
	if [ "$os" = windows ]; then
		# -j stores weft.exe at the top, -X leaves out the Unix owner and
		# times; made beside the binary, so an older zip is replaced, not added to.
		archive="$out/weft_${os}_${arch}.zip"
		zip -q -X -j "$work/weft.zip" "$work/$exe"
		mv "$work/weft.zip" "$archive"
	else
		archive="$out/weft_${os}_${arch}.tar.gz"
		tar -czf "$archive" -C "$work" weft
	fi
	echo "built $archive"
done
# The tarballs first, as before there were zips, then the zips; each by name.
(cd "$out" && set +f && set -- && for f in weft_*.tar.gz weft_*.zip; do
	[ ! -e "$f" ] || set -- "$@" "$f"
done && if command -v sha256sum >/dev/null; then sha256sum "$@"; else shasum -a 256 "$@"; fi) >"$out/SHA256SUMS"
printf '%s\n' "$version" >"$releases/latest"
echo "release $version ready in $out, and named in $releases/latest; site/releases.txt lists it only once a publish pushes it"
