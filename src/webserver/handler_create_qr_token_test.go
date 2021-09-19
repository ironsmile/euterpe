package webserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	// Used for QR code recognizing.
	_ "image/jpeg"
	_ "image/png"

	"github.com/gbrlsnchs/jwt"
	"github.com/ironsmile/euterpe/src/config"
	"github.com/ironsmile/euterpe/src/webserver"
	"github.com/liyue201/goqr"
)

// TestCreateQRTokenHandlerGeneratedCodes generates QR codes with different server
// settings and then checks that the QR could be parsed and contains the desired
// JSON message.
func TestCreateQRTokenHandlerGeneratedCodes(t *testing.T) {
	const serverAddress = "http://music.example.com"

	tests := []struct {
		desc         string
		queryAddress string
		needsAuth    bool
		auth         config.Auth

		expectedCode int
	}{
		{
			desc:         "no authentication",
			queryAddress: serverAddress,
			expectedCode: http.StatusOK,
		},
		{
			desc:         "with authentication",
			queryAddress: serverAddress,
			needsAuth:    true,
			auth: config.Auth{
				Secret: "very-secret-string-for-tests",
			},
			expectedCode: http.StatusOK,
		},
		{
			desc:         "no secret for handler configured with authentication",
			queryAddress: serverAddress,
			needsAuth:    true,
			auth: config.Auth{
				Secret: "",
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			handler := webserver.NewCreateQRTokenHandler(test.needsAuth, test.auth)
			req := httptest.NewRequest(
				http.MethodGet,
				"/",
				nil,
			)
			q := req.URL.Query()
			q.Set("address", test.queryAddress)
			req.URL.RawQuery = q.Encode()
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)

			responseCode := resp.Result().StatusCode
			if responseCode != test.expectedCode {
				t.Fatalf(
					"expected HTTP Status %d but got %d",
					test.expectedCode,
					responseCode,
				)
			}

			if test.expectedCode != http.StatusOK {
				return
			}

			qrImg, _, err := image.Decode(resp.Body)
			if err != nil {
				t.Fatalf("error decoding QR code image: %s", err)
			}

			qrCodes, err := goqr.Recognize(qrImg)
			if err != nil {
				t.Fatalf("unexpected QR reading error: %s", err)
			}

			if len(qrCodes) != 1 {
				t.Fatalf("expected one QR code but found %d", len(qrCodes))
			}

			qrBytes := make([]byte, 0, len(qrCodes[0].Payload))
			for _, b := range qrCodes[0].Payload {
				qrBytes = append(qrBytes, byte(b))
			}

			var qrParsed qrResponse
			buf := bytes.NewReader(qrBytes)
			dec := json.NewDecoder(buf)
			if err := dec.Decode(&qrParsed); err != nil {
				t.Fatalf("error JSON decoding the QR code contents: %s", err)
			}

			if test.queryAddress != qrParsed.Address {
				t.Errorf(
					"QR address mismatch. Expected `%+v` but got `%+v`",
					test.queryAddress,
					qrParsed.Address,
				)
			}

			if qrParsed.Software != "httpms" {
				t.Errorf(
					"Wrong software identity string. Expected `%+v` but got `%+v`",
					"httpms",
					qrParsed.Software,
				)
			}

			if !test.needsAuth {
				return
			}

			fmt.Printf("%s: %d\n", test.desc, test.expectedCode)

			jot, err := jwt.FromString(qrParsed.Token)
			if err != nil {
				t.Fatalf("error parsing JWT token from string: %s", err)
			}

			if err := jot.Verify(jwt.HS256(test.auth.Secret)); err != nil {
				t.Fatalf("error verifying JWT token: %s", err)
			}

			alg := jwt.AlgorithmValidator(jwt.MethodHS256)
			exp := jwt.ExpirationTimeValidator(time.Now())

			if err := jot.Validate(alg, exp); err != nil {
				t.Fatalf("error validating JWT token: %s", err)
			}
		})
	}
}

type qrResponse struct {
	Software string `json:"software"`
	Token    string `json:"token,omitempty"`
	Address  string `json:"address"`
}
