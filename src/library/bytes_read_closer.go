package library

import "bytes"

// bytesReadCloser is a helper type which helps bytes.Reader implement io.ReadCloser.
type bytesReadCloser struct {
	bytes.Buffer
}

// Close is a no-op which is here only to implement io.Closer and io.ReadCloser.
func (b *bytesReadCloser) Close() error {
	return nil
}

func newBytesReadCloser(buff []byte) *bytesReadCloser {
	return &bytesReadCloser{
		Buffer: *bytes.NewBuffer(buff),
	}
}
