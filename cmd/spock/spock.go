package main

import (
	"flag"
	"github.com/piger/spock"
	"log"
)

var (
	address  = flag.String("address", "127.0.0.1:8080", "Bind address")
	indexSrv = flag.String("indexer", "http://127.0.0.1:5000/api", "Indexer address")
	indexDir = flag.String("index", "./index.bleve", "Index directory")
	repo     = flag.String("repo", ".", "Path to the git repository")
	initRepo = flag.Bool("init", false, "Initialize a new repository")
	dataDir  = flag.String("datadir", "./data", "Path to the data directory")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	storage, err := spock.OpenGitStorage(*repo, *initRepo)
	if err != nil {
		log.Fatal(err)
	}

	index, err := spock.OpenIndex(*indexDir)
	if err != nil {
		log.Fatal(err)
	}
	defer index.Close()

	if index.DocCount() == 0 {
		go func() {
			log.Printf("New index: Indexing all pages\n")
			index.IndexWiki(storage)
			log.Printf("Indexing done!\n")
		}()
	}

	spock.DataDir = *dataDir
	err = spock.RunServer(*address, storage, *indexSrv, index)
	if err != nil {
		log.Fatal(err)
	}
}
