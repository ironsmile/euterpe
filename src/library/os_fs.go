package library

import (
	"io/fs"
	"os"
)

// osFS is a fs.FS implementation which uses the os package as the underlying file
// opener.
type osFS struct{}

var _ fs.StatFS = (*osFS)(nil)

func (osfs *osFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (osfs *osFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}
