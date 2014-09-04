#!/bin/bash
# Distribution-independent Linux launcher.
#
# Inspired by Joey Hess, author of git-annex:
# http://joeyh.name/blog/entry/completely_linux_distribution-independent_packaging/

set -e

base="$(dirname $0)"

if [[ ! -d $base ]]; then
    echo "ERR: cannot find base directory" >&2
    exit 1
fi

SPOCK_LD_LIBRARY_PATH="${base}/lib/i386-linux-gnu"
SPOCK_LINKER="${base}/lib/ld-linux-i386"

exec "$SPOCK_LINKER" --library-path "$SPOCK_LD_LIBRARY_PATH" \
    "${base}/spock" -datadir "${base}/data" "$@"
