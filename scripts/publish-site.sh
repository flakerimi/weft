#!/bin/sh
# Build a release and publish weft.base.al on the Lisa server: the pages Weft
# builds from site-src, the installer and every real release, all served by the
# Weft binary (site-src/main.go). The pushed tree has a Dockerfile, so the
# server compiles that binary and runs it as a container.
# Usage: scripts/publish-site.sh [--dry-run] <version>
#   --dry-run  stop before the push and print the push tree's path; it still
#              builds the release (site/releases/<version>) and the site locally
# Every version in site/releases.txt must be in site/releases; a verified push
# adds the new one to that list.
# Rollback to the static site: scripts/publish-site-static.sh <version>
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
tries=0
while :; do
	got=$(curl -fsSL "$url/releases/latest" 2>/dev/null | tr -d ' \r\n') || got=
	home=$(curl -fsSL "$url/" 2>/dev/null) || home=
	case "$home" in *'<weft-island data-id="CopyCommand"'*) built=1 ;; *) built= ;; esac
	[ "$got" = "$version" ] && [ -n "$built" ] && break
	tries=$((tries + 1))
	[ "$tries" -lt 60 ] || fail "after 5 minutes $url/releases/latest says '$got', expected '$version'$([ -n "$built" ] || echo ', and / is not the Weft-built page'); see lisa server logs weft"
	sleep 5
done
# Only now, with the push verified, is the version listed as published.
scripts/published-releases.sh add site "$version"
echo "published weft $version at $url; listed in site/releases.txt (commit it)"
