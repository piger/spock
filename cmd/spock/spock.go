package main

import (
	"flag"
	"github.com/gorilla/sessions"
	"github.com/piger/spock"
	"log"
)

var (
	address  = flag.String("address", "127.0.0.1:8080", "Bind address")
	indexDir = flag.String("index", "./index.bleve", "Index directory")
	repo     = flag.String("repo", ".", "Path to the git repository")
	initRepo = flag.Bool("init", false, "Initialize a new repository")
	dataDir  = flag.String("datadir", "./data", "Path to the data directory")
	cfgFile  = flag.String("config", "./cfg_spock.json", "Path to the configuration file")
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
			err = index.IndexWiki(storage)
			if err != nil {
				log.Printf("Error running the initial indexing: %s\n", err)
				log.Printf("You can ignore this error if using a new repository\n")
			} else {
				log.Printf("Indexing done!\n")
			}
		}()
	}

	cfg, err := spock.NewConfiguration(*cfgFile)
	if err != nil {
		log.Fatal(err)
	}
	if !cfg.Validate() {
		log.Fatal("Invalid configuration file: check 'secret_key' value!")
	}

	// setup application context
	appCtx := &spock.AppContext{
		SessionStore: sessions.NewCookieStore([]byte(cfg.SecretKey)),
		XsrfSecret:   cfg.SecretKey,
		Storage:      storage,
		Index:        *index,
	}

	// XXX this is really ugly
	spock.DataDir = *dataDir

	err = spock.RunServer(*address, appCtx)
	if err != nil {
		log.Fatal(err)
	}
}
