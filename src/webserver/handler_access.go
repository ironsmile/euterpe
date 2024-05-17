package webserver

import (
	"log"
	"net/http"
	"net/url"
	"time"
)

// AccessHandler is an http.Handler which wraps around another handler and prints
// access logs.
type AccessHandler struct {
	wrapped http.Handler
}

// NewAccessHandler returns an AccessHandler which will call `h` and the log
// information about the http request and response.
func NewAccessHandler(h http.Handler) *AccessHandler {
	return &AccessHandler{
		wrapped: h,
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *AccessHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	started := time.Now()
	ww := newLoggedResponseWriter(w)
	h.wrapped.ServeHTTP(ww, req)
	elapsed := time.Since(started)

	reqURL, _ := url.ParseRequestURI(req.RequestURI)
	query := reqURL.Query()
	if p := query.Get("p"); p != "" {
		query.Set("p", queryRedactedValue)
	}
	if t := query.Get("t"); t != "" {
		query.Set("t", queryRedactedValue)
	}
	if s := query.Get("s"); s != "" {
		query.Set("s", queryRedactedValue)
	}
	if t := query.Get("token"); t != "" {
		query.Set("token", queryRedactedValue)
	}
	reqURL.RawQuery = query.Encode()

	log.Printf(
		"%s %s dur=%s status=%d userAgent=%s remoteAddr=%s\n",
		req.Method, reqURL, elapsed, ww.code, req.Header.Get("User-Agent"),
		req.RemoteAddr,
	)
}

type loggedResponseWriter struct {
	http.ResponseWriter
	code int
}

func newLoggedResponseWriter(w http.ResponseWriter) *loggedResponseWriter {
	return &loggedResponseWriter{
		ResponseWriter: w,
		code:           http.StatusOK,
	}
}

func (w *loggedResponseWriter) WriteHeader(status int) {
	w.code = status
	w.ResponseWriter.WriteHeader(status)
}

const queryRedactedValue = "REDACTED"
