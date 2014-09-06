// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package spock

import (
	"crypto/md5"
	"fmt"
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
	input := strings.ToLower(strings.TrimSpace(email))
	return fmt.Sprintf("%x", md5.Sum([]byte(input)))
}
