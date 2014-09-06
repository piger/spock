// Copyright 2014 Daniel Kertesz <daniel@spatof.org>
// All rights reserved. This program comes with ABSOLUTELY NO WARRANTY.
// See the file LICENSE for details.

package spock

import (
	"os"
	"path/filepath"
)

// MkMissingDirs creates parent directories for the specified 'filename'.
func MkMissingDirs(filename string) error {
	dirname := filepath.Dir(filename)

	if _, err := os.Stat(dirname); err != nil {
		if err := os.MkdirAll(dirname, 0755); err != nil {
			return err
		}
	}

	return nil
}
