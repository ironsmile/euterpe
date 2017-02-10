package config

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
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

// Helper function for TestMergingConfigs
func getDefaultCfg() *Config {
	dflConfig := new(Config)
	dflConfig.SSL = false
	dflConfig.Listen = ":80"
	dflConfig.LogFile = "logfile"
	dflConfig.Gzip = true
	dflConfig.ReadTimeout = 10
	dflConfig.WriteTimeout = 10
	dflConfig.MaxHeadersSize = 100
	dflConfig.SqliteDatabase = "httpms.db"
	dflConfig.Auth = true
	dflConfig.Authenticate = Auth{User: "bob", Password: "marley"}
	dflConfig.LibraryScan = ScanSection{
		FilesPerOperation: 1000,
		SleepPerOperation: 10 * time.Millisecond,
		InitialWait:       500 * time.Millisecond,
	}
	return dflConfig
}

// checkMerge is a helper function for TestMergingConfigs
func checkMerge(t *testing.T, cfg *Config) {

	if !cfg.SSL {
		t.Errorf("SSL was false but it was expected to be true")
	}

	if cfg.Auth {
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

	if !cfg.Gzip {
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

	expectedLibraryScan := ScanSection{
		FilesPerOperation: 1500,
		SleepPerOperation: 15 * time.Millisecond,
		InitialWait:       1 * time.Second,
	}

	if cfg.LibraryScan != expectedLibraryScan {
		t.Errorf("LibraryScan was not as expected: It was: %#v, expected: %#v",
			cfg.LibraryScan, expectedLibraryScan)
	}
}

func TestMergingConfigs(t *testing.T) {
	cfg := new(Config)
	merged := new(MergedConfig)

	cfg.SSL = true

	cfg.merge(merged)

	if !cfg.SSL {
		t.Errorf("Zero value from the merged has been copied over")
	}

	str := ":http"
	merged = &MergedConfig{Listen: &str}

	cfg.merge(merged)

	if cfg.Listen != ":http" {
		t.Errorf("NonZero value has not been copied over")
	}

	cfg = getDefaultCfg()

	merged.Listen = new(string)
	*merged.Listen = ":8080"
	merged.SSL = new(bool)
	*merged.SSL = true
	merged.Libraries = new([]string)
	*merged.Libraries = append(*merged.Libraries, "/some/path")
	merged.Auth = new(bool)
	*merged.Auth = false
	merged.SSLCertificate = &Cert{Crt: "crt", Key: "key"}
	merged.LibraryScan = &ScanSection{
		FilesPerOperation: 1500,
		SleepPerOperation: 15 * time.Millisecond,
		InitialWait:       1 * time.Second,
	}

	cfg.merge(merged)
	checkMerge(t, cfg)
}

func TestMergingConfigsViaJSON(t *testing.T) {
	cfg := getDefaultCfg()
	testJSON := `
		{
			"listen": ":8080",
			"ssl": true,
			"ssl_certificate": {
				"key": "key",
				"crt": "crt"
			},
			"libraries": ["/some/path"],
			"library_scan": {
				"files_per_operation": 1500,
				"sleep_after_operation": "15ms",
				"initial_wait_duration": "1s"
			},
			"basic_authenticate": false
		}
	`
	if err := cfg.mergeJSON([]byte(testJSON)); err != nil {
		t.Errorf("Parsing test json failed: %s", err)
	}
	checkMerge(t, cfg)
}

func TestMergedConfigHasTheSameFieldsAsConfig(t *testing.T) {
	configType := reflect.TypeOf(Config{})
	mergedType := reflect.TypeOf(MergedConfig{})

	if configType.NumField() != mergedType.NumField() {
		t.Fatalf("Different number of fields in Config and MergedConfig")
	}

	for i := 0; i < configType.NumField(); i++ {
		configField := configType.Field(i)
		mergedField := mergedType.Field(i)

		if configField.Name != mergedField.Name {
			t.Errorf("Different field names: %s and %s", configField.Name,
				mergedField.Name)
		}

		if mergedField.Type.Kind() != reflect.Ptr {
			t.Errorf("MergedConfig field %s was not a pointer", mergedField.Name)
		}

		if configField.Tag != mergedField.Tag {
			t.Errorf("MergedConfig struct tag for %s was different", mergedField.Name)
		}
	}
}
