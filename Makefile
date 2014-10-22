# Spock - GNU Makefile

# Use libicu installation from homebrew
ifeq ($(shell uname), Darwin)
export CGO_LDFLAGS := -L/usr/local/opt/icu4c/lib 
export CGO_CFLAGS := -I/usr/local/opt/icu4c/include
endif

GO_RICE := $(GOPATH)/bin/rice
GO_FILES := *.go cmd/spock/spock.go
GO_PACKAGE := github.com/piger/spock
BUILD_TAGS := "libstemmer icu leveldb"

all: spock

install: $(GO_FILES) $(GO_RICE)
	go install -tags $(BUILD_TAGS) $(GO_PACKAGE)/cmd/spock
	$(GO_RICE) append --exec $(GOPATH)/bin/spock

spock: $(GO_FILES) $(GO_RICE)
	go build -tags $(BUILD_TAGS) cmd/spock/spock.go

clean:
	test -f spock && rm spock

# Add "-w" to ldflags to skip debug informations.
release: clean $(GO_RICE)
	go build -ldflags "-w" -tags $(BUILD_TAGS) cmd/spock/spock.go
	$(GO_RICE) append --exec spock

$(GO_RICE):
	go get github.com/GeertJohan/go.rice
	go get github.com/GeertJohan/go.rice/rice

.PHONY: install clean release
