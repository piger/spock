# Spock GNU Makefile

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

GO_RICE := $(GOPATH)/bin/rice
GO_FILES := $(shell find . -name '*.go')
GO_PACKAGE := github.com/piger/spock
BUILD_TAGS := "libstemmer icu leveldb"

all: install

install: $(GO_FILES) $(GO_RICE)
	go install -tags $(BUILD_TAGS) $(GO_PACKAGE)/cmd/spock
	$(GO_RICE) append --exec $(GOPATH)/bin/spock

spock: $(GO_FILES) $(GO_RICE)
	go build -tags $(BUILD_TAGS) cmd/spock/spock.go
	$(GO_RICE) append --exec spock

$(GO_RICE):
	go get github.com/GeertJohan/go.rice
	go get github.com/GeertJohan/go.rice/rice

.PHONY: install
