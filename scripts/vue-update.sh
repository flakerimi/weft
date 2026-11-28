#!/bin/sh
# Move every pin of Vue to the newest 3.6 build on npm: the newest stable
# 3.6.x once Vue publishes one, else the newest 3.6 release candidate. Every
# package.json that pins a 3.6 build (the packages, the site, the playground,
# the shop, the benchmark's Weft page) and the app template move together, so
# the repository never holds two Vues; the frameworks' own benchmark pages
# (bench/nuxt, bench/astro) keep their Vue 3.5. Run it before each release,
# then pnpm install and the whole suite.
# Usage: scripts/vue-update.sh [--check]
#   --check  print the pinned and the newest build, and exit 1 if they differ
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

fail() {
	echo "vue-update: $1" >&2
	exit 1
}

check=
case "${1:-}" in
'') ;;
--check) check=1 ;;
*) fail "$1 is not an option it takes; usage: scripts/vue-update.sh [--check]." ;;
esac

newest=$(npm view vue versions --json 2>/dev/null | node -e '
let s = ""
process.stdin.on("data", (d) => (s += d)).on("end", () => {
  const all = JSON.parse(s).filter((v) => /^3\.6\.\d+(-rc\.\d+)?$/.test(v))
  const key = (v) => {
    const [, patch, rc] = /^3\.6\.(\d+)(?:-rc\.(\d+))?$/.exec(v)
    return [Number(patch), rc === undefined ? Infinity : Number(rc)]
  }
  all.sort((a, b) => { const [pa, ra] = key(a), [pb, rb] = key(b); return pa - pb || ra - rb })
  if (all.length) process.stdout.write(all[all.length - 1])
})') || true
[ -n "$newest" ] || fail "could not read Vue's versions from npm; check the network and run it again."

files=$(grep -l '"vue": "3\.6\.' packages/*/package.json site-src/package.json playground/web/package.json examples/shop/package.json bench/weft/package.json templates/app/package.json.tmpl 2>/dev/null || true)
[ -n "$files" ] || fail "no package.json pins a Vue 3.6 build; this script expects the pins in packages/, site-src/, playground/web/, examples/shop/, bench/weft/ and templates/app/."
# shellcheck disable=SC2086 # the file list is split into words on purpose
pinned=$(sed -n 's/.*"vue": "\(3\.6\.[^"]*\)".*/\1/p' $files | sort -u)
[ "$(printf '%s\n' "$pinned" | wc -l | tr -d ' ')" = 1 ] ||
	fail "the pins differ ($(printf '%s' "$pinned" | tr '\n' ' ')); make them one build, then run it again."

if [ -n "$check" ]; then
	echo "Vue pinned: $pinned; newest 3.6 on npm: $newest"
	[ "$pinned" = "$newest" ]
	exit
fi
if [ "$pinned" = "$newest" ]; then
	echo "Vue is already at $newest, the newest 3.6 build."
	exit 0
fi
for f in $files; do
	sed "s/\"vue\": \"$pinned\"/\"vue\": \"$newest\"/" "$f" >"$f.tmp" && mv "$f.tmp" "$f"
done
echo "Vue moved from $pinned to $newest in:"
# shellcheck disable=SC2086 # the file list is split into words on purpose
printf '  %s\n' $files
echo "Now run pnpm install and the whole suite, then commit."
