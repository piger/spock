package main

import (
	"flag"
	"github.com/piger/spock"
	"log"
)

var (
	address  = flag.String("address", "127.0.0.1:8080", "Bind address")
	indexSrv = flag.String("indexer", "http://127.0.0.1:5000/api", "Indexer address")
	repo     = flag.String("repo", ".", "Path to the git repository")
	initRepo = flag.Bool("init", false, "Initialize a new repository")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	storage, err := spock.OpenGitStorage(*repo, *initRepo)
	if err != nil {
		panic(err)
	}

	err = spock.RunServer(*address, storage, *indexSrv)
	if err != nil {
		log.Fatal(err)
	}
}
