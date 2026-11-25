#!/bin/sh
# Stage the bundle `weft` carries: the runtime files of the JS packages, the Go
# package, the catalogue and the app template. A release stages it into
# cmd/weft/bundle before building weft; WEFT_BUNDLE can name any such folder.
# OUT must be missing, empty, hold only a README.md, or be a bundle this script
# wrote (it has .weft-bundle); anything else is refused. Builds @weft/themes and
# @weft/ui first. WEFT_VERSION, when set, becomes the Go package's Version.
# Usage: scripts/bundle.sh OUT
set -eu

if [ $# -ne 1 ]; then
	echo "usage: scripts/bundle.sh OUT" >&2
	exit 2
fi
root=$(cd "$(dirname "$0")/.." && pwd)
# Resolve OUT against the caller's folder before anything changes directory.
case "$1" in
/*) out="$1" ;;
*) out="$PWD/$1" ;;
esac

if [ -e "$out" ] && [ ! -d "$out" ]; then
	echo "scripts/bundle.sh: $out is not a folder; name a missing or empty folder." >&2
	exit 1
fi
if [ -d "$out" ] && [ ! -e "$out/.weft-bundle" ]; then
	for f in "$out"/* "$out"/.[!.]* "$out"/..?*; do
		[ -e "$f" ] || continue
		if [ "$(basename "$f")" != README.md ]; then
			echo "scripts/bundle.sh: $out is not empty and is not a bundle; name a missing or empty folder." >&2
			exit 1
		fi
	done
fi

# Mark OUT first, so a run that fails midway can be run again.
mkdir -p "$out"
touch "$out/.weft-bundle"

cd "$root"
pnpm --filter @weft/themes --filter @weft/ui build >&2

# Move what an earlier run wrote into a fresh temporary folder; nothing is deleted.
old=$(mktemp -d)
for d in weft template; do
	if [ -e "$out/$d" ]; then
		mv "$out/$d" "$old/"
		echo "scripts/bundle.sh: the earlier $d is in $old." >&2
	fi
done
mkdir -p "$out/weft/packages" "$out/weft/go" "$out/weft/catalog"

# copy_tree FROM TO copies a folder without what a bundle never needs.
copy_tree() {
	mkdir -p "$2"
	tar -C "$1" -cf - --exclude node_modules --exclude test --exclude examples \
		--exclude '*.docs.md' --exclude '*_test.go' --exclude .DS_Store . | tar -C "$2" -xf -
}

# keep_fields FILE KEY... rewrites a package.json with only the named keys.
keep_fields() {
	node -e '
		const fs = require("node:fs");
		const [file, ...keys] = process.argv.slice(1);
		const pkg = JSON.parse(fs.readFileSync(file, "utf8"));
		const kept = {};
		for (const k of keys) if (k in pkg) kept[k] = pkg[k];
		fs.writeFileSync(file, JSON.stringify(kept, null, 2) + "\n");
	' "$@"
}

fields="name version private type sideEffects exports bin dependencies"

pkg="$out/weft/packages"
mkdir -p "$pkg/ui/dist" "$pkg/themes" "$pkg/vite"
cp packages/ui/package.json "$pkg/ui/"
copy_tree packages/ui/src "$pkg/ui/src"
cp packages/ui/dist/weft.css "$pkg/ui/dist/"
cp packages/themes/package.json packages/themes/tailwind.css "$pkg/themes/"
copy_tree packages/themes/src "$pkg/themes/src"
copy_tree packages/themes/dist "$pkg/themes/dist"
cp packages/vite/package.json "$pkg/vite/"
copy_tree packages/vite/src "$pkg/vite/src"
copy_tree packages/vite/bin "$pkg/vite/bin"
for p in ui themes vite; do
	# shellcheck disable=SC2086
	keep_fields "$pkg/$p/package.json" $fields
done

# warp and every package under it, without tests.
tar -cf - --exclude '*_test.go' --exclude .DS_Store warp | tar -C "$out/weft/go" -xf -
# go:embed leaves out a folder that holds a go.mod (it is another module), so
# the bundle keeps it as go.mod.bundle and weft unpacks it as go.mod.
cp go.mod "$out/weft/go/go.mod.bundle"
cat > "$out/weft/go/version.go" <<GO
// Package weft carries the Weft release a build belongs to; an app's Go code imports github.com/flakerimi/weft/warp.
package weft

// Version is the Weft release this build belongs to.
var Version = "${WEFT_VERSION:-0.0.0-dev}"
GO

cp catalog/components.json "$out/weft/catalog/"

if [ -d templates/app ]; then
	copy_tree templates/app "$out/template"
fi
