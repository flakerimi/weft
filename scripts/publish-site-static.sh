#!/bin/sh
# This was the rollback to the static weft.base.al. It cannot be one any more:
# the sentence below says why, and what the rollback is now.
echo "publish-site-static: weft is a container app on the Lisa server, which keeps an app's type, so a static push cannot replace it; roll back with scripts/rollback-site.sh <version>, which pushes again the tree scripts/publish-site.sh kept in site/pushed/<version>/." >&2
exit 1
