package spock

import (
	"github.com/piger/git2go"
)

type Storage interface {
	CommitFile(path string, signature *CommitSignature, message string) (*git.Oid, *git.Oid, error)
	RenamePage(origPath, destPath string, signature *CommitSignature, message string) (*git.Oid, *git.Oid, error)
	DeletePage(path string, signature *CommitSignature, message string) (*git.Oid, *git.Oid, error)
	LogsForPage(path string) ([]CommitLog, error)
}
