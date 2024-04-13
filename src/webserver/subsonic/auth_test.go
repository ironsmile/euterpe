package subsonic_test

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ironsmile/euterpe/src/config"
	"github.com/ironsmile/euterpe/src/library/libraryfakes"
	"github.com/ironsmile/euterpe/src/webserver/subsonic"
	"github.com/ironsmile/euterpe/src/webserver/subsonic/subsonicfakes"
)

// TestAuthHandler checks that authentication works as explained
// in the Subsonic API documentation.
func TestAuthHandler(t *testing.T) {
	const (
		username = "the-real-user"
		password = "the-real-password"
		salt     = "random-salt"
	)

	tokenMD5 := md5.New()
	fmt.Fprintf(tokenMD5, "%s%s", password, salt)
	token := hex.EncodeToString(tokenMD5.Sum(nil))

	tests := []struct {
		Desc         string
		SkipAuth     bool
		Query        map[string]string
		Success      bool
		ExpectedCode int
	}{
		{
			Desc:     "no authentication",
			SkipAuth: true,
			Success:  true,
		},
		{
			Desc: "with plain-text password",
			Query: map[string]string{
				"u": username,
				"p": password,
			},
			Success: true,
		},
		{
			Desc: "with encoded password",
			Query: map[string]string{
				"u": username,
				"p": "enc:" + hex.EncodeToString([]byte(password)),
			},
			Success: true,
		},
		{
			Desc: "with salt and token",
			Query: map[string]string{
				"u": username,
				"s": salt,
				"t": token,
			},
			Success: true,
		},
		{
			Desc: "missing username",
			Query: map[string]string{
				"p": password,
			},
			Success:      false,
			ExpectedCode: 10,
		},
		{
			Desc: "missing password",
			Query: map[string]string{
				"u": username,
			},
			Success:      false,
			ExpectedCode: 10,
		},
		{
			Desc: "missing salt",
			Query: map[string]string{
				"u": username,
				"t": "invalid",
			},
			Success:      false,
			ExpectedCode: 10,
		},
		{
			Desc: "missing token",
			Query: map[string]string{
				"u": username,
				"s": "some-salt",
			},
			Success:      false,
			ExpectedCode: 10,
		},
		{
			Desc:         "nothing in query",
			Success:      false,
			ExpectedCode: 10,
		},
		{
			Desc: "wrong plain-text password",
			Query: map[string]string{
				"u": username,
				"p": "wrong-password",
			},
			Success:      false,
			ExpectedCode: 40,
		},
		{
			Desc: "wrong username with plain text password",
			Query: map[string]string{
				"u": "wrong-username",
				"p": password,
			},
			Success:      false,
			ExpectedCode: 40,
		},
		{
			Desc: "wrong token for salt",
			Query: map[string]string{
				"u": username,
				"s": salt,
				"t": "wrong-token",
			},
			Success:      false,
			ExpectedCode: 40,
		},
		{
			Desc: "wrong username with token and salt",
			Query: map[string]string{
				"u": "wrong-username",
				"s": salt,
				"t": token,
			},
			Success:      false,
			ExpectedCode: 40,
		},
	}

	for _, test := range tests {
		checkSuccess := func(t *testing.T, resp *http.Response) {
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				t.Errorf(
					"expected HTTP status for success but got %d",
					resp.StatusCode,
				)
			}

			dec := xml.NewDecoder(resp.Body)
			respXML := baseResponse{}
			if err := dec.Decode(&respXML); err != nil {
				t.Fatalf("failed to decode XML response: %s", err)
			}

			if respXML.Status != "ok" {
				t.Errorf("expected status tag `ok` but got `%s`", respXML.Status)
			}
		}

		checkFailure := func(t *testing.T, resp *http.Response) {
			dec := xml.NewDecoder(resp.Body)
			respXML := errorResponse{}
			if err := dec.Decode(&respXML); err != nil {
				t.Fatalf("failed to decode XML response: %s", err)
			}

			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf(
					"expected HTTP status for unauthorized but got %d",
					resp.StatusCode,
				)
			}

			if respXML.Status != "failed" {
				t.Errorf("expected status `failed` but got `%s`", respXML.Status)
			}

			if respXML.Error.Code != test.ExpectedCode {
				t.Errorf(
					"expected error code %d but got %d",
					test.ExpectedCode,
					respXML.Error.Code,
				)
			}
		}

		t.Run(test.Desc, func(t *testing.T) {
			cfg := config.Config{
				Auth: !test.SkipAuth,
				Authenticate: config.Auth{
					User:     username,
					Password: password,
				},
			}

			sh := subsonic.NewHandler(
				subsonic.Prefix,
				&libraryfakes.FakeLibrary{},
				&libraryfakes.FakeBrowser{},
				cfg,
				&subsonicfakes.FakeCoverArtHandler{},
				&subsonicfakes.FakeCoverArtHandler{},
			)

			srv := httptest.NewServer(sh)
			defer srv.Close()

			reqURL, err := url.Parse(srv.URL)
			if err != nil {
				t.Fatalf("test server URL malformed: %s", err)
			}

			query := make(url.Values)
			for qk, qv := range test.Query {
				query.Set(qk, qv)
			}
			reqURL.Path = subsonic.Prefix + "/ping/"
			reqURL.RawQuery = query.Encode()

			t.Logf("Making HTTP request %s", reqURL.String())

			req, err := http.NewRequest(
				http.MethodGet,
				reqURL.String(),
				nil,
			)
			if err != nil {
				t.Fatalf("cannot create request: %s", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("HTTP request failed: %s", err)
			}
			defer resp.Body.Close()

			if test.Success {
				checkSuccess(t, resp)
			} else {
				checkFailure(t, resp)
			}
		})
	}
}

type baseResponse struct {
	XMLName xml.Name `xml:"subsonic-response"`
	Status  string   `xml:"status,attr"`
	Version string   `xml:"version,attr"`
}

type errorResponse struct {
	XMLName xml.Name `xml:"subsonic-response"`
	Status  string   `xml:"status,attr"`
	Version string   `xml:"version,attr"`

	Error errorElement
}

type errorElement struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Message string   `xml:"message,attr"`
}
