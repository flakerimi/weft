#!/bin/sh
# Put each file of the release folder FROM into the folder TO (made if
# missing) as a hard link, or as a copy where the file system cannot link,
# so a release is kept once however many trees hold it. Fails, leaving TO as
# far as it got, if a file can be neither linked nor copied.
# Usage: scripts/link-release.sh FROM TO
set -eu
from="$1"
to="$2"
mkdir -p "$to"
for f in "$from"/* "$from"/.[!.]* "$from"/..?*; do
	[ -e "$f" ] || continue
	ln "$f" "$to/" 2>/dev/null || cp -R "$f" "$to/"
done
