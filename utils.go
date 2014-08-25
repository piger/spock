package spock

import (
	"path/filepath"
	"os"
)

// Create 'filename' parent directories.
func MkMissingDirs(filename string) error {
	dirname := filepath.Dir(filename)

	if _, err := os.Stat(dirname); err != nil {
		if err := os.MkdirAll(dirname, 0755); err != nil {
			return err
		}
	}

	return nil
}
