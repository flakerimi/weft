#!/bin/sh
# Build a release and publish the site (page, installer, that release) to weft.base.al
# on the Lisa server. Usage: scripts/publish-site.sh <version>
set -eu
version="${1:?usage: scripts/publish-site.sh <version>}"
root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

scripts/release.sh "$version"

# Stage only what the site serves: the page, the installer and this release.
stage=$(mktemp -d)
cp site/index.html site/install "$stage/"
mkdir -p "$stage/releases"
cp -R "site/releases/$version" "$stage/releases/"
printf '%s\n' "$version" >"$stage/releases/latest"
echo "publishing:"
(cd "$stage" && find . -type f | sort)

lisa server push --name weft "$stage"

got=$(curl -fsSL https://weft.base.al/releases/latest | tr -d ' \r\n')
[ "$got" = "$version" ] || { echo "publish: https://weft.base.al/releases/latest says '$got', expected '$version'" >&2; exit 1; }
echo "published weft $version at https://weft.base.al"
