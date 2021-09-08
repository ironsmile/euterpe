package version_test

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/ironsmile/euterpe/src/version"
)

// TestVersionPrinting makes sure some things are always part of the printed version
// string.
func TestVersionPrinting(t *testing.T) {
	if version.Version == "" {
		t.Fatalf("version.Version cannot be completely empty")
	}

	var buff bytes.Buffer
	version.Print(&buff)

	if !strings.Contains(buff.String(), version.Version) {
		t.Errorf("printed version does not contain the actual version string")
	}

	if !strings.Contains(buff.String(), runtime.Version()) {
		t.Errorf("printed version does not contain Golang version")
	}
}
