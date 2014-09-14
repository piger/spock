#!/bin/bash
# OS X build script.

set -e

libicu=$(dirname $(brew list icu4c | fgrep readme.html))
CGO_LDFLAGS="-L${libicu}/lib"
CGO_CFLAGS="-I${libicu}/include"
export CGO_LDFLAGS CGO_CFLAGS

go build -tags "libstemmer icu leveldb" cmd/spock/spock.go
