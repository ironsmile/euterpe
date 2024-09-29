package wrapfs

import (
	"errors"
	"io"
	"io/fs"
	"time"
)

// WithModTime returns a wrapper around `root` which makes sure calls to
// [fs.File.FileInfo] and [fs.DirEntry.Info] always return non-zero ModTime(). In case
// the wrapped file or entry return non-zero modification time it stays unchanged.
// In case they return a zero modification time then `modTime` is used instead.
func WithModTime(root fs.FS, modTime time.Time) fs.FS {
	return &fsWrapper{
		FS:      root,
		modTime: modTime,
	}
}

// fsWrapper is a fs.FS implementation which implements [fs.FS.Open] and all of the
// rest of optional interfaces in `fs`.
type fsWrapper struct {
	fs.FS

	modTime time.Time
}

func (f *fsWrapper) Open(name string) (fs.File, error) {
	fe, err := f.FS.Open(name)
	if err != nil {
		return nil, err
	}
	if _, ok := fe.(readerDir); ok {
		return &openDir{File: fe, modTime: f.modTime}, nil
	}

	of := &openFile{File: fe, modTime: f.modTime}

	seeker, isSeeker := fe.(io.Seeker)
	readerAt, isReaderAt := fe.(io.ReaderAt)
	if isSeeker && isReaderAt {
		return &openFileSeekReaderAt{
			File:     of,
			Seeker:   seeker,
			ReaderAt: readerAt,
		}, nil
	}
	if isSeeker {
		return &openFileSeeker{
			File:   of,
			Seeker: seeker,
		}, nil
	}
	if isReaderAt {
		return &openFileReaderAt{
			File:     of,
			ReaderAt: readerAt,
		}, nil
	}

	return of, nil
}

func (f *fsWrapper) Stat(name string) (fs.FileInfo, error) {
	fi, err := fs.Stat(f.FS, name)
	if err != nil {
		return nil, err
	}
	return &fileInfo{FileInfo: fi, modTime: f.modTime}, nil
}

func (f *fsWrapper) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(f.FS, name)
}

func (f *fsWrapper) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(f.FS, name)
	if err != nil {
		return nil, err
	}

	return wrapDirEntries(entries, f.modTime), nil
}

func (f *fsWrapper) Glob(pattern string) ([]string, error) {
	return fs.Glob(f.FS, pattern)
}

func (f *fsWrapper) Sub(dir string) (fs.FS, error) {
	subFS, err := fs.Sub(f.FS, dir)
	if err != nil {
		return nil, err
	}

	return &fsWrapper{
		FS:      subFS,
		modTime: f.modTime,
	}, nil
}

var (
	_ fs.ReadDirFS  = &fsWrapper{}
	_ fs.ReadFileFS = &fsWrapper{}
	_ fs.StatFS     = &fsWrapper{}
	_ fs.SubFS      = &fsWrapper{}
)

// An openFile is a regular file open for reading.
type openFile struct {
	fs.File

	modTime time.Time
}

func (f *openFile) Stat() (fs.FileInfo, error) {
	fi, err := f.File.Stat()
	if err != nil {
		return nil, err
	}
	return &fileInfo{FileInfo: fi, modTime: f.modTime}, nil
}

type openFileSeeker struct {
	fs.File
	io.Seeker
}

type openFileReaderAt struct {
	fs.File
	io.ReaderAt
}

type openFileSeekReaderAt struct {
	fs.File
	io.Seeker
	io.ReaderAt
}

// An openDir is a directory open for reading.
type openDir struct {
	fs.File

	modTime time.Time
}

func (d *openDir) Stat() (fs.FileInfo, error) {
	fi, err := d.File.Stat()
	if err != nil {
		return nil, err
	}
	return &fileInfo{FileInfo: fi, modTime: d.modTime}, nil
}

func (d *openDir) ReadDir(count int) ([]fs.DirEntry, error) {
	rd, ok := d.File.(readerDir)
	if !ok {
		return nil, errors.New("wrapped dir does not implement ReadeDir")
	}
	entries, err := rd.ReadDir(count)
	if err != nil {
		return nil, err
	}

	return wrapDirEntries(entries, d.modTime), nil
}

func wrapDirEntries(entries []fs.DirEntry, modTime time.Time) []fs.DirEntry {
	wrappedEntries := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		wrappedEntries = append(wrappedEntries, &dirEntry{
			DirEntry: entry,
			modTime:  modTime,
		})
	}

	return wrappedEntries
}

type fileInfo struct {
	fs.FileInfo

	modTime time.Time
}

func (i *fileInfo) ModTime() time.Time {
	md := i.FileInfo.ModTime()
	if md.IsZero() {
		return i.modTime
	}
	return md
}

type dirEntry struct {
	fs.DirEntry

	modTime time.Time
}

func (e *dirEntry) Info() (fs.FileInfo, error) {
	info, err := e.DirEntry.Info()
	if err != nil {
		return nil, err
	}

	return &fileInfo{FileInfo: info, modTime: e.modTime}, nil
}

type readerDir interface {
	ReadDir(count int) ([]fs.DirEntry, error)
}
