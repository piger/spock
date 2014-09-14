# Spock GNU Makefile
#
# To build the "bundle" version you'll need `go-bindata`:
# https://github.com/jteeuwen/go-bindata

UNAME := $(shell uname)

# ugly OS X hack needed to build bleve with an updated version of libicu
ifeq ($(UNAME), Darwin)
ifeq ($(shell brew list icu4c > /dev/null 2>&1; echo $$?), 1)
$(error You must install icu4c with homebrew)
endif
LIBICU_PATH := $(shell dirname `brew list icu4c | fgrep readme.html`)
export CGO_LDFLAGS := -L$(LIBICU_PATH)/lib
export CGO_CFLAGS := -I$(LIBICU_PATH)/include
endif

GO_BINDATA := go-bindata
GO_FILES := $(shell find . -name '*.go' \! -name static-data.go)
SPOCK_CMD := cmd/spock/spock.go

all: spock

spock: $(GO_FILES)
	go build -tags "libstemmer icu leveldb" $(SPOCK_CMD)

spock-bundle: static-data.go $(GO_FILES)
	go build -tags "libstemmer icu leveldb bundle" $(SPOCK_CMD)

static-data.go:
	$(GO_BINDATA) -o static-data.go -pkg spock -tags bundle -ignore '~\z' -prefix data/ data/...

static-data.tar.gz:
	tar zcvf static-data.tar.gz data/

.PHONY: static-data.go static-data.tar.gz spock-bundle
