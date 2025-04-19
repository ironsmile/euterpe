package config_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ironsmile/euterpe/src/config"
	"github.com/spf13/afero"
)

// TestScanSectionUnmarshalJSON makes sure that decoding the JSON for the "library_scan"
// configuration key is working as expected.
func TestScanSectionUnmarshalJSON(t *testing.T) {
	jsonBuff := bytes.NewBufferString(`
		{
			"disable": false,
			"files_per_operation": 100,
			"sleep_after_operation": "15ms",
			"initial_wait_duration": "100ms"
		}
	`)

	var ss config.ScanSection
	dec := json.NewDecoder(jsonBuff)
	err := dec.Decode(&ss)
	if err != nil {
		t.Fatalf("decoding ScanSection JSON failed: %s", err)
	}

	expected := config.ScanSection{
		Disable:           false,
		FilesPerOperation: 100,
		SleepPerOperation: 15 * time.Millisecond,
		InitialWait:       100 * time.Millisecond,
	}

	if ss != expected {
		t.Errorf("expected `%+v` but got `%+v`", expected, ss)
	}
}

// TestScanSectionErrors checks that various errors during parsing the "scan" section
// of the configuration return appropriate errors.
func TestScanSectionErrors(t *testing.T) {
	tests := []struct {
		desc        string
		cfgString   string
		syntaxError bool
		errContains string
	}{
		{
			desc:        "bad JSON syntax",
			cfgString:   "wow totally not a JSON",
			syntaxError: true,
		},
		{
			desc: "negative files per operation",
			cfgString: `{
							"disable": false,
							"files_per_operation": -20,
							"sleep_after_operation": "15ms",
							"initial_wait_duration": "100ms"
						}`,
			errContains: "files_per_operation",
		},
		{
			desc: "invalid initial wait",
			cfgString: `{
							"disable": false,
							"files_per_operation": 100,
							"sleep_after_operation": "15ms",
							"initial_wait_duration": "baba"
						}`,
			errContains: "initial_wait_duration",
		},
		{
			desc: "invalid sleep per operation",
			cfgString: `{
							"disable": false,
							"files_per_operation": 100,
							"sleep_after_operation": "baba",
							"initial_wait_duration": "15ms"
						}`,
			errContains: "sleep_after_operation",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var ss config.ScanSection
			dec := json.NewDecoder(bytes.NewBufferString(test.cfgString))
			err := dec.Decode(&ss)
			if err == nil {
				t.Fatal("expected an error but got none")
			}

			if test.syntaxError {
				_, ok := err.(*json.SyntaxError)
				if !ok {
					t.Errorf("expected syntax error but got `%s`", err)
				}
			}

			if test.errContains != "" && !strings.Contains(err.Error(), test.errContains) {
				t.Logf("Received error: %s\n", err)
				t.Errorf("expected error to contain '%s' but it did not", test.errContains)
			}
		})
	}
}

// TestFindAndParseCreatesConfig makes sure that a new configuration file is created
// when there was not when run.
func TestFindAndParseCreatesConfig(t *testing.T) {
	testfs := afero.NewMemMapFs()

	_, err := config.FindAndParse(testfs)
	if err != nil {
		t.Fatalf("error finding and parsing configuration file: %s", err)
	}

	configPath := config.UserConfigPath(testfs)
	st, err := testfs.Stat(configPath)
	if err != nil {
		t.Fatalf("error on stat for config file `%s`: %s", configPath, err)
	}

	if !st.Mode().IsRegular() {
		t.Errorf("expected the configuration file to be regular file")
	}
}

// TestFindAndParse makes sure that a file which is already created is read
// and parsed.
func TestFindAndParse(t *testing.T) {
	testfs := afero.NewMemMapFs()

	configPath := config.UserConfigPath(testfs)

	const (
		listenAddress = "1.2.3.4:1234"
		user          = "test-user"
		pass          = "test-pass"
		secret        = "test-secret"
	)

	// Function is used for easier clean-up with defer.
	func() {
		fh, err := testfs.Create(configPath)
		if err != nil {
			t.Fatalf("error setting up test, config file create: %s", err)
		}
		defer fh.Close()

		fmt.Fprintf(fh, `{
			"listen": "%s",
			"basic_authenticate": true,
			"authentication": {
				"user": "%s",
				"password": "%s",
				"secret": "%s"
			}
		}`, listenAddress, user, pass, secret)
	}()

	cfg, err := config.FindAndParse(testfs)
	if err != nil {
		t.Fatalf("error finding and parsing configuration file: %s", err)
	}

	if cfg.Listen != listenAddress {
		t.Errorf("expected listen address `%s` but got `%s`", listenAddress, cfg.Listen)
	}

	if !cfg.Auth {
		t.Error("expected basic authenticate to be True but it was not")
	}

	if cfg.Authenticate.User != user {
		t.Errorf("expected username `%s` but got `%s`", cfg.Authenticate.User, user)
	}

	if cfg.Authenticate.Password != pass {
		t.Errorf("expected password `%s` but got `%s`", cfg.Authenticate.Password, pass)
	}

	if cfg.Authenticate.Secret != secret {
		t.Errorf("expected secret `%s` but got `%s`", cfg.Authenticate.Secret, secret)
	}
}
