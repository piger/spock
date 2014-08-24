package spock

import (
	"github.com/piger/git2go"
	"time"
)

var PAGE_EXTENSIONS = []string{"md", "rst", "txt"}

type Storage interface {
	CommitFile(path string, signature *CommitSignature, message string) (*git.Oid, *git.Oid, error)
	RenamePage(origPath, destPath string, signature *CommitSignature, message string) (*git.Oid, *git.Oid, error)
	DeletePage(path string, signature *CommitSignature, message string) (*git.Oid, *git.Oid, error)
	LogsForPage(path string) ([]CommitLog, error)
	LookupPage(pagepath string) (*Page, error)
	GetLastCommit(path string) (*CommitLog, error)
	SavePage(page *Page, sig *CommitSignature, message string) error
}

type CommitLog struct {
	Message string
	Name    string
	Email   string
	When    time.Time
}

type CommitSignature struct {
	Name  string
	Email string
	When  time.Time
}
