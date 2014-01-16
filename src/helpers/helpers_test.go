package helpers

import (
	"testing"

	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func TestProjectRoot(t *testing.T) {
	// If you are running the tests you should always get the source root

	path, err := ProjectRoot()

	if err != nil {
		t.Errorf(err.Error())
	}

	p := strings.Split(os.ExpandEnv("$GOPATH"), ":")[0]
	expected := filepath.Join(p, filepath.FromSlash("src/github.com/ironsmile/httpms"))

	if path != expected {
		t.Errorf(fmt.Sprintf("Expected `%s` but got `%s`", expected, path))
	}
}
