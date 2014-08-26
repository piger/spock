package main

import (
	"flag"
	"github.com/piger/spock"
	"log"
)

var (
	address  = flag.String("address", "127.0.0.1:8080", "Bind address")
	indexSrv = flag.String("indexer", "127.0.0.1:5000", "Indexer address")
	repo     = flag.String("repo", ".", "Path to the git repository")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	storage, err := spock.OpenGitStorage(*repo)
	if err != nil {
		panic(err)
	}

	err = spock.RunServer(*address, storage)
	if err != nil {
		log.Fatal(err)
	}
}
