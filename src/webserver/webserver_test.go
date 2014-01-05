package webserver

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	TestPort = 9092
	TestRoot = "http_test"
)

func testUrl() string {
	return fmt.Sprintf("http://127.0.0.1:%d/", TestPort)
}

func stopAfter(seconds time.Duration, message string) chan int {
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

func TestStartAndStop(t *testing.T) {

	_, err := http.Get(testUrl())

	if err == nil {
		t.Fatalf("Something is running on testing port %d", TestPort)
	}

	var wsCfg ServerConfig
	wsCfg.Address = fmt.Sprintf(":%d", TestPort)
	wsCfg.Root = TestRoot

	srv := NewServer(wsCfg)
	srv.Serve()

	_, err = http.Get(testUrl())

	if err != nil {
		t.Errorf("Web server is not running %d", TestPort)
	}

	srv.Stop()

	ch := stopAfter(2, "Web server did not stop in time")
	srv.Wait()
	ch <- 42

	_, err = http.Get(testUrl())

	if err == nil {
		t.Errorf("The webserver was not stopped")
	}
}

func TestStaticFilesServing(t *testing.T) {

}

func TestSearchUrl(t *testing.T) {

}

func TestGetFileUrl(t *testing.T) {

}

func TestSSL(t *testing.T) {

}

func TestUserAuthentication(t *testing.T) {

}

func TestDefaultPorts(t *testing.T) {

}
