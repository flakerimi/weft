#!/bin/sh
# Build a release and publish weft.base.al on the Lisa server: the pages Weft
# builds from site-src, the installer and every real release, all served by the
# Weft binary (site-src/main.go). The pushed tree has a Dockerfile, so the
# server compiles that binary and runs it as a container.
# Usage: scripts/publish-site.sh [--dry-run] <version>
#   --dry-run  stop before the push and print the push tree's path; it still
#              builds the release (site/releases/<version>) and the site locally
# Every version in site/releases.txt must be in site/releases, and the version
# published must not be one of them (building it again would change tarballs
# its users have checksums for). A verified push adds it to that list and keeps
# its tree as site/pushed/<version>/ (git-ignored), one per version.
# Rollback: scripts/rollback-site.sh <version> pushes such a kept tree again.
set -eu
dry=
if [ "${1:-}" = --dry-run ]; then
	dry=1
	shift
fi
version="${1:-}"
root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
url=https://weft.base.al

fail() {
	echo "publish: $1" >&2
	exit 1
}

# The version goes into the Dockerfile's -ldflags: nothing but a version may.
scripts/version.sh "$version" ||
	fail "give a version like 0.0.1 or 0.1.0-alpha.1 (one line of digits, dots, lowercase letters and hyphens); usage: scripts/publish-site.sh [--dry-run] <version>."

# A published version is never built again; its kept tree is never replaced.
grep -qxF "$version" site/releases.txt 2>/dev/null &&
	fail "$version is already published (site/releases.txt lists it), and building it again would change its tarballs; publish a new version, or roll back with scripts/rollback-site.sh $version."
[ ! -e "site/pushed/$version" ] ||
	fail "site/pushed/$version already holds a pushed tree, so $version was pushed before; publish a new version, or roll back with scripts/rollback-site.sh $version."

# A push replaces the whole site: every release it has published must be staged again.
scripts/published-releases.sh check site || exit 1

# The release the site serves now must be staged too, even if the list missed it.
live=$(curl -fsSL "$url/releases/latest" 2>/dev/null | tr -d ' \r\n') || live=
if scripts/version.sh "$live"; then
	[ "$live" = "$version" ] || [ -f "site/releases/$live/SHA256SUMS" ] ||
		fail "$url serves release $live, which site/releases does not have; fetch it (SHA256SUMS and the files it lists, from $url/releases/$live/) before publishing"
elif [ -n "$dry" ]; then
	echo "publish: cannot read the live release from $url/releases/latest; not checked in a dry run" >&2
else
	fail "cannot read the live release from $url/releases/latest, so cannot check that it is kept"
fi

scripts/release.sh "$version"
# Stage the installer and every real release into site-src/public, then build the pages.
scripts/site.sh build "$version"

# The push tree: the weft package, the site's main package and its build, and
# a Dockerfile that compiles them into a binary on an empty image.
tmp=${TMPDIR:-/tmp}
tree=$(mktemp -d "${tmp%/}/weft-site-push.XXXXXX")
cp go.mod "$tree/"
for f in *.go; do
	case "$f" in *_test.go) continue ;; esac
	cp "$f" "$tree/"
done
mkdir "$tree/site-src"
cp site-src/main.go "$tree/site-src/"
cp -R site-src/dist "$tree/site-src/dist"
cat >"$tree/Dockerfile" <<EOF
FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X github.com/flakerimi/weft.Version=$version" -o /out/weft-site ./site-src

FROM scratch
COPY --from=build /out/weft-site /weft-site
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/weft-site"]
EOF
echo "push tree:"
(cd "$tree" && find . -type f | sort)

if [ -n "$dry" ]; then
	echo "dry run: not pushed; the tree is $tree"
	exit 0
fi

lisa server push --name weft "$tree"

# The server builds the image after the push; wait for the new site to answer.
scripts/wait-live.sh "$version" || fail "the push of $version is not verified (see above), so it is not listed as published; its tree stays in $tree."
# Only now, with the push verified, is the version listed as published, and
# its tree kept for a rollback.
scripts/published-releases.sh add site "$version"
mkdir -p site/pushed
mv "$tree" "site/pushed/$version"
echo "published weft $version at $url; listed in site/releases.txt (commit it); its tree is site/pushed/$version"
