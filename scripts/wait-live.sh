#!/bin/sh
# Wait, after a push, until weft.base.al serves VERSION from the Weft-built
# page: the server builds the image after the push, which takes a while. Fails
# in one sentence after 5 minutes.
# Usage: scripts/wait-live.sh VERSION
set -eu
version="${1:?usage: scripts/wait-live.sh VERSION}"
url=https://weft.base.al
tries=0
while :; do
	got=$(curl -fsSL "$url/releases/latest" 2>/dev/null | tr -d ' \r\n') || got=
	home=$(curl -fsSL "$url/" 2>/dev/null) || home=
	case "$home" in *'<weft-island data-id="CopyCommand"'*) built=1 ;; *) built= ;; esac
	[ "$got" = "$version" ] && [ -n "$built" ] && exit 0
	tries=$((tries + 1))
	if [ "$tries" -ge 60 ]; then
		echo "wait-live: after 5 minutes $url/releases/latest says '$got', expected '$version'$([ -n "$built" ] || echo ', and / is not the Weft-built page'); see lisa server logs weft" >&2
		exit 1
	fi
	sleep 5
done
