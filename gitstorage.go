// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package spock

import (
	"errors"
	"fmt"
	"gopkg.in/libgit2/git2go.v23"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type GitStorage struct {
	WorkDir string
	r       *git.Repository
}

// Create a new git repository, initializing it.
func CreateGitStorage(path string) (*GitStorage, error) {
	repo, err := git.InitRepository(path, false)
	if err != nil {
		return nil, err
	}

	gitstorage := &GitStorage{WorkDir: path, r: repo}
	return gitstorage, nil
}

// Open an existing git repository, optionally creating a new one if the
// specified directory is not found and 'create' is true.
func OpenGitStorage(path string, create bool) (*GitStorage, error) {
	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		if create {
			return CreateGitStorage(path)
		} else {
			return nil, err
		}
	}
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

func (gs *GitStorage) JoinPath(relpath string) (string, error) {
	res := filepath.Join(gs.WorkDir, relpath)
	if !strings.HasPrefix(res, gs.WorkDir) {
		return "", errors.New("Requested file outside repository directory")
	}
	return res, nil
}

// Returns the last (root) commit and tree objects.
func (gs *GitStorage) currentState() (commit *git.Commit, tree *git.Tree, err error) {
	var head *git.Reference
	head, err = gs.r.Head()
	if err != nil {
		return
	}
	commit, err = gs.r.LookupCommit(head.Target())
	if err != nil {
		return
	}
	tree, err = commit.Tree()

	return
}

// Returns true if the git repository has a "root commit" (i.e. the so called
// initial commit).
func (gs *GitStorage) hasRootCommit() bool {
	refname := "refs/heads/master"
	_, err := gs.r.References.Lookup(refname)
	if err != nil {
		return false
	}

	return true
}

func (gs *GitStorage) saveIndex(idx *git.Index, signature *CommitSignature, message string) (*git.Oid, error) {
	sig := &git.Signature{
		Name:  signature.Name,
		Email: signature.Email,
		When:  signature.When,
	}

	treeId, err := idx.WriteTree()
	if err != nil {
		return nil, err
	}

	// http://stackoverflow.com/questions/16056759/untracked-dirs-on-commit-with-pygit2
	// We need to also call Write() to avoid leaving "untracked files".
	if err = idx.Write(); err != nil {
		return nil, err
	}
	tree, err := gs.r.LookupTree(treeId)
	if err != nil {
		return nil, err
	}

	var commitId *git.Oid
	if gs.hasRootCommit() {
		var currentTip *git.Commit

		currentTip, _, err = gs.currentState()
		if err != nil {
			return nil, err
		}

		commitId, err = gs.r.CreateCommit("HEAD", sig, sig, message, tree, currentTip)
	} else {
		commitId, err = gs.r.CreateCommit("HEAD", sig, sig, message, tree)
	}

	return commitId, err
}

func (gs *GitStorage) CommitFile(path string, signature *CommitSignature, message string) (revId RevID, err error) {
	idx, err := gs.r.Index()
	if err != nil {
		return
	}
	// XXX should we "RemoveByPath()" on error condition ?
	if err = idx.AddByPath(path); err != nil {
		return
	}

	commitId, err := gs.saveIndex(idx, signature, message)
	if err != nil {
		return
	}

	revId = RevID(commitId.String())
	return
}

func (gs *GitStorage) RenamePage(origPath, destPath string, signature *CommitSignature, message string) (revId RevID, err error) {
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

	commitId, err := gs.saveIndex(idx, signature, message)
	if err != nil {
		return
	}

	revId = RevID(commitId.String())
	return
}

func (gs *GitStorage) DeletePage(path string, signature *CommitSignature, message string) (revId RevID, err error) {
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

	commitId, err := gs.saveIndex(idx, signature, message)
	if err != nil {
		return
	}

	revId = RevID(commitId.String())

	return
}

func extractCommitLog(commit *git.Commit) *CommitLog {
	author := commit.Author()
	return &CommitLog{
		Message: commit.Message(),
		Name:    author.Name,
		Email:   author.Email,
		When:    author.When,
		Id:      commit.Id().String(),
	}
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
		cl := extractCommitLog(commit)
		result = append(result, *cl)
	}

	return
}

// LookupPage is used to fetch pages from the wiki storage. The "relpath"
// argument refers to a page path relative to the root of the wiki; for
// example "notes/Linux" and "/notes/Linux" both refers to the "Linux"
// page found inside the "/notes" wiki directory. Remember that wiki
// pages are qualified by their "wiki filename", that is the page filename
// without the extension.
//
// Paths are checked for directory traversal attacks and an error will be
// returned if the "relpath" argument refers to a file outside the wiki
// repository.
func (gs *GitStorage) LookupPage(relpath string) (*Page, bool, error) {
	abspath, err := gs.JoinPath(relpath)
	if err != nil {
		return nil, false, err
	}

	for i := range PAGE_EXTENSIONS {
		filename := abspath + "." + PAGE_EXTENSIONS[i]
		relfilename := relpath + "." + PAGE_EXTENSIONS[i]
		if _, err := os.Stat(filename); err == nil {
			page, err := LoadPage(filename, relfilename)
			if err != nil {
				return nil, true, err
			}
			return page, true, nil
		}
	}
	filename := relpath + "." + DefaultExtension
	page := NewPage(filename)
	return page, false, nil
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
	commit, tree, err := gs.currentState()
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

	if err := MkMissingDirs(fullpath); err != nil {
		return err
	}

	if err := ioutil.WriteFile(fullpath, page.RawBytes, 0644); err != nil {
		return err
	}

	_, err := gs.CommitFile(page.Path, sig, message)
	return err
}

func (gs *GitStorage) ListPages() ([]string, error) {
	var result []string

	// Return early if we are on a new repository (i.e. one without a "root"
	// commit).
	if !gs.hasRootCommit() {
		return result, nil
	}

	exts := make(map[string]bool)
	for _, ext := range PAGE_EXTENSIONS {
		exts["."+ext] = true
	}

	_, tree, err := gs.currentState()
	if err != nil {
		return result, err
	}

	err = tree.Walk(func(root string, t *git.TreeEntry) int {
		switch git.Filemode(t.Filemode) {
		case git.FilemodeBlob, git.FilemodeBlobExecutable:
			pageext := filepath.Ext(t.Name)
			if len(pageext) > 0 {
				if _, ok := exts[pageext]; ok {
					result = append(result, ShortenPageName(root+t.Name))
				}
			}
		}

		// to avoid going into sibdirectories return 1
		return 0
	})
	return result, err
}

// Return the git.Tree of the specified SHA id.
func (gs *GitStorage) treeFromId(id string) (*git.Tree, error) {
	oid, err := git.NewOid(id)
	if err != nil {
		return nil, err
	}
	commit, err := gs.r.LookupCommit(oid)
	if err != nil {
		return nil, err
	}
	return commit.Tree()
}

func (gs *GitStorage) DiffPage(page *Page, revA, revB string) ([]string, error) {
	// commit A
	oldTree, err := gs.treeFromId(revA)
	if err != nil {
		return nil, err
	}

	// commit B
	newTree, err := gs.treeFromId(revB)
	if err != nil {
		return nil, err
	}

	// run git diff
	diffopts, err := git.DefaultDiffOptions()
	if err != nil {
		return nil, err
	}
	diff, err := gs.r.DiffTreeToTree(newTree, oldTree, &diffopts)
	if err != nil {
		return nil, err
	}

	// we can't know in advance how many deltas are useful to us inside this diff.
	result := make([]string, 0)

	dlen, err := diff.NumDeltas()
	if err != nil {
		return nil, err
	}
	for i := 0; i < dlen; i++ {
		delta, err := diff.GetDelta(i)
		if err != nil {
			return nil, err
		}

		// skip patches for other files
		if delta.OldFile.Path != page.Path {
			continue
		}

		patch, err := diff.Patch(i)
		if err != nil {
			return nil, err
		}
		if patchStr, err := patch.String(); err == nil {
			result = append(result, patchStr)
		} else {
			fmt.Print(err)
		}
	}

	return result, nil
}
