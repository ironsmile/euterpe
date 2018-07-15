package library

import "bytes"

// bytesReadCloser is a helper type which helps bytes.Reader implement io.ReadCloser.
type bytesReadCloser struct {
	r *bytes.Reader
}

// Close is a no-op which is here only to implement io.Closer and io.ReadCloser.
func (b *bytesReadCloser) Close() error {
	return nil
}

// Read is needed to implement io.Reader and io.ReadCloser.
func (b *bytesReadCloser) Read(buff []byte) (int, error) {
	return b.r.Read(buff)
}

// ReadAt implements io.ReaderAt.
func (b *bytesReadCloser) ReadAt(buff []byte, off int64) (int, error) {
	return b.r.ReadAt(buff, off)
}

// Seek implements io.Seeker.
func (b *bytesReadCloser) Seek(offset int64, whence int) (int64, error) {
	return b.r.Seek(offset, whence)
}

// Len returns the number of bytes of the unread portion of the slice.
func (b *bytesReadCloser) Len() int {
	return b.r.Len()
}

// Size returns the original length of the underlying byte slice. Size is the number of
// bytes available for reading via ReadAt. The returned value is always the same and is
// not affected by calls to any other method.
func (b *bytesReadCloser) Size() int64 {
	return b.r.Size()
}

func newBytesReadCloser(r *bytes.Reader) *bytesReadCloser {
	return &bytesReadCloser{
		r: r,
	}
}
