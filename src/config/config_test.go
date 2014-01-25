package config

import (
	"path/filepath"
	"testing"
)

func TestFindTheRighConfigFile(t *testing.T) {
	cfg := new(Config)
	cfg.UserPath = filepath.FromSlash("/some/path")

	found := cfg.UserConfigPath()
	expected := filepath.Join(cfg.UserPath, "config.json")

	if found != expected {
		t.Errorf("Expected %s but found %s", expected, found)
	}

	cfg.UserPath = ""
	found = cfg.UserConfigPath()

	if !filepath.IsAbs(found) {
		t.Errorf("User config path was not rooted: %s", found)
	}

	if len(found) < 1 {
		t.Errorf("User config path was empty")
	}
}

func TestMergingConfigs(t *testing.T) {
	cfg := new(Config)
	merged := new(Config)

	cfg.SSL = true

	cfg.merge(merged)

	if cfg.SSL != true {
		t.Errorf("Zero value from the merged has been copied over")
	}

	merged.Listen = ":http"

	cfg.merge(merged)

	if cfg.Listen != ":http" {
		t.Errorf("NonZero value has not been copied over")
	}

	cfg.SSL = false
	cfg.Listen = ":80"
	cfg.LogFile = "logfile"
	cfg.Gzip = true
	cfg.ReadTimeout = 10
	cfg.WriteTimeout = 10
	cfg.MaxHeadersSize = 100
	cfg.SqliteDatabase = "httpms.db"
	cfg.Auth = true
	cfg.Authenticate = ConfigAuth{User: "bob", Password: "marley"}

	merged.Listen = ":8080"
	merged.SSL = true
	merged.SSLCertificate = ConfigCert{Crt: "crt", Key: "key"}
	merged.Libraries = append(merged.Libraries, "/some/path")
	merged.Auth = false

	cfg.merge(merged)

	if cfg.SSL != true {
		t.Errorf("SSL was false but it was expected to be true")
	}

	if cfg.Auth != false {
		t.Errorf("Auth was true but it was expected to be false")
	}

	if cfg.SSLCertificate.Crt != "crt" || cfg.SSLCertificate.Key != "key" {
		t.Errorf("SSL Certificate was not as expected: %#v", cfg.SSLCertificate)
	}

	if cfg.Authenticate.User != "bob" || cfg.Authenticate.Password != "marley" {
		t.Errorf("Authenticate user and password were wrong: %#v", cfg.Authenticate)
	}

	if cfg.Listen != ":8080" {
		t.Errorf("Listen was %s", cfg.Listen)
	}

	if cfg.Gzip != true {
		t.Errorf("Gzip was %t", cfg.Gzip)
	}

	if cfg.ReadTimeout != 10 {
		t.Errorf("ReadTimeout was %d", cfg.ReadTimeout)
	}

	if cfg.WriteTimeout != 10 {
		t.Errorf("WriteTimeout was %d", cfg.WriteTimeout)
	}

	if cfg.MaxHeadersSize != 100 {
		t.Errorf("MaxHeadersSize was %d", cfg.MaxHeadersSize)
	}

	if cfg.SqliteDatabase != "httpms.db" {
		t.Errorf("SqliteDatabase was %s", cfg.SqliteDatabase)
	}

	if len(cfg.Libraries) != 1 {
		t.Errorf("Libraries was not as expected: %#v", cfg.Libraries)
	} else {
		if cfg.Libraries[0] != "/some/path" {
			t.Errorf("Library was wrong: %s", cfg.Libraries[0])
		}
	}
}
