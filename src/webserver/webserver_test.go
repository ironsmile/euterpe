package webserver

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ironsmile/httpms/src/config"
	"github.com/ironsmile/httpms/src/helpers"
	"github.com/ironsmile/httpms/src/library"
)

const (
	TestPort = 9092
	TestRoot = "http_root"
)

func testUrl() string {
	return fmt.Sprintf("http://127.0.0.1:%d/", TestPort)
}

func testErrorAfter(seconds time.Duration, message string) chan int {
	ch := make(chan int)

	go func() {
		select {
		case _ = <-ch:
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

func setUpServer() *Server {
	projRoot, err := getProjectRoot()

	if err != nil {
		println(err)
		os.Exit(1)
	}

	var wsCfg config.Config
	wsCfg.Listen = fmt.Sprintf("127.0.0.1:%d", TestPort)
	wsCfg.HTTPRoot = filepath.Join(projRoot, "test_files", TestRoot)
	wsCfg.Gzip = true

	return NewServer(wsCfg, nil)
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
	url := fmt.Sprintf("%s://127.0.0.1:%d", proto, TestPort)

	_, err := http.Get(url)
	_, err = http.Get(url)

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

	lib, err := library.NewLocalLibrary("/tmp/test-web-file-get.db")

	if err != nil {
		t.Fatal(err)
	}

	err = lib.Initialize()

	if err != nil {
		t.Error(err)
	}

	lib.AddLibraryPath(filepath.Join(projRoot, "test_files", "library"))
	lib.Scan()

	ch := testErrorAfter(5, "Library in TestGetFileUrl did not finish scaning on time")
	lib.WaitScan()
	ch <- 42

	var wsCfg config.Config
	wsCfg.Listen = fmt.Sprintf("127.0.0.1:%d", TestPort)
	wsCfg.HTTPRoot = filepath.Join(projRoot, "test_files", TestRoot)

	srv := NewServer(wsCfg, lib)
	srv.Serve()

	return srv, lib
}

func TestStaticFilesServing(t *testing.T) {
	srv := setUpServer()
	srv.Serve()
	defer tearDownServer(srv)

	testStaticFile := func(url, expected string) {

		resp, err := http.Get(url)

		if err != nil {
			t.Error(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Unexpected response status code: %d", resp.StatusCode)
		}

		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			t.Error(err)
		}

		if string(body) != expected {
			t.Errorf("Wrong static file found: %s", string(body))
		}
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/static", TestPort)
	testStaticFile(url, "This is a static file")

	url = fmt.Sprintf("http://127.0.0.1:%d/second/static", TestPort)
	testStaticFile(url, "Second static file")
}

func TestStartAndStop(t *testing.T) {

	_, err := http.Get(testUrl())

	if err == nil {
		t.Fatalf("Something is running on testing port %d", TestPort)
	}

	srv := setUpServer()
	srv.Serve()

	_, err = http.Get(testUrl())

	if err != nil {
		t.Errorf("Web server is not running %d", TestPort)
	}

	srv.Stop()

	ch := testErrorAfter(2, "Web server did not stop in time")
	srv.Wait()
	ch <- 42

	_, err = http.Get(testUrl())

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
	wsCfg.Listen = fmt.Sprintf("127.0.0.1:%d", TestPort)
	wsCfg.HTTPRoot = TestRoot
	wsCfg.SSL = true
	wsCfg.SSLCertificate = config.ConfigCert{
		Crt: filepath.Join(certDir, "cert.pem"),
		Key: filepath.Join(certDir, "key.pem"),
	}

	srv := NewServer(wsCfg, nil)
	srv.Serve()

	defer tearDownServer(srv)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	_, err = client.Get(fmt.Sprintf("https://127.0.0.1:%d", TestPort))

	if err != nil {
		t.Errorf("Error GETing a SSL url: %s", err)
	}
}

func TestUserAuthentication(t *testing.T) {
	url := fmt.Sprintf("http://127.0.0.1:%d/static", TestPort)

	projRoot, err := getProjectRoot()

	if err != nil {
		t.Error(err)
	}

	var wsCfg config.Config
	wsCfg.Listen = fmt.Sprintf("127.0.0.1:%d", TestPort)
	wsCfg.HTTPRoot = filepath.Join(projRoot, "test_files", TestRoot)
	wsCfg.Auth = true
	wsCfg.Authenticate = config.ConfigAuth{
		User:     "testuser",
		Password: "testpass",
	}

	srv := NewServer(wsCfg, nil)
	srv.Serve()
	defer tearDownServer(srv)

	resp, err := http.Get(url)

	if err != nil {
		t.Error(err)
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
		t.Error(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 but got: %d", resp.StatusCode)
	}

	req, _ = http.NewRequest("GET", url, nil)
	req.SetBasicAuth("wronguser", "wrongpass")
	resp, err = client.Do(req)

	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("Expected 401 but got: %d", resp.StatusCode)
	}
}

func TestSearchUrl(t *testing.T) {
	projRoot, _ := getProjectRoot()

	lib, _ := library.NewLocalLibrary("/tmp/test-web-search.db")
	err := lib.Initialize()

	if err != nil {
		t.Error(err)
	}

	defer lib.Truncate()

	lib.AddLibraryPath(filepath.Join(projRoot, "test_files", "library"))
	lib.Scan()

	ch := testErrorAfter(5, "Library in TestSearchUrl did not finish scaning on time")
	lib.WaitScan()
	ch <- 42

	var wsCfg config.Config
	wsCfg.Listen = fmt.Sprintf("127.0.0.1:%d", TestPort)
	wsCfg.HTTPRoot = filepath.Join(projRoot, "test_files", TestRoot)

	srv := NewServer(wsCfg, lib)
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

	url := fmt.Sprintf("http://127.0.0.1:%d/search/Album+Of+Tests", TestPort)
	resp, err := http.Get(url)

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

	responseBody, err := ioutil.ReadAll(resp.Body)

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

	url = fmt.Sprintf("http://127.0.0.1:%d/search/Not+There", TestPort)
	resp, err = http.Get(url)

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Unexpected response status code: %d", resp.StatusCode)
	}

	responseBody, err = ioutil.ReadAll(resp.Body)

	var noResults []library.SearchResult

	err = json.Unmarshal(responseBody, &noResults)

	if err != nil {
		t.Error(err)
	}

	if len(noResults) != 0 {
		t.Errorf("Expected no results from search but they were %d", len(noResults))
	}
}

func TestGetFileUrl(t *testing.T) {
	srv, lib := getLibraryServer(t)
	defer lib.Truncate()
	defer tearDownServer(srv)

	found := lib.Search("Buggy Bugoff")

	if len(found) != 1 {
		t.Fatalf("Problem finding Buggy Bugoff test track")
	}

	trackID := found[0].ID

	url := fmt.Sprintf("http://127.0.0.1:%d/file/%d", TestPort, trackID)

	resp, err := http.Get(url)

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Unexpected response status code: %d", resp.StatusCode)
	}

	responseBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		t.Fatal(err)
	}

	if len(responseBody) != 17314 {
		t.Errorf("Track size was not as expected. It was %d", len(responseBody))
	}
}

func TestGzipEncoding(t *testing.T) {
	projRoot, err := getProjectRoot()

	if err != nil {
		t.Fatal(err)
	}

	testGzipResponse := func(tests [][2]string) {
		url := fmt.Sprintf("http://127.0.0.1:%d/static", TestPort)
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

			var responseBody []byte
			if contentEncoding == "gzip" {
				reader, err := gzip.NewReader(resp.Body)
				if err != nil {
					t.Fatal(err)
				}
				defer reader.Close()
				responseBody, err = ioutil.ReadAll(reader)
			} else {
				responseBody, err = ioutil.ReadAll(resp.Body)
			}

			if err != nil {
				t.Fatal(err)
			}

			if len(responseBody) != 21 {
				t.Errorf("Expected response size 21 but found %d", len(responseBody))
			}

			if string(responseBody) != "This is a static file" {
				t.Errorf("Returned file was not the one expected")
			}
		}
	}

	var wsCfg config.Config
	wsCfg.Listen = fmt.Sprintf("127.0.0.1:%d", TestPort)
	wsCfg.HTTPRoot = filepath.Join(projRoot, "test_files", TestRoot)
	wsCfg.Gzip = true

	srv := NewServer(wsCfg, nil)
	srv.Serve()

	tests := [][2]string{
		{"gzip, deflate", "gzip"},
		{"gzip", "gzip"},
		{"identity", ""},
	}

	testGzipResponse(tests)

	tearDownServer(srv)

	wsCfg.Gzip = false
	srv = NewServer(wsCfg, nil)
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
	defer lib.Truncate()
	defer tearDownServer(srv)

	found := lib.Search("Buggy Bugoff")

	if len(found) != 1 {
		t.Fatalf("Problem finding Buggy Bugoff test track")
	}

	trackID := found[0].ID

	url := fmt.Sprintf("http://127.0.0.1:%d/file/%d", TestPort, trackID)

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

func TestAlbumHandlerOverHttp(t *testing.T) {
	srv, lib := getLibraryServer(t)
	defer lib.Truncate()
	defer tearDownServer(srv)

	artistID, _ := lib.(*library.LocalLibrary).GetArtistID("Artist Testoff")
	albumID, _ := lib.(*library.LocalLibrary).GetAlbumID("Album Of Tests", artistID)

	albumURL := fmt.Sprintf("http://127.0.0.1:%d/album/%d", TestPort, albumID)

	resp, err := http.Get(albumURL)

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
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

	albumURL = fmt.Sprintf("http://127.0.0.1:%d/album/666", TestPort)

	resp, err = http.Get(albumURL)

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("Unexpected response status code: %d", resp.StatusCode)
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

	err = albumHandler.writeZipContents(buf, files)

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

		if zippedFile.FileHeader.UncompressedSize != uint32(st.Size()) {
			t.Errorf("Zipped file %s was incorect size: %d. Expected %d",
				zippedFile.Name, zippedFile.FileHeader.UncompressedSize, st.Size())
		}
	}
}
