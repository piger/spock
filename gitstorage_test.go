package spock

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

var testPageContent string = `---
title: "Index page"
description: "The index page"
language: "it"
---
# Index

Hello world!
`

func checkFatal(t *testing.T, err error) {
	if err == nil {
		return
	}

	_, file, line, ok := runtime.Caller(1)
	if !ok {
		t.Fatal()
	}

	t.Fatalf("Fail at %v:%v; %v", file, line, err)
}

func createTestRepo(t *testing.T) *GitStorage {
	path, err := ioutil.TempDir("", "spock")
	checkFatal(t, err)

	repo, err := NewGitStorage(path)
	checkFatal(t, err)
	err = repo.InitRepository()
	checkFatal(t, err)

	return repo
}

func cleanup(t *testing.T, gs *GitStorage) {
	err := os.RemoveAll(gs.WorkDir)
	checkFatal(t, err)
}

func createIndexPage(t *testing.T, gs *GitStorage) string {
	pageName := "index.md"
	pagePath := filepath.Join(gs.WorkDir, pageName)
	checkFatal(t, ioutil.WriteFile(pagePath, []byte(testPageContent), 0644))
	return pageName
}

func TestNewGitStorage(t *testing.T) {
	gs, err := NewGitStorage("/tmp/foo")
	checkFatal(t, err)
	exGitDir := filepath.Join("/tmp/foo", ".git")

	if gs.GitDir != exGitDir {
		t.Fatalf("GitDir should be %s, is %s", exGitDir, gs.GitDir)
	}
}

func TestInitRepository(t *testing.T) {
	gs := createTestRepo(t)
	defer cleanup(t, gs)
}

func TestCommitFile(t *testing.T) {
	gs := createTestRepo(t)
	defer cleanup(t, gs)

	pageName := createIndexPage(t, gs)

	_, _, err := gs.CommitFile(pageName, "Test User", "test@example.com", "import index.md")
	checkFatal(t, err)
}

func TestRenamePage(t *testing.T) {
	gs := createTestRepo(t)
	defer cleanup(t, gs)

	pageName := createIndexPage(t, gs)
	_, _, err := gs.CommitFile(pageName, "Test User", "test@example.com", "import index.md")
	checkFatal(t, err)

	newPageName := "foobar.md"
	output, err := gs.RenamePage(pageName, newPageName)
	t.Log(output)
	checkFatal(t, err)

	pagePath := filepath.Join(gs.WorkDir, newPageName)
	_, err = os.Stat(pagePath)
	checkFatal(t, err)
}

func TestLogsForPage(t *testing.T) {
	gs := createTestRepo(t)
	defer cleanup(t, gs)

	pageName := createIndexPage(t, gs)

	_, _, err := gs.CommitFile(pageName, "Test User", "test@example.com", "import index.md")
	checkFatal(t, err)

	pagePath := filepath.Join(gs.WorkDir, pageName)
	checkFatal(t, ioutil.WriteFile(pagePath, []byte("foo bar baz"), 0644))
	_, _, err = gs.CommitFile(pageName, "Another Test User", "a_test@example.com", "modify index.md for fun")
	checkFatal(t, err)

	logs, err := gs.LogsForPage(pageName, 0)
	checkFatal(t, err)

	if len(logs) != 2 {
		t.Fatalf("should return 2 logs but returned %d", len(logs))
	}

	exMessages := []string{
		"modify index.md for fun",
		"import index.md",
	}

	for i, commitLog := range logs {
		if commitLog.Message != exMessages[i] {
			t.Fatalf("Message should be \"%s\", is \"%s\"", exMessages[i], commitLog.Message)
		}
	}
}
