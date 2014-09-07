package spock

import (
	"testing"
	"time"
)

func TestFormatDatetime(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Rome")
	checkFatal(t, err)
	dt := time.Date(2014, 9, 7, 11, 42, 0, 0, loc)
	rv := formatDatetime(dt, "Mon Jan 2 15:04 2006")
	expected := "Sun Sep 7 11:42 2014"

	if rv != expected {
		t.Fatalf("result should be %s, is %s\n", expected, rv)
	}
}

func TestGravatarHash(t *testing.T) {
	email1 := "user@example.com"
	email2 := "  USer@example.com"

	hash1 := gravatarHash(email1)
	hash2 := gravatarHash(email2)

	if hash1 != hash2 {
		t.Fatalf("hash for %s should be %s, is %s\n", email1, hash1, hash2)
	}
}
