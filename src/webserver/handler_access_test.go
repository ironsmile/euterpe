package webserver_test

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ironsmile/euterpe/src/webserver"
)

// TestAccessHandler makes sure that the access handler works and also sensitive
// that strings are not stored into the access log. These are passwords and tokens.
func TestAccessHandler(t *testing.T) {
	recorder := &recordingHandler{}
	accessHandler := webserver.NewAccessHandler(recorder)

	const (
		pass    = "hidden-subsonic-password"
		ssToken = "hidden-subsonic-token"
		salt    = "hidden-subsonic-salt"
		token   = "hidden-token"
	)

	buffer := &bytes.Buffer{}
	log.Default().SetOutput(buffer)
	defer func() {
		log.Default().SetOutput(os.Stdout)
	}()

	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"/v1/api?p=%s&t=%s&s=%s&token=%s&unrelated=5",
			pass, ssToken, salt, token,
		),
		nil,
	)
	req.Header.Set("User-Agent", "http-unit-test")

	resp := httptest.NewRecorder()

	accessHandler.ServeHTTP(resp, req)

	if recorder.called != 1 {
		t.Errorf(
			"expected wrapped handler to be called once but it was called %d times",
			recorder.called,
		)
	}

	logged := buffer.String()
	t.Logf("ACCESS LOG BUFFER: %s\n", logged)

	if logged == "" {
		t.Error("the access log did not log anything")
	}

	if !strings.Contains(logged, "/v1/api") || !strings.Contains(logged, "unrelated=5") {
		t.Errorf("access log was missing parts of the request URL")
	}

	if strings.Contains(logged, pass) {
		t.Errorf("access log did not hide the `p` (subsonic password) query value")
	}

	if strings.Contains(logged, ssToken) {
		t.Errorf("access log did not hide the `t` (subsonic token) query value")
	}

	if strings.Contains(logged, salt) {
		t.Errorf("access log did not hide the `s` (subsonic salt) query value")
	}

	if strings.Contains(logged, token) {
		t.Errorf("access log did not hide the `token` query value")
	}
}

type recordingHandler struct {
	called int
}

func (h *recordingHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {
	h.called++
}
