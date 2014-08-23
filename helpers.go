package spock

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"time"
)

// Proxy to Time.Format()
func formatDatetime(t time.Time, layout string) string {
	return t.Format(layout)
}

// Return md5 checksum of an email address that can be used to lookup a gravatar
// icon.
func gravatarHash(email string) string {
	hasher := md5.New()
	hasher.Write([]byte(strings.ToLower(email)))
	return hex.EncodeToString(hasher.Sum(nil))
}
