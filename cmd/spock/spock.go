// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/sessions"
	"github.com/piger/spock"
	"log"
	"os"
	"os/signal"
	"path/filepath"
)

var (
	address  = flag.String("address", "127.0.0.1:8080", "Bind address")
	repoDir  = flag.String("repo", "", "Path to the git repository")
	initRepo = flag.Bool("init", false, "Initialize a new repository")
	cfgFile  = flag.String("config", "./cfg_spock.json", "Path to the configuration file")
	reIndex  = flag.Bool("reindex", false, "Reindex the wiki")
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

	if len(*repoDir) == 0 {
		fmt.Printf("ERROR: You must specify a repository with -repo\nUsage:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	storage, err := spock.OpenGitStorage(makeAbs(*repoDir), *initRepo)
	if err != nil {
		log.Fatal(err)
	}

	index, err := spock.OpenIndex(makeAbs(*repoDir))
	if err != nil {
		log.Fatal(err)
	}
	// FIXME: there should be a better way to ensure that the index is closed
	// e.g. log.Fatal() skip defers...
	defer index.Close()

	// If we are opening an existing repository and the index is empty we
	// run an initial indexing of the whole repository content.
	if count, err := index.DocCount(); err != nil {
		log.Printf("Error counting documents: %s\n", err)
	} else if (count == 0 && !*initRepo) || *reIndex {
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

	csig := make(chan os.Signal, 1)
	errsig := make(chan error, 1)
	signal.Notify(csig, os.Interrupt)
	go func() {
		errsig <- spock.RunServer(*address, appCtx)
	}()

	select {
	case <-csig:
		log.Printf("Exiting\n")
	case err := <-errsig:
		log.Println(err)
	}
}
