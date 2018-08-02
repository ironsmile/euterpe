package webserver

import (
	"crypto/subtle"
	"log"
	"net/http"

	"github.com/ironsmile/httpms/src/config"
)

// HandlerFuncWithError is similar to http.HandlerFunc but returns an error when
// the handling of the request failed.
type HandlerFuncWithError func(http.ResponseWriter, *http.Request) error

// InternalErrorOnErrorHandler is used to wrap around handlers-like functions which just
// return error. This function actually writes the HTTP error and renders the error in
// the html.
func InternalErrorOnErrorHandler(writer http.ResponseWriter, req *http.Request,
	fnc HandlerFuncWithError) {
	WithInternalError(fnc)(writer, req)
}

// WithInternalError converts HandlerFuncWithError to http.HandlerFunc by making sure
// all errors returned are flushed to the writer and Internal Server Error HTTP status
// is sent.
func WithInternalError(fnc HandlerFuncWithError) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		err := fnc(writer, req)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			if _, err := writer.Write([]byte(err.Error())); err != nil {
				log.Printf("error writing body in InternalErrorHandler: %s", err)
			}
		}
	}
}

// The following check is carefully orchestrated so that it will take constant
// time for wrong and correct pairs of username and password. This mitigates
// simple timing attacks.
func checkLoginCreds(user, pass string, auth config.Auth) bool {
	userCheck := subtle.ConstantTimeCompare([]byte(user), []byte(auth.User))
	passCheck := subtle.ConstantTimeCompare([]byte(pass), []byte(auth.Password))

	return userCheck&passCheck == 1
}
