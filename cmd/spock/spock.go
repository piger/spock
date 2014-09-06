package main

import (
	"flag"
	"github.com/gorilla/sessions"
	"github.com/piger/spock"
	"log"
	"path/filepath"
)

var (
	address  = flag.String("address", "127.0.0.1:8080", "Bind address")
	indexDir = flag.String("index", "./index.bleve", "Index directory")
	repoDir  = flag.String("repo", ".", "Path to the git repository")
	dataDir  = flag.String("datadir", "./data", "Path to the data directory")
	initRepo = flag.Bool("init", false, "Initialize a new repository")
	cfgFile  = flag.String("config", "./cfg_spock.json", "Path to the configuration file")
)

func makeAbs(p string) string {
	rv, err := filepath.Abs(p)
	if err != nil {
		log.Fatalf("Cannot get absolute path for %s: %s\n", p, err)
	}
	return rv
}

func main() {
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	storage, err := spock.OpenGitStorage(makeAbs(*repoDir), *initRepo)
	if err != nil {
		log.Fatal(err)
	}

	index, err := spock.OpenIndex(makeAbs(*indexDir))
	if err != nil {
		log.Fatal(err)
	}
	defer index.Close()

	// If we are opening an existing repository and the index is empty we
	// run an initial indexing of the whole repository content.
	if index.DocCount() == 0 && !*initRepo {
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
	spock.DataDir = makeAbs(*dataDir)

	err = spock.RunServer(*address, appCtx)
	if err != nil {
		log.Fatal(err)
	}
}
