package library

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// TestOsFSOpen makes sure that the osFS Open function actually uses the OS file system
// to do its work.
func TestOsFSOpen(t *testing.T) {
	const tmpFileContents = "some-file-contents"

	osfs := &osFS{}

	_, err := osfs.Open(filepath.FromSlash("does/not/exists/here.png"))
	if !os.IsNotExist(err) {
		t.Errorf("unexpected error for file which does not exist: `%+v`", err)
	}

	tmpFile, err := os.CreateTemp(os.TempDir(), "pattern")
	if err != nil {
		t.Fatalf("failed to create a temporary file: %s", err)
	}
	tmpFileName := tmpFile.Name()
	defer os.Remove(tmpFileName)
	fmt.Fprint(tmpFile, tmpFileContents)
	tmpFile.Close()

	fh, err := osfs.Open(tmpFileName)
	if err != nil {
		t.Fatalf("error opening temporary file with osFS: %s", err)
	}
	defer fh.Close()

	foundContents, err := ioutil.ReadAll(fh)
	if err != nil {
		t.Fatalf("error reading temporary file: %s", err)
	}

	if !bytes.Equal([]byte(tmpFileContents), foundContents) {
		t.Errorf("expected file `%s` but got `%s`", tmpFileContents, foundContents)
	}
}
