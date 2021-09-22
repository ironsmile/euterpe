package webserver_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/library/libraryfakes"
	"github.com/ironsmile/euterpe/src/webserver"
)

// TestArtstImageHandlerGET checks that the HTTP handler for GET requests is parsing
// its arguments as expected and responds with the image found by its image manager.
func TestArtstImageHandlerGET(t *testing.T) {
	imgBytesOriginal := []byte("artist 321 image original")
	imgBytesSmall := []byte("artist 321 image small")

	errSpecialError := fmt.Errorf("finding image error")

	fakeIM := &libraryfakes.FakeArtistImageManager{
		FindAndSaveArtistImageStub: func(
			ctx context.Context,
			artistID int64,
			size library.ImageSize,
		) (io.ReadCloser, error) {
			if artistID == 42 {
				return nil, errSpecialError
			}

			if artistID != 321 {
				return nil, library.ErrArtworkNotFound
			}

			if size == library.SmallImage {
				return io.NopCloser(bytes.NewReader(imgBytesSmall)), nil
			}

			return io.NopCloser(bytes.NewReader(imgBytesOriginal)), nil
		},
	}

	aimgHandler := webserver.NewArtistImagesHandler(fakeIM)

	// Try with malformed URL which should return "not found" when variables are
	// not parsed by the Gorilla muxer.
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/artist/image", nil)
	aimgHandler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("no router: expected response code %d but got %d",
			http.StatusNotFound, resp.Code)
	}

	handler := routeArtistImageHandler(aimgHandler)

	// Test getting the original image.
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/artist/321/image", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("original: expected code %d but got %d", http.StatusOK, resp.Code)
	}

	if !bytes.Equal(imgBytesOriginal, resp.Body.Bytes()) {
		t.Errorf("original: expected image `%s` but got `%s`",
			imgBytesOriginal, resp.Body.Bytes())
	}

	// Test getting the small image.
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/artist/321/image?size=small", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("small: expected code %d but got %d", http.StatusOK, resp.Code)
	}

	if !bytes.Equal(imgBytesSmall, resp.Body.Bytes()) {
		t.Errorf("small: expected image `%s` but got `%s`",
			imgBytesSmall, resp.Body.Bytes())
	}

	// Try with artistID which is not an integer.
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/artist/boba/image", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("not found: expected response code %d but got %d",
			http.StatusBadRequest, resp.Code)
	}

	// Try with artistID which has not artwork according to the image finder.
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/artist/777/image", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("not found: expected response code %d but got %d",
			http.StatusNotFound, resp.Code)
	}

	// Make sure internal errors cause 500 status code and the error message
	// is part of the response body.
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/artist/42/image", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("internal error: expected response code %d but got %d",
			http.StatusInternalServerError, resp.Code)
	}

	respString := resp.Body.String()
	if !strings.Contains(respString, errSpecialError.Error()) {
		t.Errorf(
			"internal error: error not propagated to response body. It was: %s",
			respString,
		)
	}
}

// TestArtstImageHandlerDELETE tests what happens when artwork is removed.
func TestArtstImageHandlerDELETE(t *testing.T) {
	fakeIM := &libraryfakes.FakeArtistImageManager{
		RemoveArtistImageStub: func(ctx context.Context, albumID int64) error {
			if albumID != 42 {
				return fmt.Errorf("some error happened")
			}
			return nil
		},
	}

	aimgHandler := webserver.NewArtistImagesHandler(fakeIM)
	router := routeArtistImageHandler(aimgHandler)

	// Test removing an image.
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/artist/42/image", nil)
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Errorf("expected code %d but got %d", http.StatusNoContent, resp.Code)
	}

	if fakeIM.RemoveArtistImageCallCount() != 1 {
		t.Errorf("the image manager's RemoveArtistImage method was not called")
	}

	// Test what happens when an error occurs when removing an image.
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/v1/artist/55/image", nil)
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Errorf(
			"expected error to result in code %d but got %d",
			http.StatusInternalServerError,
			resp.Code,
		)
	}
}

// TestArtstImageHandlerPUT tests what happens when uploading images. It also simulates
// all possible errors from the image manager.
func TestArtstImageHandlerPUT(t *testing.T) {
	tests := []struct {
		desc         string
		aim          library.ArtistImageManager
		expectedCode int
	}{
		{
			desc: "uploaded image too big",
			aim: &libraryfakes.FakeArtistImageManager{
				SaveArtistImageStub: func(_ context.Context, _ int64, _ io.Reader) error {
					return library.ErrArtworkTooBig
				},
			},
			expectedCode: 413,
		},
		{
			desc: "artwork not of a good format",
			aim: &libraryfakes.FakeArtistImageManager{
				SaveArtistImageStub: func(_ context.Context, _ int64, _ io.Reader) error {
					return library.NewArtworkError("test error")
				},
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			desc: "internal error",
			aim: &libraryfakes.FakeArtistImageManager{
				SaveArtistImageStub: func(_ context.Context, _ int64, _ io.Reader) error {
					return fmt.Errorf("some general error")
				},
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			aimgHandler := webserver.NewArtistImagesHandler(test.aim)
			router := routeArtistImageHandler(aimgHandler)

			resp := httptest.NewRecorder()
			reqBody := bytes.NewReader([]byte("artwork body"))
			req := httptest.NewRequest(http.MethodPut, "/v1/artist/42/image", reqBody)
			router.ServeHTTP(resp, req)

			if resp.Code != test.expectedCode {
				t.Errorf("expected code %d but got %d", test.expectedCode, resp.Code)
			}
		})
	}

	// And now test actual uploading.
	var (
		uploadedImage bytes.Buffer
		requestBody   = []byte("the actual request body")
	)

	fakeAIM := &libraryfakes.FakeArtistImageManager{
		SaveArtistImageStub: func(_ context.Context, id int64, body io.Reader) error {
			if id != 42 {
				return fmt.Errorf("no such artist found")
			}

			if _, err := io.Copy(&uploadedImage, body); err != nil {
				return fmt.Errorf("error copying body: %w", err)
			}

			return nil
		},
	}
	aimgHandler := webserver.NewArtistImagesHandler(fakeAIM)
	router := routeArtistImageHandler(aimgHandler)

	resp := httptest.NewRecorder()
	reqBody := bytes.NewReader(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/v1/artist/42/image", reqBody)
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Errorf("expected code %d but got %d", http.StatusCreated, resp.Code)
	}

	uploadedBytes := uploadedImage.Bytes()
	if !bytes.Equal(uploadedBytes, requestBody) {
		t.Errorf("uploading corruption, expected `%s` but got `%s`",
			requestBody, uploadedBytes,
		)
	}
}

// routeArtistImageHandler wraps a handler the same way the web server will do when
// constructing the main application router. This is needed for tests so that the
// Gorilla mux variables will be parsed.
func routeArtistImageHandler(h http.Handler) http.Handler {
	router := mux.NewRouter()
	router.StrictSlash(true)
	router.UseEncodedPath()
	router.Handle(webserver.APIv1EndpointArtistImage, h).Methods(
		webserver.APIv1Methods[webserver.APIv1EndpointArtistImage]...,
	)

	return router
}
