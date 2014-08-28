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
	LookupPage(pagepath string) (*Page, bool, error)
	GetLastCommit(path string) (*CommitLog, error)
	SavePage(page *Page, sig *CommitSignature, message string) error
	ListPages() ([]string, error)
	DiffPage(page *Page, otherSha string) ([]string, error)

	Search(query string) (*ResponseType, error)
	IndexDocument(title string) error
}

type CommitLog struct {
	Id      string
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
