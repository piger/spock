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
