#!/usr/bin/env bash
# Rewrites literal /forever/ asset paths to $SITE_BASE for a fork's Pages site.
# Build with SITE_BASE set too, so generated URLs, class links and workers use the same path.
# Hosts such as wowhead.com/forever/ are left alone.
set -euo pipefail
: "${SITE_BASE:?set SITE_BASE, e.g. /Forever/next/}"
dir=${1:-dist/forever}
find "$dir" -type f \( -name '*.js' -o -name '*.html' -o -name '*.css' \) -print0 |
	xargs -0 sed -i -E "s#(^|[^A-Za-z0-9_.])/forever/#\1${SITE_BASE}#g"
