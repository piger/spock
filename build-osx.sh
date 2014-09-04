#!/bin/bash
# OS X build script.

set -e

LIBICU=$(dirname $(brew list icu4c | fgrep readme.html))
CGO_LDFLAGS="-L${LIBICU}/lib"
CGO_CFLAGS="-I${LIBICU}/include"
DYLD_LIBRARY_PATH="${LIBICU}/lib"
export CGO_LDFLAGS CGO_CFLAGS

go build -tags "libstemmer icu leveldb" cmd/spock/spock.go
