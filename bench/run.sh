#!/bin/sh
# Weft against Astro: build both landings, serve them, measure them.
#
#   Weft   bench/weft, built by Weft and served by its own binary (warp.Serve)
#   Astro  bench/astro, a production build, served by `astro preview` and by a
#          plain Go static file server with gzip on the fly (bench/static)
#
# Writes bench/results/<date>.json and the table in bench/README.md.
# Needs Node 22.23.4 or later, pnpm, Go and Google Chrome (CHROME_PATH).
# Usage: bench/run.sh [measure.mjs options, e.g. --runs 1 --duration 2s]
set -eu
cd "$(dirname "$0")/.."
: "${CHROME_PATH:=/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
export CHROME_PATH

WEFT_ADDR=127.0.0.1:4301
STATIC_ADDR=127.0.0.1:4302
PREVIEW_PORT=4321
export WEFT_URL="http://$WEFT_ADDR" ASTRO_STATIC_URL="http://$STATIC_ADDR" ASTRO_PREVIEW_URL="http://127.0.0.1:$PREVIEW_PORT"

for addr in "$WEFT_ADDR" "$STATIC_ADDR" "127.0.0.1:$PREVIEW_PORT"; do
	if curl -s -o /dev/null "http://$addr/"; then
		echo "bench: something already answers on $addr; stop it first." >&2
		exit 1
	fi
done

# Builds. Weft's packages, then both sites from the same image source.
pnpm install --frozen-lockfile
pnpm --filter @weft/themes build
pnpm --filter @weft/ui build
(cd bench/tools && npm ci --no-fund --no-audit)
(cd bench/astro && npm ci --no-fund --no-audit)
node bench/tools/images.mjs
pnpm --filter weft-bench build
(cd bench/astro && npx astro build)

bin=bench/.bin
mkdir -p "$bin"
go build -o "$bin/weft-bench" ./bench/weft
go build -o "$bin/static" ./bench/static
go build -o "$bin/load" ./bench/load

pids=""
stop() {
	for p in $pids; do kill "$p" 2>/dev/null || true; done
	(cd bench/astro && npx astro preview stop >/dev/null 2>&1) || true
}
trap stop EXIT INT TERM

# One core for each Go server; `astro preview` is Node, one JavaScript thread.
GOMAXPROCS=1 ADDR="$WEFT_ADDR" "$bin/weft-bench" &
pids="$pids $!"
GOMAXPROCS=1 "$bin/static" -dir bench/astro/dist -addr "$STATIC_ADDR" &
pids="$pids $!"
(cd bench/astro && npx astro preview --background --host 127.0.0.1 --port "$PREVIEW_PORT")

for url in "$WEFT_URL/" "$ASTRO_STATIC_URL/" "$ASTRO_PREVIEW_URL/"; do
	i=0
	until curl -sf -o /dev/null "$url"; do
		i=$((i + 1))
		if [ "$i" -gt 100 ]; then
			echo "bench: $url did not answer" >&2
			exit 1
		fi
		sleep 0.1
	done
done

out="bench/results/$(date +%F).json"
node bench/tools/measure.mjs --load "$bin/load" --out "$out" "$@"
node bench/tools/table.mjs "$out" bench/README.md
echo "bench: wrote $out and the table in bench/README.md"
