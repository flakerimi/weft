#!/bin/sh
# Build release binaries into site/releases/<version>/. Does not tag or publish.
# Usage: scripts/release.sh <version>
set -eu

version="${1:-}"
printf '%s' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[a-z0-9.]+)?$' || {
	echo "release: give a version like 0.0.1 or 0.1.0-alpha.1" >&2
	exit 2
}

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
go test ./...

out="site/releases/$version"
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
printf '%s\n' "$version" >site/releases/latest
echo "release $version ready in $out"
