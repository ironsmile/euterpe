package webserver

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// Custom writer to make our webserver gzip output when possible.
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	if "" == w.Header().Get("Content-Type") {
		// If no content type, apply sniffing algorithm to un-gzipped body.
		w.Header().Set("Content-Type", http.DetectContentType(b))
	}
	return w.Writer.Write(b)
}

// GzipHandler gzips our output using a custom Writer. It will check if gzip is among the
// accepted encodings and gzip if so. Otherwise it will do nothing.
type GzipHandler struct {
	wrapped    http.Handler
	exceptions []string
}

// ServeHTTP satisfies the http.Handler interface
func (gzh GzipHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	if !strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
		gzh.wrapped.ServeHTTP(writer, req)
		return
	}

	for _, path := range gzh.exceptions {
		if strings.HasPrefix(req.URL.Path, path) {
			gzh.wrapped.ServeHTTP(writer, req)
			return
		}
	}

	writer.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(writer)
	defer gz.Close()
	gzr := gzipResponseWriter{Writer: gz, ResponseWriter: writer}
	gzh.wrapped.ServeHTTP(gzr, req)
}

// NewGzipHandler returns GzipHandler which will gzip anything written in the supplied
// handler. Must be the main handler given to the net.Server
func NewGzipHandler(handler http.Handler, exceptions []string) http.Handler {
	return &GzipHandler{
		wrapped:    handler,
		exceptions: exceptions,
	}
}
