#!/bin/sh
# Run the example app, examples/shop, with this repository's Weft. It stages
# the bundle into a new folder under $TMPDIR with scripts/bundle.sh, builds
# weft from the repository into the same folder, writes examples/shop/.env
# when there is none (with a fresh AUTH_SECRET), and then, in examples/shop
# with WEFT_BUNDLE set:
#   dev (the default)  weft dev, on http://localhost:5173 (or the port
#                      SHOP_PORT names)
#   build              weft build, which writes examples/shop/shop
#   test               weft gen (which unpacks .weft/, installs the packages
#                      and writes the client), then the Go tests
#                      (go test -tags dev ./...), the web tests (pnpm test)
#                      and the type check (pnpm typecheck)
# What it makes in examples/shop (.weft/, node_modules/, web/dist/,
# web/.weft-types/, data/, .env, pnpm-lock.yaml, shop) is git-ignored. It
# deletes nothing; each run leaves its staging folder, whose path it prints.
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
	# Only .env is private; the umask of the rest of the run is the person's.
	(
		umask 077
		cat >"$app/.env" <<EOF
PORT=8080
DB_DRIVER=sqlite
DB_URL=file:data/shop.db
AUTH_SECRET=$secret
AUTH_OWNER_EMAIL=owner@example.com
API_KEYS=on
EOF
	)
	echo "scripts/shop.sh: wrote examples/shop/.env, for the owner owner@example.com" >&2
fi

export WEFT_BUNDLE="$tmp/bundle"
cd "$app"
case "$what" in
dev)
	exec "$tmp/weft" dev --port "${SHOP_PORT:-5173}"
	;;
build)
	exec "$tmp/weft" build
	;;
test)
	"$tmp/weft" gen
	go test -tags dev ./...
	pnpm test
	exec pnpm typecheck
	;;
esac
