package library

import (
	"io/fs"
	"os"
)

// osFS is a fs.FS implementation which uses the os package as the underlying file
// opener.
type osFS struct{}

func (fs *osFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}
