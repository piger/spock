package spock

import (
	"testing"
)

func TestShortName(t *testing.T) {
	p := &Page{Path: "notes/diary.md"}
	if p.ShortName() != "notes/diary" {
		t.Fatalf("ShortName() should be \"notes/diary\", is %s", p.ShortName())
	}
}

var pageBytes []byte = []byte(`---
title: "My page"
description: "My page description"
tags: [ "cow", "dog", "cat" ]
language: "en"
---
# My animals

A page about my animals
`)

func TestParsePageBytes(t *testing.T) {
	ph, content, err := ParsePageBytes(pageBytes)
	checkFatal(t, err)
	if ph.Title != "My page" {
		t.Fatalf("header.Title should be 'My page', is: %s\n", ph.Title)
	}
	if ph.Description != "My page description" {
		t.Fatalf("header.Description should be 'My page description', is: %s\n", ph.Description)
	}

	expectedContent := `
# My animals

A page about my animals
`
	if string(content) != expectedContent {
		t.Fatal("content is different from expected")
	}
}
