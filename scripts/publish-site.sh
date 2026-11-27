#!/bin/sh
# Build a release and publish weft.base.al on the Lisa server: the pages Weft
# builds from site-src, the installer, every published release (those
# site/releases.txt lists) and the new one, all served by the Weft binary
# (site-src/main.go). The pushed tree has a Dockerfile, so the
# server compiles that binary, which embeds the pages, and runs it as a
# container whose /data/releases holds the installer and the releases.
# Usage: scripts/publish-site.sh [--dry-run] <version>
#   --dry-run  stop before the push and print the push tree's path (in
#              TMPDIR), and builds the site locally
# The release is built in TMPDIR, by a dry run and a real publish alike, and
# copied into site/releases only once pushed, so a publish that fails before
# can be run again with the same version. Temporary folders are never
# deleted here; their paths are printed for the owner to trash.
# Every version in site/releases.txt must be in site/releases, and the version
# published must be neither listed nor in site/releases (building it again
# would change tarballs its users have checksums for). A version is listed in
# site/releases.txt the moment its push succeeds: pushed, it counts as
# published, and every later publish stages it, even if it is never verified
# live. A verified push keeps its tree as site/pushed/<version>/ (git-ignored),
# one per version. Each release a verified push served is kept once, in
# site/pushed/releases/<version>/: the trees hold hard links to it, not copies.
# Rollback: scripts/rollback-site.sh <version> pushes such a kept tree again.
# A publish also tags and releases on GitHub. The annotated tag v<version> is
# made on the commit HEAD names, which the release is built from (tracked
# files must match it), and pushed to origin before the site, so the module
# github.com/flakerimi/weft is at that tag before any app this release makes
# requires it. A tag of that name on the same commit (a publish that stopped
# after pushing it) is used as it is; on another commit it stops the publish
# before anything is built. After the site push, scripts/github-release.sh
# makes the public GitHub Release v<version> from the files the site serves;
# a release of that name stops the publish before anything is built, and gh
# missing or signed out is said after the site push, with the commands to run.
# A dry run checks the tag and the release and prints the git and gh commands
# a publish would run.
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

# --dry-run counts only before the version: one anywhere else, an option this
# script does not know, or a second argument is refused before anything is
# done, so a dry run asked for in the wrong place never publishes.
for arg in "$@"; do
	case "$arg" in -*) fail "$arg is not taken, since no argument but --dry-run may begin with - and --dry-run goes first, before the version; usage: scripts/publish-site.sh [--dry-run] <version>." ;; esac
done
[ $# -le 1 ] ||
	fail "takes one version and nothing after it; usage: scripts/publish-site.sh [--dry-run] <version>."

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

# The tag names the commit the release is built from: the module Go fetches at
# it must be the Go this release's apps are made for. site/releases.txt is
# left out, since a publish writes it and the owner commits it afterwards.
tag="v$version"
commit=$(git rev-parse -q --verify 'HEAD^{commit}') ||
	fail "cannot read the commit HEAD names, which the tag $tag would name; publish from a git checkout of the repository with a commit."
changed=$(git status --porcelain --untracked-files=no -- . ':(exclude)site/releases.txt') ||
	fail "git status cannot read the checkout, so whether $commit is what is built is unknown; fix the checkout and publish again."
if [ -n "$changed" ]; then
	[ -n "$dry" ] ||
		fail "tracked files differ from $commit, which the tag $tag would name, so the module at the tag would not be what this release is built from; commit them, or set them aside, and publish again."
	echo "publish: tracked files differ from $commit, which a publish refuses (the tag $tag would not name what is built); not refused in a dry run" >&2
fi
# Nor may a file git neither tracks nor ignores be in what is built, which
# would be pushed but not be in the tag. What is built is what these read:
# release.sh (go test and go build of the root package, warp and cmd/weft:
# go.mod, go.sum, the root's .go files, warp/, cmd/), bundle.sh (packages/,
# built with the workspace's package.json, pnpm-lock.yaml,
# pnpm-workspace.yaml and tsconfig.base.json; warp/ and version.go for a
# development bundle; catalog/, templates/), the site build (site-src/,
# packages/, catalog/, skills/, and docs/ for the gallery's parity and sizes
# pages, and bench/results/, whose newest file the home page embeds), this
# push tree (site/, the installer), and the scripts/ that do it.
# Elsewhere, such as the owner's notes at the root, an untracked file is no
# matter.
stray=$(git ls-files --others --exclude-standard -- go.mod go.sum ':(glob)*.go' package.json pnpm-lock.yaml pnpm-workspace.yaml tsconfig.base.json \
	warp cmd templates packages catalog site-src skills docs site scripts bench/results) ||
	fail "git ls-files cannot list the files git does not track, so whether $commit is what is built is unknown; fix the checkout and publish again."
if [ -n "$stray" ]; then
	first=$(printf '%s\n' "$stray" | head -n 3 | paste -sd, - | sed 's/,/, /g')
	[ "$(printf '%s\n' "$stray" | wc -l)" -le 3 ] || first="$first, …"
	[ -n "$dry" ] ||
		fail "files git does not track are in what the release and the site are built from ($first), so the tag $tag would not name what is built; commit them, or move them out, and publish again."
	echo "publish: files git does not track are in what the release and the site are built from ($first), which a publish refuses (the tag $tag would not name them); not refused in a dry run" >&2
fi
# A tag of this name: the same commit's is used, another's stops the publish.
tag_here=
tag_there=
if old=$(git rev-parse -q --verify "refs/tags/$tag^{commit}"); then
	[ "$old" = "$commit" ] ||
		fail "the tag $tag already names $old, not $commit, which this publish builds from; publish a new version (a tag the module proxy has served never moves)."
	tag_here=1
fi
if remote=$(git ls-remote origin "refs/tags/$tag" "refs/tags/$tag^{}"); then
	# An annotated tag is listed twice: its own object, and ^{}, its commit.
	peeled=$(printf '%s\n' "$remote" | awk -v t="refs/tags/$tag^{}" '$2 == t { print $1 }')
	[ -n "$peeled" ] || peeled=$(printf '%s\n' "$remote" | awk -v t="refs/tags/$tag" '$2 == t { print $1 }')
	if [ -n "$peeled" ]; then
		[ "$peeled" = "$commit" ] ||
			fail "origin already has the tag $tag on $peeled, not on $commit, which this publish builds from; publish a new version (a tag the module proxy has served never moves)."
		tag_there=1
	fi
elif [ -n "$dry" ]; then
	echo "publish: cannot read the tags of origin, so whether $tag is there is not checked in a dry run" >&2
else
	fail "cannot read the tags of origin (git ls-remote origin failed), so whether $tag is there already is unknown; fix the connection to origin and publish again."
fi
# A GitHub Release of this name stops the publish; gh missing or signed out
# is said after the site push, which does not need it.
# gh is asked of github.com alone: an account elsewhere with a problem would
# make a bare gh auth status fail.
gh_ready=
gh_missing=
gh_todo=
if ! command -v gh >/dev/null 2>&1; then
	gh_missing="gh is not installed"
	gh_todo="install gh (https://cli.github.com), run gh auth login, then scripts/github-release.sh $version"
elif ! gh auth status --hostname github.com >/dev/null 2>&1; then
	gh_missing="gh is not signed in to github.com"
	gh_todo="run gh auth login, then scripts/github-release.sh $version"
else
	gh_ready=1
	# gh release view finds a draft of that name too (gh's FetchRelease looks
	# for a published release by tag and for a draft by tagName).
	! gh release view "$tag" --repo "${WEFT_GITHUB_REPO:-flakerimi/weft}" >/dev/null 2>&1 ||
		fail "the GitHub Release $tag already exists, and a publish never replaces one; publish a new version."
fi
if [ -z "$gh_ready" ] && [ -n "$dry" ]; then
	echo "publish: $gh_missing, so whether the GitHub Release $tag exists is not checked; a publish would say so after the site push" >&2
fi

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

# The release is built in TMPDIR, by a dry run and a real publish alike, and
# copied into site/releases only once it is pushed: until then a failure
# leaves site/releases and site/releases.txt as they were, and the same
# version can be published again. Nothing here deletes the build folder; its
# path is printed for the owner to trash.
tmp=${TMPDIR:-/tmp}
releases=$(mktemp -d "${tmp%/}/weft-release.XXXXXX")
tree=
pushed=
tagged=
stopped() {
	status=$?
	[ "$status" -ne 0 ] && [ -z "$pushed" ] || return 0
	echo "publish: stopped before $version was pushed; site/releases and site/releases.txt are as they were${tagged:+, and the tag $tag is on origin, naming $commit (publishing $version again from that commit uses it)}. This run's release build is in $releases${tree:+ and its push tree in $tree}; trash them when you no longer need them." >&2
}
trap stopped EXIT
scripts/release.sh "$version" "$releases"
# The pages name the release they describe (/index.md).
WEFT_SITE_VERSION="$version" scripts/site.sh build

# The push tree: the root package, warp and every package under it (without
# tests), the site's main package and its build, the installer, every
# published release and this one, and a Dockerfile that compiles the binary
# on an empty image beside the releases. It is made where it will be kept, so
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
tar -cf - --exclude '*_test.go' --exclude .DS_Store warp | tar -C "$tree" -xf -
mkdir "$tree/site-src"
cp site-src/main.go "$tree/site-src/"
cp -R site-src/dist "$tree/site-src/dist"
scripts/stage-releases.sh --link-from "$store" ${unlisted:+--live "$unlisted"} --new-from "$releases" site "$tree/site-src/releases" "$version"
cat >"$tree/Dockerfile" <<EOF
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod *.go ./
COPY warp warp
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
	if [ -n "$tag_here" ]; then
		echo "dry run: the tag $tag names $commit already; a publish would not make it again"
	elif [ -n "$tag_there" ]; then
		echo "dry run: would run, before the site push: git fetch origin refs/tags/$tag:refs/tags/$tag"
	else
		echo "dry run: would run, before the site push: git tag -a $tag -m 'Weft $version' $commit"
	fi
	if [ -n "$tag_there" ]; then
		echo "dry run: origin has the tag $tag on $commit already; a publish would not push it again"
	else
		echo "dry run: would run, before the site push: git push origin refs/tags/$tag"
	fi
	echo "dry run: after the site push, scripts/github-release.sh $version would make the GitHub Release from site/releases/$version, the copy of $releases/$version:"
	scripts/github-release.sh --dry-run "$version" "$releases/$version" | sed "s|$releases/$version/|site/releases/$version/|g"
	echo "dry run: built $version in $releases/$version; site/releases is as it was, so $version can still be published."
	echo "dry run: not pushed; the tree is $tree (trash it and $releases when you no longer need them)"
	exit 0
fi

# The tag first, so the module is at it before any app this release makes
# requires it; a site push that fails leaves it, for the next publish of this
# version from the same commit to use.
# A tag origin has already is fetched, not made here again: a tag object of
# this checkout's own would differ from origin's, and a later git fetch
# --tags would refuse to clobber it.
if [ -z "$tag_here" ] && [ -n "$tag_there" ]; then
	git fetch origin "refs/tags/$tag:refs/tags/$tag" ||
		fail "cannot fetch the tag $tag from origin (see above); fix that and publish again."
elif [ -z "$tag_here" ]; then
	git tag -a "$tag" -m "Weft $version" "$commit" ||
		fail "cannot make the tag $tag on $commit (see above); fix that and publish again."
fi
if [ -z "$tag_there" ]; then
	git push origin "refs/tags/$tag" ||
		fail "cannot push the tag $tag to origin (see above), so nothing is pushed; fix that and publish again, which uses the tag made here."
fi
tagged=1

lisa server push --name weft "$tree"
pushed=1

# Pushed, so published: the site may serve it from now on, and every later
# publish must stage it. Listed at once, before waiting, so a wait that times
# out cannot drop it (nor the live release this push carried).
for v in $unlisted "$version"; do
	scripts/published-releases.sh add site "$v" ||
		fail "$v was pushed, but site/releases.txt could not be written; add the line $v to it by hand, so the next publish keeps $v."
done
# And copied (never moved) into site/releases, as it was pushed, for the next
# publish to stage; the build folder stays for the owner to trash.
if ! { mkdir -p site/releases && cp -R "$releases/$version" "site/releases/$version" &&
	diff -r "$releases/$version" "site/releases/$version" >/dev/null &&
	printf '%s\n' "$version" >site/releases/latest; }; then
	fail "$version was pushed and is listed in site/releases.txt, but its release could not be copied from $releases/$version into site/releases/$version; copy it there by hand, since the next publish stages it from there (and makes the GitHub Release from there: scripts/github-release.sh $version)."
fi

# The GitHub Release, from the files the site serves (site/releases/$version,
# copied from what was pushed and compared with it). The site is live without
# it; a release that could not be made is said now and again at the end.
released=
if [ -z "$gh_ready" ]; then
	echo "publish: $version is pushed and tagged $tag, but $gh_missing, so its GitHub Release is not made; $gh_todo." >&2
elif scripts/github-release.sh "$version"; then
	released=1
fi

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
echo "published weft $version at $url, tagged $tag; listed in site/releases.txt (commit it); its tree is $kept; its release build in $releases can be trashed"
[ -n "$released" ] || [ -z "$gh_ready" ] ||
	fail "$version is live and tagged $tag, but its GitHub Release is not made (see above); fix that and run scripts/github-release.sh $version."
[ -n "$released" ] ||
	fail "$version is live and tagged $tag, but its GitHub Release is not made, since $gh_missing; $gh_todo."
echo "published the GitHub Release $tag"
