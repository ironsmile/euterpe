package webserver

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ironsmile/euterpe/src/config"
	"github.com/ironsmile/euterpe/src/helpers"
	"github.com/ironsmile/euterpe/src/library"
)

const (
	testPort = 9092
)

func testURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/", testPort)
}

func testErrorAfter(seconds time.Duration, message string) chan int {
	ch := make(chan int)

	go func() {
		select {
		case <-ch:
			close(ch)
			return
		case <-time.After(seconds * time.Second):
			close(ch)
			println(message)
			os.Exit(1)
		}
	}()

	return ch
}

// getTestFileSystems returns the HTTP root FS and the HTML templates FS to be
// used throughout webserver tests. They are the same as the one to be used
// in the actual binary. But using os.DirFS instead of embed.FS.
func getTestFileSystems() (fs.FS, fs.FS) {
	return os.DirFS("../../http_root"), os.DirFS("../../templates")
}

func setUpServer() *Server {
	var wsCfg config.Config
	wsCfg.Listen = fmt.Sprintf("127.0.0.1:%d", testPort)
	wsCfg.Gzip = true

	httpFS, templatesFS := getTestFileSystems()
	return NewServer(context.Background(), wsCfg, nil, httpFS, templatesFS)
}

func tearDownServer(srv *Server) {
	srv.Stop()
	ch := testErrorAfter(2, "Web server did not stop in time")
	srv.Wait()
	ch <- 42

	proto := "http"
	if srv.cfg.SSL {
		proto = "https"
	}
	url := fmt.Sprintf("%s://127.0.0.1:%d", proto, testPort)

	_, _ = http.Get(url)
	_, err := http.Get(url)

	if err == nil {
		println("Web server did not stop")
		os.Exit(1)
	}
}

func getProjectRoot() (string, error) {
	path, err := filepath.Abs(filepath.FromSlash("../.."))
	if err != nil {
		return "", err
	}
	return path, nil
}

func getLibraryServer(t *testing.T) (*Server, library.Library) {
	projRoot, _ := getProjectRoot()

	sqlsFS := os.DirFS("../../sqls")
	lib, err := library.NewLocalLibrary(
		context.TODO(),
		library.SQLiteMemoryFile,
		sqlsFS,
	)

	if err != nil {
		t.Fatal(err)
	}

	err = lib.Initialize()

	if err != nil {
		t.Error(err)
	}

	lib.AddLibraryPath(filepath.Join(projRoot, "test_files", "library"))

	ch := testErrorAfter(5, "Library in TestGetFileUrl did not finish scaning on time")
	lib.Scan()
	ch <- 42

	var wsCfg config.Config
	wsCfg.Listen = fmt.Sprintf("127.0.0.1:%d", testPort)

	httpFS, templatesFS := getTestFileSystems()
	srv := NewServer(
		context.Background(),
		wsCfg,
		lib,
		httpFS,
		templatesFS,
	)
	srv.Serve()

	return srv, lib
}

func TestStaticFilesServing(t *testing.T) {
	srv := setUpServer()
	srv.Serve()
	defer tearDownServer(srv)

	url := fmt.Sprintf("http://127.0.0.1:%d/index.html", testPort)
	resp, err := http.Get(url)

	if err != nil {
		t.Error(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Unexpected response status code: %d", resp.StatusCode)
	}

	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Error(err)
	}
}

func TestStartAndStop(t *testing.T) {

	_, err := http.Get(testURL())

	if err == nil {
		t.Fatalf("Something is running on testing port %d", testPort)
	}

	srv := setUpServer()
	srv.Serve()

	_, err = http.Get(testURL())

	if err != nil {
		t.Errorf("Web server is not running %d", testPort)
	}

	srv.Stop()

	ch := testErrorAfter(2, "Web server did not stop in time")
	srv.Wait()
	ch <- 42

	_, err = http.Get(testURL())

	if err == nil {
		t.Errorf("The webserver was not stopped")
	}
}

func TestSSL(t *testing.T) {

	projectRoot, err := getProjectRoot()
	if err != nil {
		t.Fatalf("Could not determine project path: %s", err)
	}
	certDir := filepath.Join(projectRoot, "test_files", "ssl")

	var wsCfg config.Config
	wsCfg.Listen = fmt.Sprintf("127.0.0.1:%d", testPort)
	wsCfg.SSL = true
	wsCfg.SSLCertificate = config.Cert{
		Crt: filepath.Join(certDir, "cert.pem"),
		Key: filepath.Join(certDir, "key.pem"),
	}

	httpFS, templatesFS := getTestFileSystems()
	srv := NewServer(context.Background(), wsCfg, nil, httpFS, templatesFS)
	srv.Serve()

	defer tearDownServer(srv)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	} // #nosec
	client := &http.Client{Transport: tr}
	_, err = client.Get(fmt.Sprintf("https://127.0.0.1:%d", testPort))

	if err != nil {
		t.Errorf("Error GETing a SSL url: %s", err)
	}
}

func TestUserAuthentication(t *testing.T) {
	url := fmt.Sprintf("http://127.0.0.1:%d/", testPort)

	var wsCfg config.Config
	wsCfg.Listen = fmt.Sprintf("127.0.0.1:%d", testPort)
	wsCfg.Auth = true
	wsCfg.Authenticate = config.Auth{
		User:     "testuser",
		Password: "testpass",
	}

	httpFS, templatesFS := getTestFileSystems()
	srv := NewServer(context.Background(), wsCfg, nil, httpFS, templatesFS)
	srv.Serve()
	defer tearDownServer(srv)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401 but got: %d", resp.StatusCode)
	}

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth("testuser", "testpass")
	resp, err = client.Do(req)

	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 but got: %d", resp.StatusCode)
	}

	req, _ = http.NewRequest("GET", url, nil)
	req.SetBasicAuth("wronguser", "wrongpass")
	resp, err = client.Do(req)

	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401 but got: %d", resp.StatusCode)
	}
}

func TestSearchUrl(t *testing.T) {
	projRoot, _ := getProjectRoot()

	sqlsFS := os.DirFS("../../sqls")
	lib, _ := library.NewLocalLibrary(context.TODO(), library.SQLiteMemoryFile, sqlsFS)
	err := lib.Initialize()

	if err != nil {
		t.Error(err)
	}

	defer func() { _ = lib.Truncate() }()

	lib.AddLibraryPath(filepath.Join(projRoot, "test_files", "library"))

	ch := testErrorAfter(5, "Library in TestSearchUrl did not finish scaning on time")
	lib.Scan()
	ch <- 42

	var wsCfg config.Config
	wsCfg.Listen = fmt.Sprintf("127.0.0.1:%d", testPort)

	httpFS, templatesFS := getTestFileSystems()
	srv := NewServer(context.Background(), wsCfg, lib, httpFS, templatesFS)
	srv.Serve()
	defer tearDownServer(srv)

	/*
		The expected
		[
			{title:"", album:"", artist:"", track:0, id:0},
			...
			{title:"", album:"", artist:"", track:0, id:0}
		]
	*/

	searchURLs := []string{
		// Backward compatibility must be kept for the sake of all 1.0.0 clients
		fmt.Sprintf("http://127.0.0.1:%d/search/Album+Of+Tests", testPort),

		// The new way of searching which makes it possible to add additional parameters
		// to the search.
		fmt.Sprintf("http://127.0.0.1:%d/search/?q=Album+Of+Tests", testPort),
	}

	for _, searchURL := range searchURLs {
		resp, err := http.Get(searchURL)

		if err != nil {
			t.Fatal(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Unexpected response status code: %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")

		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Wrong content-type: %s", contentType)
		}

		responseBody, err := io.ReadAll(resp.Body)

		if err != nil {
			t.Error(err)
		}

		var results []library.SearchResult

		err = json.Unmarshal(responseBody, &results)

		if err != nil {
			t.Error(err)
		}

		if len(results) != 2 {
			t.Errorf("Expected two results from search but they were %d", len(results))
		}

		for _, result := range results {
			if result.Album != "Album Of Tests" {
				t.Errorf("Wrong album in search results: %s", result.Album)
			}
		}
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/search/Not+There", testPort)
	resp, err := http.Get(url)

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Unexpected response status code: %d", resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}

	var noResults []library.SearchResult

	err = json.Unmarshal(responseBody, &noResults)

	if err != nil {
		t.Error(err)
	}

	if len(noResults) != 0 {
		t.Errorf("Expected no results from search but they were %d", len(noResults))
	}
}

// TestGetFileURL runs a real web server and then makes sure that the URL for getting
// a file returns the expected file.
func TestGetFileURL(t *testing.T) {
	srv, lib := getLibraryServer(t)
	defer func() { _ = lib.Truncate() }()
	defer tearDownServer(srv)

	found := lib.Search(library.SearchArgs{Query: "Buggy Bugoff"})

	if len(found) != 1 {
		t.Fatalf("Problem finding Buggy Bugoff test track")
	}

	trackID := found[0].ID

	url := fmt.Sprintf("http://127.0.0.1:%d/v1/file/%d", testPort, trackID)

	resp, err := http.Get(url)

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Unexpected response status code: %d", resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)

	if err != nil {
		t.Fatal(err)
	}

	if len(responseBody) != 17314 {
		t.Errorf("Track size was not as expected. It was %d", len(responseBody))
	}

	// Try with a file which is not in the library.
	url = fmt.Sprintf("http://127.0.0.1:%d/v1/file/7742", testPort)
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Unexpected response status code: %d", resp.StatusCode)
	}
}

func TestGzipEncoding(t *testing.T) {
	testGzipResponse := func(tests [][2]string) {
		url := fmt.Sprintf("http://127.0.0.1:%d/", testPort)
		for _, test := range tests {
			header := test[0]
			expected := test[1]
			client := &http.Client{}
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Add("Accept-Encoding", header)
			resp, err := client.Do(req)

			if err != nil {
				t.Error(err)
			}
			defer resp.Body.Close()

			contentEncoding := resp.Header.Get("Content-Encoding")
			if contentEncoding != expected {
				t.Errorf("Expected Content-Encoding `%s` but found `%s`", expected,
					contentEncoding)
			}

			var bodyReader io.Reader
			if contentEncoding == "gzip" {
				reader, err := gzip.NewReader(resp.Body)
				if err != nil {
					t.Fatal(err)
				}
				defer reader.Close()
				bodyReader = reader
			} else {
				bodyReader = resp.Body
			}

			_, err = io.ReadAll(bodyReader)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	var wsCfg config.Config
	wsCfg.Listen = fmt.Sprintf("127.0.0.1:%d", testPort)
	wsCfg.Gzip = true
	httpFS, templatesFS := getTestFileSystems()

	srv := NewServer(context.Background(), wsCfg, nil, httpFS, templatesFS)
	srv.Serve()

	tests := [][2]string{
		{"gzip, deflate", "gzip"},
		{"gzip", "gzip"},
		{"identity", ""},
	}

	testGzipResponse(tests)

	tearDownServer(srv)

	wsCfg.Gzip = false
	srv = NewServer(context.Background(), wsCfg, nil, httpFS, templatesFS)
	srv.Serve()
	defer tearDownServer(srv)

	tests = [][2]string{
		{"gzip, deflate", ""},
		{"gzip", ""},
		{"identity", ""},
	}

	testGzipResponse(tests)
}

func TestFileNameHeaders(t *testing.T) {
	srv, lib := getLibraryServer(t)
	defer func() { _ = lib.Truncate() }()
	defer tearDownServer(srv)

	found := lib.Search(library.SearchArgs{Query: "Buggy Bugoff"})

	if len(found) != 1 {
		t.Fatalf("Problem finding Buggy Bugoff test track")
	}

	trackID := found[0].ID

	url := fmt.Sprintf("http://127.0.0.1:%d/file/%d", testPort, trackID)

	resp, err := http.Get(url)

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Unexpected response status code: %d", resp.StatusCode)
	}

	expected := "filename=\"third_file.mp3\""
	nameHeader := resp.Header.Get("Content-Disposition")

	if nameHeader != expected {
		t.Errorf("Expected filename `%s` but found `%s`", expected, nameHeader)
	}
}

// TestAlbumHandlerOverHTTP starts a web server and then checks that GETing the album
// returns status OK for good requests. Then tests with strange requests.
func TestAlbumHandlerOverHTTP(t *testing.T) {
	srv, lib := getLibraryServer(t)
	defer func() { _ = lib.Truncate() }()
	defer tearDownServer(srv)

	albumPaths, err := lib.(*library.LocalLibrary).GetAlbumFSPathByName("Album Of Tests")

	if err != nil {
		t.Fatalf("Cannot get album path: %s", err)
	}

	albumID, _ := lib.(*library.LocalLibrary).GetAlbumID("Album Of Tests", albumPaths[0])

	albumURL := fmt.Sprintf("http://127.0.0.1:%d/album/%d", testPort, albumID)

	resp, err := http.Get(albumURL)

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Unexpected response status code: %d", resp.StatusCode)
	}

	headers := map[string]string{
		"Content-Type":        "application/zip",
		"Content-Disposition": `filename="Album Of Tests.zip"`,
	}

	for header := range headers {
		expected := headers[header]
		found := resp.Header.Get(header)
		if found != expected {
			t.Errorf("Expected %s: %s but it was `%s`", header, expected, found)
		}
	}

	albumURL = fmt.Sprintf("http://127.0.0.1:%d/album/666", testPort)

	resp, err = http.Get(albumURL)

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Unexpected response status code: %d", resp.StatusCode)
	}

	// Test with malformed URL.
	albumURL = fmt.Sprintf("http://127.0.0.1:%d/album/foo", testPort)
	resp, err = http.Get(albumURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Unexpected status code for bogus request: %d", resp.StatusCode)
	}
}

func TestAlbumHandlerZipFunction(t *testing.T) {
	buf := new(bytes.Buffer)

	projRoot, err := helpers.ProjectRoot()

	if err != nil {
		t.Fatalf("Was not able to find test_files directory: %s", err)
	}

	testLibraryPath := filepath.Join(projRoot, "test_files", "library")

	files := []string{
		filepath.Join(testLibraryPath, "test_file_one.mp3"),
		filepath.Join(testLibraryPath, "test_file_two.mp3"),
	}

	albumHandler := new(AlbumHandler)

	_, err = albumHandler.writeZipContents(buf, files)
	if err != nil {
		t.Error(err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))

	if err != nil {
		t.Fatal(err)
	}

	if len(reader.File) != 2 {
		t.Errorf("Expected two files in the zip but found %d", len(reader.File))
	}

	for _, zippedFile := range reader.File {
		fsPath := filepath.Join(testLibraryPath, zippedFile.Name)

		st, err := os.Stat(fsPath)

		if err != nil {
			t.Errorf("zipped file %s not found on file system: %s", zippedFile.Name,
				err)
			continue
		}

		if zippedFile.FileHeader.UncompressedSize64 != uint64(st.Size()) {
			t.Errorf("Zipped file %s was incorrect size: %d. Expected %d",
				zippedFile.Name, zippedFile.FileHeader.UncompressedSize64, st.Size())
		}
	}
}
