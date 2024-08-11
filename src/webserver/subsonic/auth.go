package subsonic

import (
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

func (s *subsonic) authHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if err := r.ParseForm(); err != nil {
			http.Error(
				w,
				fmt.Sprintf("cannot parse request: %s", err),
				http.StatusBadRequest,
			)
			return
		}

		if !s.needsAuth || r.URL.Path == "/rest/ping.view" {
			handler.ServeHTTP(w, r)
			return
		}

		user := r.Form.Get("u")
		pass := r.Form.Get("p")
		token := r.Form.Get("t")
		salt := r.Form.Get("s")

		if user == "" || (pass == "" && (token == "" || salt == "")) {
			resp := responseError(
				errCodeMissingParameter,
				"Required parameter is missing",
			)

			w.WriteHeader(http.StatusUnauthorized)
			encodeResponse(w, r, resp)
			return
		}

		var authSuccess bool

		if pass != "" {
			if strings.HasPrefix(pass, "enc:") {
				pass = strings.TrimPrefix(pass, "enc:")
				decPass, err := hex.DecodeString(pass)
				if err != nil {
					resp := responseError(
						errCodeWrongUserOrPass,
						fmt.Sprintf(
							"Password encoded wrong: %s",
							err,
						),
					)

					w.WriteHeader(http.StatusUnauthorized)
					encodeResponse(w, r, resp)
					return
				}

				pass = string(decPass)
			}

			userCheck := subtle.ConstantTimeCompare([]byte(user), []byte(s.auth.User))
			passCheck := subtle.ConstantTimeCompare([]byte(pass), []byte(s.auth.Password))

			if userCheck&passCheck == 1 {
				authSuccess = true
			}
		} else {
			correctToken := md5.New()
			_, _ = fmt.Fprintf(correctToken, "%s%s", s.auth.Password, salt)
			correctTokenHex := hex.EncodeToString(correctToken.Sum(nil))

			userCheck := subtle.ConstantTimeCompare([]byte(user), []byte(s.auth.User))
			tokenCheck := subtle.ConstantTimeCompare(
				[]byte(token),
				[]byte(correctTokenHex),
			)

			authSuccess = tokenCheck&userCheck == 1
		}

		if !authSuccess {
			resp := responseError(
				errCodeWrongUserOrPass,
				"Wrong username or password",
			)

			w.WriteHeader(http.StatusUnauthorized)
			encodeResponse(w, r, resp)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
