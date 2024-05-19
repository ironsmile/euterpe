package webserver_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gorilla/mux"
	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/library/libraryfakes"
	"github.com/ironsmile/euterpe/src/webserver"
)

// TestAlbumArtworkHandler makes sure the artwork handler is processing the HTTP
// request correctly and sending the expected arguments to its artwork manager.
func TestAlbumArtworkHandler(t *testing.T) {
	imgBytesOriginal := []byte("album 321 image original")
	imgBytesSmall := []byte("album 321 image small")

	errSpecialError := fmt.Errorf("finding image error")

	fakeAM := &libraryfakes.FakeArtworkManager{
		FindAndSaveAlbumArtworkStub: func(
			ctx context.Context,
			albumID int64,
			size library.ImageSize,
		) (io.ReadCloser, error) {
			if albumID == 42 {
				return nil, errSpecialError
			}

			if albumID != 321 {
				return nil, library.ErrArtworkNotFound
			}

			if size == library.SmallImage {
				return io.NopCloser(bytes.NewReader(imgBytesSmall)), nil
			}

			return io.NopCloser(bytes.NewReader(imgBytesOriginal)), nil
		},
	}

	const (
		notFoundImage         = "images/notfound.png"
		notFoundImageContents = "not-found-image"
	)
	testFS := fstest.MapFS{
		notFoundImage: &fstest.MapFile{
			Data:    []byte(notFoundImageContents),
			Mode:    0644,
			ModTime: time.Now(),
		},
	}

	aartHandler := webserver.NewAlbumArtworkHandler(
		fakeAM,
		testFS,
		notFoundImage,
	)

	// Try with malformed URL which should return "not found" when variables are
	// not parsed by the Gorilla muxer.
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/album/artwork", nil)
	aartHandler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("no router: expected response code %d but got %d",
			http.StatusNotFound, resp.Code)
	}

	handler := routeAlbumArtworkHandler(aartHandler)

	// Test getting the original image.
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/album/321/artwork", nil)
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
	req = httptest.NewRequest(http.MethodGet, "/v1/album/321/artwork?size=small", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("small: expected code %d but got %d", http.StatusOK, resp.Code)
	}

	if !bytes.Equal(imgBytesSmall, resp.Body.Bytes()) {
		t.Errorf("small: expected image `%s` but got `%s`",
			imgBytesSmall, resp.Body.Bytes())
	}

	// Try with albumID which is not an integer.
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/album/boba/artwork", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("not found: expected response code %d but got %d",
			http.StatusBadRequest, resp.Code)
	}

	// Try with albumID which has not artwork according to the image finder.
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/album/777/artwork", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("not found: expected response code %d but got %d",
			http.StatusNotFound, resp.Code)
	}

	respBody := resp.Body.String()
	if respBody != notFoundImageContents {
		t.Errorf(
			"not found: expected body `%s` but got `%s`",
			notFoundImageContents,
			respBody,
		)
	}

	// Make sure internal errors cause 500 status code and the error message
	// is part of the response body.
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/album/42/artwork", nil)
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

	// Test the HEAD request. It should return the Content-Length and 200 for
	// images which are present.
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodHead, "/v1/album/321/artwork?size=small", nil)
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("HEAD small: expected code %d but got %d", http.StatusOK, resp.Code)
	}

	clExpected := strconv.FormatInt(int64(len(imgBytesSmall)), 10)
	if cl := resp.Result().Header.Get("Content-Length"); clExpected != cl {
		t.Errorf("HEAD small: expected Content-Length `%s` but got `%s`",
			clExpected, cl)
	}
	respBodySize, err := io.Copy(io.Discard, resp.Result().Body)
	if err != nil || respBodySize != 0 {
		t.Errorf("HEAD response should have empty body")
	}
}

// TestAlbumArtworkHandlerDELETE tests what happens when artwork is removed.
func TestAlbumArtworkHandlerDELETE(t *testing.T) {
	fakeAM := &libraryfakes.FakeArtworkManager{
		RemoveAlbumArtworkStub: func(ctx context.Context, albumID int64) error {
			if albumID != 42 {
				return fmt.Errorf("some error happened")
			}
			return nil
		},
	}

	const (
		notFoundImage         = "images/notfound.png"
		notFoundImageContents = "not-found-image"
	)
	testFS := fstest.MapFS{
		notFoundImage: &fstest.MapFile{
			Data:    []byte(notFoundImageContents),
			Mode:    0644,
			ModTime: time.Now(),
		},
	}
	aimgHandler := webserver.NewAlbumArtworkHandler(
		fakeAM,
		testFS,
		notFoundImageContents,
	)
	router := routeAlbumArtworkHandler(aimgHandler)

	// Test removing an image.
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/album/42/artwork", nil)
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Errorf("expected code %d but got %d", http.StatusNoContent, resp.Code)
	}

	if fakeAM.RemoveAlbumArtworkCallCount() != 1 {
		t.Errorf("the image manager's RemoveAlbumArtwork method was not called")
	}

	// Test what happens when an error occurs when removing an image.
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/v1/album/55/artwork", nil)
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Errorf(
			"expected error to result in code %d but got %d",
			http.StatusInternalServerError,
			resp.Code,
		)
	}
}

// TestAlbumArtworkHandlerPUT tests what happens when uploading images. It also simulates
// all possible errors from the image manager.
func TestAlbumArtworkHandlerPUT(t *testing.T) {

	const (
		notFoundImage         = "images/notfound.png"
		notFoundImageContents = "not-found-image"
	)
	testFS := fstest.MapFS{
		notFoundImage: &fstest.MapFile{
			Data:    []byte(notFoundImageContents),
			Mode:    0644,
			ModTime: time.Now(),
		},
	}

	tests := []struct {
		desc         string
		aim          library.ArtworkManager
		expectedCode int
	}{
		{
			desc: "uploaded image too big",
			aim: &libraryfakes.FakeArtworkManager{
				SaveAlbumArtworkStub: func(_ context.Context, _ int64, _ io.Reader) error {
					return library.ErrArtworkTooBig
				},
			},
			expectedCode: 413,
		},
		{
			desc: "artwork not of a good format",
			aim: &libraryfakes.FakeArtworkManager{
				SaveAlbumArtworkStub: func(_ context.Context, _ int64, _ io.Reader) error {
					return library.NewArtworkError(errors.New("test error"))
				},
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			desc: "internal error",
			aim: &libraryfakes.FakeArtworkManager{
				SaveAlbumArtworkStub: func(_ context.Context, _ int64, _ io.Reader) error {
					return fmt.Errorf("some general error")
				},
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {

			aimgHandler := webserver.NewAlbumArtworkHandler(
				test.aim,
				testFS,
				notFoundImageContents,
			)
			router := routeAlbumArtworkHandler(aimgHandler)

			resp := httptest.NewRecorder()
			reqBody := bytes.NewReader([]byte("artwork body"))
			req := httptest.NewRequest(http.MethodPut, "/v1/album/42/artwork", reqBody)
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

	fakeAIM := &libraryfakes.FakeArtworkManager{
		SaveAlbumArtworkStub: func(_ context.Context, id int64, body io.Reader) error {
			if id != 42 {
				return fmt.Errorf("no such album found")
			}

			if _, err := io.Copy(&uploadedImage, body); err != nil {
				return fmt.Errorf("error copying body: %w", err)
			}

			return nil
		},
	}
	aimgHandler := webserver.NewAlbumArtworkHandler(
		fakeAIM,
		testFS,
		notFoundImageContents,
	)
	router := routeAlbumArtworkHandler(aimgHandler)

	resp := httptest.NewRecorder()
	reqBody := bytes.NewReader(requestBody)
	req := httptest.NewRequest(http.MethodPut, "/v1/album/42/artwork", reqBody)
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

// routeAlbumArtworkHandler wraps a handler the same way the web server will do when
// constructing the main application router. This is needed for tests so that the
// Gorilla mux variables will be parsed.
func routeAlbumArtworkHandler(h http.Handler) http.Handler {
	router := mux.NewRouter()
	router.StrictSlash(true)
	router.UseEncodedPath()
	router.Handle(webserver.APIv1EndpointAlbumArtwork, h).Methods(
		webserver.APIv1Methods[webserver.APIv1EndpointAlbumArtwork]...,
	)

	return router
}
