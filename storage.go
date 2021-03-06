// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package spock

// Abstract interface to a VCS used as a wiki storage.

import (
	"time"
)

// Valid page extensions.
var PAGE_EXTENSIONS = []string{"md", "rst", "org", "txt"}

// This is the interface to the version control system used as a backend.
type Storage interface {
	JoinPath(relpath string) (string, error)

	// Lookup a single Page
	LookupPage(pagepath string) (*Page, bool, error)

	// CRUD
	RenamePage(origPath, destPath string, signature *CommitSignature, message string) (RevID, error)
	DeletePage(path string, signature *CommitSignature, message string) (RevID, error)
	SavePage(page *Page, sig *CommitSignature, message string) error

	// Get the commit logs for a Page.
	LogsForPage(path string) ([]CommitLog, error)

	// Get the last commit for a single Page. (deprecate?)
	GetLastCommit(path string) (*CommitLog, error)

	// List all the pages inside this Storage.
	ListPages() ([]string, error)

	// Returns a diff between the current page content and another revision. (rewrite?)
	DiffPage(page *Page, revA, revB string) ([]string, error)
}

// The struct used to pack all informations regarding a single VCS commit.
type CommitLog struct {
	Id      string
	Message string
	Name    string
	Email   string
	When    time.Time
}

// The struct used when creating a new commit
type CommitSignature struct {
	Name  string
	Email string
	When  time.Time
}

// RevID represents a revision id; with git is the SHA hash of a commit.
type RevID string
