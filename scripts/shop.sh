#!/bin/sh
# Run the example app, examples/shop, with this repository's Weft. It stages
# the bundle into a new folder under $TMPDIR with scripts/bundle.sh, builds
# weft from the repository into the same folder, writes examples/shop/.env
# when there is none (with a fresh AUTH_SECRET), and then, in examples/shop
# with WEFT_BUNDLE set:
#   dev (the default)  weft dev, on http://localhost:5173 (or the port
#                      SHOP_PORT names)
#   build              weft build, which writes examples/shop/shop
#   test               weft gen (which unpacks .weft/ and installs the
#                      packages), then go test -tags dev ./...
# What it makes in examples/shop (.weft/, node_modules/, web/dist/,
# web/.weft-types/, data/, .env, shop) is git-ignored. It deletes nothing: an
# earlier examples/shop/.weft is moved into the temporary folder.
# Usage: scripts/shop.sh [dev|build|test]
set -eu

what=${1:-dev}
case "$what" in
dev | build | test) ;;
*)
	echo "usage: scripts/shop.sh [dev|build|test]" >&2
	exit 2
	;;
esac
if [ $# -gt 1 ]; then
	echo "usage: scripts/shop.sh [dev|build|test]" >&2
	exit 2
fi

root=$(cd "$(dirname "$0")/.." && pwd)
app="$root/examples/shop"
base=${TMPDIR:-/tmp}
tmp=$(mktemp -d "${base%/}/weft-shop.XXXXXX")
echo "scripts/shop.sh: Weft is staged in $tmp" >&2

"$root/scripts/bundle.sh" "$tmp/bundle"
(cd "$root" && go build -o "$tmp/weft" ./cmd/weft)

if [ ! -e "$app/.env" ]; then
	secret=$(head -c 48 /dev/urandom | base64 | tr -d '\n')
	umask 077
	cat >"$app/.env" <<EOF
PORT=8080
DB_DRIVER=sqlite
DB_URL=file:data/shop.db
AUTH_SECRET=$secret
AUTH_OWNER_EMAIL=owner@example.com
EOF
	umask 022
	echo "scripts/shop.sh: wrote examples/shop/.env, for the owner owner@example.com" >&2
fi

# weft unpacks a bundle only when .weft/ holds another version, and this
# repository's Weft is always 0.0.0-dev: an earlier .weft/ would be kept as it
# was, with the Go and the packages of an earlier run. Move it into the
# temporary folder (nothing is deleted), so weft unpacks the one just staged.
if [ -e "$app/.weft" ]; then
	mv "$app/.weft" "$tmp/earlier-weft"
	echo "scripts/shop.sh: the earlier examples/shop/.weft is in $tmp/earlier-weft" >&2
fi

export WEFT_BUNDLE="$tmp/bundle"
cd "$app"
case "$what" in
dev)
	# weft dev does not give the Go server the page server's address, so the
	# setup link it logs would name the Go server's own port, which serves no
	# pages under weft dev. APP_URL in the environment wins over .env.
	port=${SHOP_PORT:-5173}
	APP_URL=${APP_URL:-http://localhost:$port} exec "$tmp/weft" dev --port "$port"
	;;
build)
	exec "$tmp/weft" build
	;;
test)
	# Nothing in weft only unpacks .weft/, which go test needs for the Weft
	# module; weft gen does it on the way.
	"$tmp/weft" gen
	exec go test -tags dev ./...
	;;
esac
