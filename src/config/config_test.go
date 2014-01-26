package config

import (
	"path/filepath"
	"reflect"
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
	merged := new(MergedConfig)

	cfg.SSL = true

	cfg.merge(merged)

	if cfg.SSL != true {
		t.Errorf("Zero value from the merged has been copied over")
	}

	str := ":http"
	merged = &MergedConfig{Listen: &str}

	cfg.merge(merged)

	if cfg.Listen != ":http" {
		t.Errorf("NonZero value has not been copied over")
	}

	getDefaultCfg := func() *Config {
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
		dflConfig.Authenticate = ConfigAuth{User: "bob", Password: "marley"}
		return dflConfig
	}

	cfg = getDefaultCfg()

	merged.Listen = new(string)
	*merged.Listen = ":8080"
	merged.SSL = new(bool)
	*merged.SSL = true
	merged.SSLCertificate = new(ConfigCert)
	*merged.SSLCertificate = ConfigCert{Crt: "crt", Key: "key"}
	merged.Libraries = new([]string)
	*merged.Libraries = append(*merged.Libraries, "/some/path")
	merged.Auth = new(bool)
	*merged.Auth = false

	cfg.merge(merged)

	checkMerge := func(cfg *Config) {

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

	checkMerge(cfg)

	cfg = getDefaultCfg()
	cfg.mergeJSON([]byte(`
		{
			"listen": ":8080",
			"ssl": true,
			"ssl_certificate": {
				"key": "key",
				"crt": "crt"
			},
			"libraries": ["/some/path"],
			"basic_authenticate": false
		}
	`))
	checkMerge(cfg)
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
