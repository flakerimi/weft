#!/bin/sh
# Build a release and publish weft.base.al on the Lisa server: the pages Weft
# builds from site-src, the installer, every published release (those
# site/releases.txt lists) and the new one, all served by the Weft binary
# (site-src/main.go). The pushed tree has a Dockerfile, so the
# server compiles that binary, which embeds the pages, and runs it as a
# container whose /data/releases holds the installer and the releases.
# Usage: scripts/publish-site.sh [--dry-run] <version>
#   --dry-run  stop before the push and print the push tree's path (in
#              TMPDIR); it builds the release in TMPDIR too, leaving
#              site/releases as it was, and the site locally
# Every version in site/releases.txt must be in site/releases, and the version
# published must be neither listed nor in site/releases (building it again
# would change tarballs its users have checksums for). A version is listed in
# site/releases.txt the moment its push succeeds: pushed, it counts as
# published, and every later publish stages it, even if it is never verified
# live. A verified push keeps its tree as site/pushed/<version>/ (git-ignored),
# one per version. Each release a verified push served is kept once, in
# site/pushed/releases/<version>/: the trees hold hard links to it, not copies.
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
# Nor is a version built before rebuilt, whoever built it: its tarballs may be out there.
[ ! -e "site/releases/$version" ] ||
	fail "$version is already released (site/releases/$version exists), and building it again would change its tarballs; publish a new version."
[ ! -e "site/pushed/$version" ] ||
	fail "site/pushed/$version already holds a pushed tree, so $version was pushed before; publish a new version, or roll back with scripts/rollback-site.sh $version."

# A push replaces the whole site: every release it has published must be staged again.
scripts/published-releases.sh check site || exit 1

# The release the site serves now must be staged too, even if the list missed
# it: a publish from before pushes were listed at once may have pushed it.
live=$(curl -fsSL "$url/releases/latest" 2>/dev/null | tr -d ' \r\n') || live=
unlisted=
if scripts/version.sh "$live"; then
	[ "$live" = "$version" ] || [ -f "site/releases/$live/SHA256SUMS" ] ||
		fail "$url serves release $live, which site/releases does not have; fetch it (SHA256SUMS and the files it lists, from $url/releases/$live/) before publishing"
	if [ "$live" != "$version" ] && ! grep -qxF "$live" site/releases.txt; then
		unlisted=$live
		echo "publish: $url serves $live, which was pushed earlier without being listed in site/releases.txt; it is staged so its users keep it, and listed when $version is pushed." >&2
	fi
elif [ -n "$dry" ]; then
	echo "publish: cannot read the live release from $url/releases/latest; not checked in a dry run" >&2
else
	fail "cannot read the live release from $url/releases/latest, so cannot check that it is kept"
fi

# And as a verified push served them: checked now, before anything is built or a tree made.
store=site/pushed/releases
scripts/stage-releases.sh --link-from "$store" ${unlisted:+--live "$unlisted"} --check site || exit 1

# A dry run builds the release in TMPDIR, leaving site/releases as it was, so
# the version can still be published for real.
tmp=${TMPDIR:-/tmp}
if [ -n "$dry" ]; then
	releases=$(mktemp -d "${tmp%/}/weft-dry-release.XXXXXX")
	scripts/release.sh "$version" "$releases"
else
	releases=
	scripts/release.sh "$version"
fi
scripts/site.sh build

# The push tree: the weft package, the site's main package and its build, the
# installer, every published release and this one, and a Dockerfile that
# compiles the binary on an empty image beside the releases. It is made where it will be kept, so
# a release kept before is a hard link to it there; a dry run's goes to TMPDIR.
if [ -n "$dry" ]; then
	tree=$(mktemp -d "${tmp%/}/weft-site-push.XXXXXX")
else
	mkdir -p site/pushed
	tree=$(mktemp -d "$root/site/pushed/.weft-site-push.XXXXXX")
fi
cp go.mod "$tree/"
for f in *.go; do
	case "$f" in *_test.go) continue ;; esac
	cp "$f" "$tree/"
done
mkdir "$tree/site-src"
cp site-src/main.go "$tree/site-src/"
cp -R site-src/dist "$tree/site-src/dist"
scripts/stage-releases.sh --link-from "$store" ${unlisted:+--live "$unlisted"} ${releases:+--new-from "$releases"} site "$tree/site-src/releases" "$version"
cat >"$tree/Dockerfile" <<EOF
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod *.go ./
COPY site-src/main.go site-src/
COPY site-src/dist site-src/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X github.com/flakerimi/weft.Version=$version" -o /out/weft-site ./site-src
# Readable by the server's user whatever the umask the tree was staged under.
COPY site-src/releases /data/releases
RUN chmod -R u=rwX,go=rX /data/releases

FROM scratch
COPY --from=build /out/weft-site /weft-site
COPY --from=build /data/releases /data/releases
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/weft-site"]
EOF
echo "push tree:"
(cd "$tree" && find . -type f | sort)

if [ -n "$dry" ]; then
	echo "dry run: built $version in $releases/$version; site/releases is as it was, so $version can still be published."
	echo "dry run: not pushed; the tree is $tree"
	exit 0
fi

lisa server push --name weft "$tree"

# Pushed, so published: the site may serve it from now on, and every later
# publish must stage it. Listed at once, before waiting, so a wait that times
# out cannot drop it (nor the live release this push carried).
[ -z "$unlisted" ] || scripts/published-releases.sh add site "$unlisted"
scripts/published-releases.sh add site "$version"

# The server builds the image after the push; wait for the new site to answer.
scripts/wait-live.sh "$version" ||
	fail "$version was pushed and is listed in site/releases.txt (commit it), so the next publish stages it, but it is not verified live (see above); its tree stays in $tree: once $url serves $version, move it to site/pushed/$version for a rollback."

# Only now, with the push verified, is its tree kept for a rollback.
kept=site/pushed/$version
mv "$tree" "$kept" ||
	fail "$version is live and listed in site/releases.txt, but its tree $tree could not be kept as $kept; move the tree there for a rollback."

# Each release the tree holds is kept once, beside the trees, for the next
# push to link: hard links to the tree's files, read-only so that a write
# through any link fails, made in a folder of its own that takes its name
# only when it is whole.
keep_releases() {
	for dir in "$kept"/site-src/releases/releases/*/; do
		[ -d "$dir" ] || continue
		v=$(basename "$dir")
		[ ! -e "$store/$v" ] || continue
		mkdir -p "$store" || return 1
		part=$(mktemp -d "$store/.$v.XXXXXX") && chmod 755 "$part" || return 1
		scripts/link-release.sh "$dir" "$part" || return 1
		diff -r "$dir" "$part" >/dev/null || return 1
		chmod a-w "$part"/* || return 1
		mv "$part" "$store/$v" || return 1
	done
}
keep_releases ||
	echo "publish: $kept holds every release it served, but not all could be kept once in $store; the next publish copies those again." >&2
echo "published weft $version at $url; listed in site/releases.txt (commit it); its tree is $kept"
