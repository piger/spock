package spock

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
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

func createTestPage(t *testing.T, gs *GitStorage, path, content, authorName, authorEmail, message string, when time.Time) string {
	pagePath := filepath.Join(gs.WorkDir, path)
	checkFatal(t, ioutil.WriteFile(pagePath, []byte(content), 0644))
	sig := &CommitSignature{Name: authorName, Email: authorEmail, When: when}
	_, _, err := gs.CommitFile(path, sig, message)
	checkFatal(t, err)

	return path
}

func createSignature(t *testing.T) *CommitSignature {
	loc, err := time.LoadLocation("Europe/Rome")
	checkFatal(t, err)
	sig := &CommitSignature{
		Name:  "Palle Nyborg",
		Email: "palle@superfoppa.com",
		When:  time.Date(2014, 8, 24, 22, 12, 0, 0, loc),
	}
	return sig
}


// Tests
func TestInitRepository(t *testing.T) {
	gs := createTestRepo(t)
	defer cleanup(t, gs)
}

func TestCommitFile(t *testing.T) {
	gs := createTestRepo(t)
	defer cleanup(t, gs)

	pageName := createIndexPage(t, gs)
	_, _, err := gs.CommitFile(pageName, createSignature(t), "import index.md")
	checkFatal(t, err)
}

func TestRenamePage(t *testing.T) {
	gs := createTestRepo(t)
	defer cleanup(t, gs)

	pageName := createIndexPage(t, gs)
	sig := createSignature(t)
	_, _, err := gs.CommitFile(pageName, sig, "import index.md")
	checkFatal(t, err)

	newPageName := "foobar.md"
	message := "Renamed index.md to foobar.md"
	_, _, err = gs.RenamePage(pageName, newPageName, sig, message)
	checkFatal(t, err)

	pagePath := filepath.Join(gs.WorkDir, newPageName)
	_, err = os.Stat(pagePath)
	checkFatal(t, err)

	// here we will get just 1 log insted of 2 (which would be correct), because
	// our LogsForPage doesn't detect renamed files.
	logs, err := gs.LogsForPage(newPageName)
	checkFatal(t, err)
	if len(logs) != 1 {
		t.Fatalf("There should be 1 log, there are %d", len(logs))
	}
	commitLog := logs[0]
	if commitLog.Message != message {
		t.Fatalf("Commit message should be \"%s\", is \"%s\"", message+"\n", commitLog.Message)
	}
}

func TestDeletePage(t *testing.T) {
	gs := createTestRepo(t)
	defer cleanup(t, gs)

	pageName := createIndexPage(t, gs)
	sig := createSignature(t)

	_, _, err := gs.CommitFile(pageName, sig, "import index.md")
	checkFatal(t, err)

	_, _, err = gs.DeletePage(pageName, sig, "get rid of index.md")
	checkFatal(t, err)

	pagePath := filepath.Join(gs.WorkDir, pageName)
	if _, err := os.Stat(pagePath); err == nil {
		t.Fatalf("File should have been deleted: %s", pagePath)
	}
}

func TestLogsForPage(t *testing.T) {
	gs := createTestRepo(t)
	defer cleanup(t, gs)

	pageName := createIndexPage(t, gs)

	sig := createSignature(t)
	messages := []string{
		"import index.md",
		"modify index.md for fun",
	}
	_, _, err := gs.CommitFile(pageName, sig, messages[0])
	checkFatal(t, err)

	pagePath := filepath.Join(gs.WorkDir, pageName)
	checkFatal(t, ioutil.WriteFile(pagePath, []byte("foo bar baz"), 0644))
	sig2 := &CommitSignature{Name: "Another Test User", Email: "a_test@example.com", When: time.Now()}
	_, _, err = gs.CommitFile(pageName, sig2, messages[1])
	checkFatal(t, err)

	logs, err := gs.LogsForPage(pageName)
	checkFatal(t, err)

	if len(logs) != 2 {
		t.Fatalf("should return %d logs but returned %d", len(messages), len(logs))
	}

	// reverse the messages array, as we are getting commit messages in
	// descending order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	for i, commitLog := range logs {
		if commitLog.Message != messages[i] {
			t.Fatalf("Message should be \"%s\", is \"%s\"", messages[i], commitLog.Message)
		}
	}
}


func TestGetLastCommit(t *testing.T) {
	gs := createTestRepo(t)
	defer cleanup(t, gs)

	pageName := createIndexPage(t, gs)

	sig := &CommitSignature{Name: "Test User", Email: "test@example.com", When: time.Now()}
	msg := "import index.md"
	_, _, err := gs.CommitFile(pageName, sig, msg)
	checkFatal(t, err)

	lastcommit, err := gs.GetLastCommit(pageName)
	checkFatal(t, err)
	if lastcommit.Message != msg {
		t.Fatalf("Commit message should be \"%s\", is \"%s\"", msg, lastcommit.Message)
	}
}

func TestListPages(t *testing.T) {
	gs := createTestRepo(t)
	defer cleanup(t, gs)

	createTestPage(t, gs, "index.md", "this is my index", "test user", "test@email.com", "created", time.Now())
	createTestPage(t, gs, "foobar.md", "my foobar page!", "test user", "test@email.com", "created", time.Now())

	pages, err := gs.ListPages()
	checkFatal(t, err)
	if len(pages) != 2 {
		t.Fatal("There should be 2 pages, there are %d", len(pages))
	}
}
