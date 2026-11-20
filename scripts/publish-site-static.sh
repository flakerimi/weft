#!/bin/sh
# Rollback: build a release and publish the static site (site/index.html, the
# installer, every real release) to weft.base.al on the Lisa server, as before
# the site was built with Weft (scripts/publish-site.sh).
# Usage: scripts/publish-site-static.sh <version>
set -eu
version="${1:?usage: scripts/publish-site-static.sh <version>}"
root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

scripts/release.sh "$version"

# Stage the page, the installer and every real release (a push replaces the whole site).
stage=$(mktemp -d)
scripts/stage-site.sh site "$stage" "$version"
echo "publishing:"
(cd "$stage" && find . -type f | sort)

lisa server push --name weft "$stage"

got=$(curl -fsSL https://weft.base.al/releases/latest | tr -d ' \r\n')
[ "$got" = "$version" ] || { echo "publish: https://weft.base.al/releases/latest says '$got', expected '$version'" >&2; exit 1; }
echo "published weft $version at https://weft.base.al"
