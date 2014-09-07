package spock

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestMkMissingDirs(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "spock")
	checkFatal(t, err)
	d := filepath.Join(tmpdir, "foo", "bar")
	fn := filepath.Join(d, "myfile.md")
	err = MkMissingDirs(fn)
	checkFatal(t, err)

	fi, err := os.Stat(d)
	checkFatal(t, err)
	if !fi.IsDir() {
		t.Fatalf("%s is not a directory\n", d)
	}

	err = os.RemoveAll(tmpdir)
	checkFatal(t, err)
}
