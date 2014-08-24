package spock

import (
	"errors"
	"github.com/piger/git2go"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

var baseGitIgnore string = `*~
*.bak
`

type GitStorage struct {
	WorkDir string
	r       *git.Repository
}

func NewGitStorage(path string) (*GitStorage, error) {
	gitstorage := &GitStorage{WorkDir: path}
	return gitstorage, nil
}

func OpenGitStorage(path string) (*GitStorage, error) {
	repo, err := git.OpenRepository(path)
	if err != nil {
		return nil, err
	}
	gitstorage := &GitStorage{WorkDir: path, r: repo}
	return gitstorage, nil
}

func (gs *GitStorage) MakeAbsPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(gs.WorkDir, path)
}

func (gs *GitStorage) InitRepository() error {
	repo, err := git.InitRepository(gs.WorkDir, false)
	if err != nil {
		return err
	}

	gs.r = repo
	err = gs.seedEmptyRepo()
	if err != nil {
		return err
	}

	return nil
}

func (gs *GitStorage) seedEmptyRepo() error {
	// write file contents
	gitIgnoreFile := filepath.Join(gs.WorkDir, ".gitignore")
	if err := ioutil.WriteFile(gitIgnoreFile, []byte(baseGitIgnore), 0644); err != nil {
		return err
	}

	sig := &git.Signature{
		Name:  "Spock Wiki",
		Email: "spock@localhost",
		When:  time.Now(),
	}

	idx, err := gs.r.Index()
	if err != nil {
		return err
	}
	if err = idx.AddByPath(".gitignore"); err != nil {
		return err
	}
	treeId, err := idx.WriteTree()
	if err != nil {
		return err
	}
	if err = idx.Write(); err != nil {
		return err
	}

	message := "Add .gitignore file\n"
	tree, err := gs.r.LookupTree(treeId)
	if err != nil {
		return err
	}
	_, err = gs.r.CreateCommit("HEAD", sig, sig, message, tree)
	if err != nil {
		return err
	}

	return nil
}

func (gs *GitStorage) CommitFile(path string, signature *CommitSignature, message string) (commitId *git.Oid, treeId *git.Oid, err error) {
	sig := &git.Signature{
		Name:  signature.Name,
		Email: signature.Email,
		When:  signature.When,
	}

	idx, err := gs.r.Index()
	if err != nil {
		return
	}
	// XXX should we "RemoveByPath()" on error condition ?
	if err = idx.AddByPath(path); err != nil {
		return
	}
	treeId, err = idx.WriteTree()
	if err != nil {
		return
	}
	// http://stackoverflow.com/questions/16056759/untracked-dirs-on-commit-with-pygit2
	// We need to also call Write() to avoid leaving "untracked files".
	if err = idx.Write(); err != nil {
		return
	}

	currentBranch, err := gs.r.Head()
	if err != nil {
		return
	}
	currentTip, err := gs.r.LookupCommit(currentBranch.Target())
	if err != nil {
		return
	}

	tree, err := gs.r.LookupTree(treeId)
	if err != nil {
		return
	}
	commitId, err = gs.r.CreateCommit("HEAD", sig, sig, message, tree, currentTip)
	return
}

func (gs *GitStorage) RenamePage(origPath, destPath string, signature *CommitSignature, message string) (commitId *git.Oid, treeId *git.Oid, err error) {
	sig := &git.Signature{
		Name:  signature.Name,
		Email: signature.Email,
		When:  signature.When,
	}

	idx, err := gs.r.Index()
	if err != nil {
		return
	}

	// 1. rename file
	// 2. add renamed file to index
	// 3. remove old file from index (from the docs we see "it may exists")
	// 4. commit
	if err = os.Rename(gs.MakeAbsPath(origPath), gs.MakeAbsPath(destPath)); err != nil {
		return
	}

	if err = idx.AddByPath(destPath); err != nil {
		return
	}
	if err = idx.RemoveByPath(origPath); err != nil {
		return
	}
	treeId, err = idx.WriteTree()
	if err != nil {
		return
	}
	// http://stackoverflow.com/questions/16056759/untracked-dirs-on-commit-with-pygit2
	// We need to also call Write() to avoid leaving "untracked files".
	if err = idx.Write(); err != nil {
		return
	}

	currentBranch, err := gs.r.Head()
	if err != nil {
		return
	}
	currentTip, err := gs.r.LookupCommit(currentBranch.Target())
	if err != nil {
		return
	}

	tree, err := gs.r.LookupTree(treeId)
	if err != nil {
		return
	}
	commitId, err = gs.r.CreateCommit("HEAD", sig, sig, message, tree, currentTip)
	return
}

func (gs *GitStorage) DeletePage(path string, signature *CommitSignature, message string) (commitId *git.Oid, treeId *git.Oid, err error) {
	sig := &git.Signature{
		Name:  signature.Name,
		Email: signature.Email,
		When:  signature.When,
	}

	idx, err := gs.r.Index()
	if err != nil {
		return
	}

	if err = os.Remove(gs.MakeAbsPath(path)); err != nil {
		return
	}

	if err = idx.RemoveByPath(path); err != nil {
		return
	}
	treeId, err = idx.WriteTree()
	if err != nil {
		return
	}
	// http://stackoverflow.com/questions/16056759/untracked-dirs-on-commit-with-pygit2
	// We need to also call Write() to avoid leaving "untracked files".
	if err = idx.Write(); err != nil {
		return
	}

	currentBranch, err := gs.r.Head()
	if err != nil {
		return
	}
	currentTip, err := gs.r.LookupCommit(currentBranch.Target())
	if err != nil {
		return
	}

	tree, err := gs.r.LookupTree(treeId)
	if err != nil {
		return
	}
	commitId, err = gs.r.CreateCommit("HEAD", sig, sig, message, tree, currentTip)

	return
}

func extractCommitLog(commit *git.Commit) *CommitLog {
	author := commit.Author()
	return &CommitLog{Message: commit.Message(), Name: author.Name, Email: author.Email, When: author.When}
}

func (gs *GitStorage) LogsForPage(path string) (result []CommitLog, err error) {
	var oidList []git.Oid
	var commitMap = make(map[git.Oid]*git.Commit)

	walker, err := gs.r.Walk()
	if err != nil {
		return
	}

	if err = walker.PushHead(); err != nil {
		return
	}

	err = walker.Iterate(func(commit *git.Commit) bool {
		tree, err := commit.Tree()
		if err != nil {
			log.Println("LogsForPage: cannot get tree of walked commit:", err)
			return false
		}
		entry, err := tree.EntryByPath(path)
		if err != nil {
			// the requested file does not exists in this tree, stop the walk
			return false
		}

		_, found := commitMap[*entry.Id]
		if !found {
			oidList = append(oidList, *entry.Id)
		}
		commitMap[*entry.Id] = commit

		return true
	})
	if err != nil {
		return
	}

	for _, oid := range oidList {
		commit := commitMap[oid]
		sig := commit.Author()
		cl := CommitLog{
			Message: commit.Message(),
			Name:    sig.Name,
			Email:   sig.Email,
		}
		result = append(result, cl)
	}

	return
}

func (gs *GitStorage) LookupPage(pagepath string) (*Page, error) {
	absbasepath := filepath.Join(gs.WorkDir, pagepath)
	if absbasepath[0:len(gs.WorkDir)] != gs.WorkDir {
		return nil, errors.New("Page path outside of repository directory: " + absbasepath)
	}

	var found bool
	var pageext string
	for _, ext := range PAGE_EXTENSIONS {
		if _, err := os.Stat(absbasepath + "." + ext); err == nil {
			found = true
			pageext = ext
		}
	}

	if !found {
		return nil, nil
	}

	page, err := LoadPage(absbasepath+"."+pageext, pagepath+"."+pageext)
	if err != nil {
		return nil, err
	}

	return page, nil
}

type OidSet struct {
	set map[*git.Oid]bool
}

func NewOidSet() *OidSet {
	return &OidSet{set: make(map[*git.Oid]bool)}
}

func (o *OidSet) Add(oid *git.Oid) bool {
	_, found := o.set[oid]
	o.set[oid] = true
	return !found
}

func (gs *GitStorage) GetLastCommit(path string) (*CommitLog, error) {
	head, err := gs.r.Head()
	if err != nil {
		return nil, err
	}
	commit, err := gs.r.LookupCommit(head.Target())
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	blob, err := tree.EntryByPath(path)
	if err != nil {
		return nil, err
	}

	visited := NewOidSet()
	var queue []*git.Commit
	var cc *git.Commit

	visited.Add(blob.Id)
	stop := false
	queue = append(queue, commit)

	for {
		if len(queue) == 0 {
			break
		}
		cc = queue[0]
		queue = queue[1:]

		var i uint
		for i = 0; i < cc.ParentCount(); i++ {
			parent := cc.Parent(i)
			if parent == nil {
				log.Fatal("parent = nil")
			}
			ptree, err := parent.Tree()
			if err != nil {
				return nil, err
			}
			pblob, err := ptree.EntryByPath(path)
			if err != nil {
				continue
			}

			if !blob.Id.Equal(pblob.Id) {
				stop = true
			} else {
				if rv := visited.Add(parent.TreeId()); rv {
					queue = append(queue, parent)
				}
			}
		}

		if stop {
			break
		}
	}

	return extractCommitLog(cc), nil
}

func (gs *GitStorage) SavePage(page *Page, sig *CommitSignature, message string) error {
	fullpath := filepath.Join(gs.WorkDir, page.Path)

	if err := ioutil.WriteFile(fullpath, page.RawBytes, 0644); err != nil {
		return err
	}

	_, _, err := gs.CommitFile(page.Path, sig, message)
	return err
}
