#!/bin/sh
# Weft against Astro, Nuxt, Next.js and SvelteKit: build the same landing with
# each, serve them, measure them.
#
#   Weft       bench/weft, built by Weft and served by its own binary (weft.Serve)
#   Astro      bench/astro, a static build, served by `astro preview`
#   Nuxt       bench/nuxt, prerendered, served by its Node server (Nitro)
#   Next.js    bench/next, prerendered, served by `next start`
#   SvelteKit  bench/sveltekit, prerendered, served by adapter-node's server
#
# Each of the four also has a static build, which is served twice by a plain
# Go file server (bench/static): gzipped on the fly, and from memory with the
# brotli and gzip stored at build, the terms weft.Serve gives Weft.
#
# Writes bench/results/<date>.json and the table in bench/README.md.
# Needs Node 22.23.4 or later, pnpm, Go and Google Chrome (CHROME_PATH).
# Usage: bench/run.sh [measure.mjs options, e.g. --runs 1 --duration 2s]
# A trial run: BENCH_TRIAL=<file> writes the results there and leaves
# bench/results/ and the README alone.
set -eu
cd "$(dirname "$0")/.."
: "${CHROME_PATH:=/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
export CHROME_PATH

# The ports. bench/tools/measure.mjs has the same numbers as its defaults.
WEFT=4301
ASTRO_GZIP=4302 ASTRO_PACKED=4303 ASTRO_OWN=4321
NUXT_GZIP=4311 NUXT_PACKED=4312 NUXT_OWN=4313
NEXT_GZIP=4331 NEXT_PACKED=4332 NEXT_OWN=4333
SVELTEKIT_GZIP=4341 SVELTEKIT_PACKED=4342 SVELTEKIT_OWN=4343
ports="$WEFT $ASTRO_GZIP $ASTRO_PACKED $ASTRO_OWN $NUXT_GZIP $NUXT_PACKED $NUXT_OWN $NEXT_GZIP $NEXT_PACKED $NEXT_OWN $SVELTEKIT_GZIP $SVELTEKIT_PACKED $SVELTEKIT_OWN"

for port in $ports; do
	if curl -s -o /dev/null "http://127.0.0.1:$port/"; then
		echo "bench: something already answers on 127.0.0.1:$port; stop it first." >&2
		exit 1
	fi
done

# Builds. Weft's packages, then every site from the same content and image.
pnpm install --frozen-lockfile
pnpm --filter @weft/themes build
pnpm --filter @weft/ui build
for dir in tools astro nuxt next sveltekit; do
	(cd "bench/$dir" && npm ci --no-fund --no-audit)
done
node bench/tools/images.mjs
pnpm --filter weft-bench build
(cd bench/astro && npx astro build)
for dir in nuxt next sveltekit; do
	(cd "bench/$dir" && npm run build:static && npm run build:server)
done

# Each static build, copied with its brotli and gzip stored beside each file.
packed=bench/.packed
node bench/tools/precompress.mjs bench/astro/dist "$packed/astro"
node bench/tools/precompress.mjs bench/nuxt/.output-static/public "$packed/nuxt"
node bench/tools/precompress.mjs bench/next/out-static "$packed/next"
node bench/tools/precompress.mjs bench/sveltekit/build-static "$packed/sveltekit"

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

# One core for each Go server. The frameworks' own servers are Node: one
# JavaScript thread each, and nothing here holds them to one core.
GOMAXPROCS=1 ADDR="127.0.0.1:$WEFT" "$bin/weft-bench" &
pids="$pids $!"

# static <folder> <port> [-precompressed]
static() {
	GOMAXPROCS=1 "$bin/static" -dir "$1" -addr "127.0.0.1:$2" ${3:-} &
	pids="$pids $!"
}
static bench/astro/dist "$ASTRO_GZIP"
static bench/nuxt/.output-static/public "$NUXT_GZIP"
static bench/next/out-static "$NEXT_GZIP"
static bench/sveltekit/build-static "$SVELTEKIT_GZIP"
static "$packed/astro" "$ASTRO_PACKED" -precompressed
static "$packed/nuxt" "$NUXT_PACKED" -precompressed
static "$packed/next" "$NEXT_PACKED" -precompressed
static "$packed/sveltekit" "$SVELTEKIT_PACKED" -precompressed

(cd bench/astro && npx astro preview --background --host 127.0.0.1 --port "$ASTRO_OWN")
HOST=127.0.0.1 PORT="$NUXT_OWN" NODE_ENV=production node bench/nuxt/.output/server/index.mjs &
pids="$pids $!"
(cd bench/next && NODE_ENV=production exec node node_modules/next/dist/bin/next start -H 127.0.0.1 -p "$NEXT_OWN") &
pids="$pids $!"
HOST=127.0.0.1 PORT="$SVELTEKIT_OWN" NODE_ENV=production node bench/sveltekit/build/index.js &
pids="$pids $!"

for port in $ports; do
	i=0
	until curl -sf -o /dev/null "http://127.0.0.1:$port/"; do
		i=$((i + 1))
		if [ "$i" -gt 100 ]; then
			echo "bench: http://127.0.0.1:$port/ did not answer" >&2
			exit 1
		fi
		sleep 0.1
	done
done

if [ -n "${BENCH_TRIAL:-}" ]; then
	node bench/tools/measure.mjs --load "$bin/load" --out "$BENCH_TRIAL" "$@"
	echo "bench: wrote the trial run $BENCH_TRIAL"
	exit 0
fi
out="bench/results/$(date +%F).json"
node bench/tools/measure.mjs --load "$bin/load" --out "$out" "$@"
node bench/tools/table.mjs "$out" bench/README.md
echo "bench: wrote $out and the table in bench/README.md"
