#!/bin/sh
# Build release binaries into RELEASES/<version>/, and name it in
# RELEASES/latest; RELEASES is site/releases unless given (a dry run builds
# elsewhere, leaving site/releases as it was). Does not tag, publish or list it
# in site/releases.txt. Each binary carries the bundle (the packages, the Go
# package, the catalogue and the app template): scripts/bundle.sh stages it
# into cmd/weft/bundle for go:embed, and afterwards, failed or not, it is moved
# out to a temporary folder, so the folder holds only its README.md again.
# Usage: scripts/release.sh <version> [RELEASES]
set -eu

version="${1:-}"
root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
scripts/version.sh "$version" || {
	echo "release: give a version like 0.0.1 or 0.1.0-alpha.1" >&2
	exit 2
}
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
for target in darwin/arm64 darwin/amd64 linux/arm64 linux/amd64; do
	os=${target%/*}
	arch=${target#*/}
	work=$(mktemp -d)
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
		-ldflags "-s -w -X github.com/flakerimi/weft.Version=$version" \
		-o "$work/weft" ./cmd/weft
	tar -czf "$out/weft_${os}_${arch}.tar.gz" -C "$work" weft
	echo "built $out/weft_${os}_${arch}.tar.gz"
done
(cd "$out" && if command -v sha256sum >/dev/null; then sha256sum weft_*.tar.gz; else shasum -a 256 weft_*.tar.gz; fi) >"$out/SHA256SUMS"
printf '%s\n' "$version" >"$releases/latest"
echo "release $version ready in $out, and named in $releases/latest; site/releases.txt lists it only once a publish pushes it"
