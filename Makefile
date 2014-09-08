GO_BINDATA = go-bindata

static-data.go: data
	$(GO_BINDATA) -o static-data.go -pkg spock -tags bundle -ignore '~\z' -prefix data/ data/...
