GO_BINDATA := go-bindata
GO_FILES := $(shell find . -name '*.go')

all: spock

spock: $(GO_FILES)
	./build-osx.sh

static-data.go:
	$(GO_BINDATA) -o static-data.go -pkg spock -tags bundle -ignore '~\z' -prefix data/ data/...

static-data.tar.gz:
	tar zcvf static-data.tar.gz data/

.PHONY: static-data.go static-data.tar.gz
