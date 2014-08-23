package spock

import (
	"github.com/libgit2/git2go"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"time"
)

var GitCommandName string = "git"

var baseGitIgnore string = `*~
*.bak
`

type GitStorage struct {
	WorkDir string
	GitDir  string
	GitBin  string
	r       *git.Repository
}

func NewGitStorage(path string) (*GitStorage, error) {
	gitBin, err := exec.LookPath(GitCommandName)
	if err != nil {
		return nil, err
	}

	gitDir := filepath.Join(path, ".git")

	gitstorage := &GitStorage{
		WorkDir: path,
		GitDir:  gitDir,
		GitBin:  gitBin,
	}

	return gitstorage, nil
}

func (gs *GitStorage) MakeAbsPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(gs.WorkDir, path)
}

func (gs *GitStorage) RunGitCommand(command string, args ...string) (output string, err error) {
	var cmdArgs []string = []string{"-C", gs.WorkDir, command}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(gs.GitBin, cmdArgs...)
	var out []byte
	out, err = cmd.CombinedOutput()
	output = string(out)
	return
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

func (gs *GitStorage) CommitFile(path, authorName, authorEmail, message string) (commitId *git.Oid, treeId *git.Oid, err error) {
	sig := &git.Signature{
		Name:  authorName,
		Email: authorEmail,
		When:  time.Now(),
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

func (gs *GitStorage) RenamePage(origPath, destPath string) (string, error) {
	output, err := gs.RunGitCommand("mv", origPath, destPath)
	return output, err
}

type CommitLog struct {
	Message string
}

func (gs *GitStorage) LogsForPage(path string, limit int) (result []CommitLog, err error) {
	// XXX missing support for limit parameter!
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
		result = append(result, CommitLog{Message: commit.Message()})
	}

	return
}
