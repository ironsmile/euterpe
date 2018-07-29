// Package config is resposible for finding, parsing and merging the HTTPMS user
// configuration  with the default. Configuration locations should be different
// depending on the host OS.
//
// Linux/BSD configurations should be in $HOME/.httpms/config.json
// Windows configurations should be in %APPDATA%/httpms/config.json
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/ironsmile/httpms/src/helpers"
)

const (
	// ConfigName contains the name of the actual configuration file. This is the one
	// the user is supposed to change.
	ConfigName = "config.json"

	defaultlistAddress = ":9996"
)

// defaultConfig contains all the default values for the HTTPMS configuration. Users
// can overwrite values here with their user's configuraiton.
var defaultConfig = Config{
	Listen:         defaultlistAddress,
	LogFile:        "httpms.log",
	SqliteDatabase: "httpms.db",
	Gzip:           true,
	ReadTimeout:    15,
	WriteTimeout:   1200,
	MaxHeadersSize: 1048576,
}

// Config contains representation for everything in config.json
type Config struct {
	Listen          string      `json:"listen,omitempty"`
	SSL             bool        `json:"ssl,omitempty"`
	SSLCertificate  Cert        `json:"ssl_certificate,omitempty"`
	Auth            bool        `json:"basic_authenticate,omitempty"`
	Authenticate    Auth        `json:"authentication,omitempty"`
	Libraries       []string    `json:"libraries,omitempty"`
	LibraryScan     ScanSection `json:"library_scan,omitempty"`
	UserPath        string      `json:"user_path,omitempty"`
	LogFile         string      `json:"log_file,omitempty"`
	SqliteDatabase  string      `json:"sqlite_database,omitempty"`
	Gzip            bool        `json:"gzip,omitempty"`
	ReadTimeout     int         `json:"read_timeout,omitempty"`
	WriteTimeout    int         `json:"write_timeout,omitempty"`
	MaxHeadersSize  int         `json:"max_header_bytes,omitempty"`
	DownloadArtwork bool        `json:"download_artwork,omitempty"`
}

// ScanSection is used for merging the two configs. Its purpose is to essentially
// hold the default values for its properties.
type ScanSection struct {
	FilesPerOperation int64         `json:"files_per_operation,omitempty"`
	SleepPerOperation time.Duration `json:"sleep_after_operation,omitempty"`
	InitialWait       time.Duration `json:"initial_wait_duration,omitempty"`
}

// Cert represents a configuration for TLS certificate
type Cert struct {
	Crt string `json:"crt,omitempty"`
	Key string `json:"key,omitempty"`
}

// Auth represents a configuration HTTP Basic authentication
type Auth struct {
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

// FindAndParse actually finds the configuration file, parsing it and merging it on
// top the default configuration.
func FindAndParse() (Config, error) {
	if !UserConfigExists() {
		err := CopyDefaultOverUser()
		if err != nil {
			return Config{}, err
		}
	}

	cfg := defaultConfig
	userCfgPath := UserConfigPath()

	fh, err := os.Open(userCfgPath)
	if err != nil {
		return Config{}, fmt.Errorf("opening config: %s", err)
	}
	defer fh.Close()

	dec := json.NewDecoder(fh)

	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decoding config: %s", err)
	}

	return cfg, nil
}

// UserConfigPath returns the full path to the place where the user's configuration
// file should be
func UserConfigPath() string {
	path, err := helpers.ProjectUserPath()
	if err != nil {
		log.Println(err)
		return ""
	}
	return filepath.Join(path, ConfigName)
}

// UserConfigExists returns true if the user configuration is present and in order.
// Otherwise false.
func UserConfigExists() bool {
	path := UserConfigPath()
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !st.IsDir()
}

// CopyDefaultOverUser will create (or replace if neccessery) the user configuration
// using the default config new config.
func CopyDefaultOverUser() error {
	var homeDir = "~"
	user, err := user.Current()
	if err == nil {
		homeDir = user.HomeDir
	}

	userCfg := Config{
		Listen: defaultlistAddress,
		Libraries: []string{
			filepath.Join(homeDir, "Music"),
		},
	}

	userCfgPath := UserConfigPath()
	fh, err := os.Create(userCfgPath)
	if err != nil {
		return fmt.Errorf("create config `%s`: %s", userCfgPath, err)
	}
	defer fh.Close()

	enc := json.NewEncoder(fh)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&userCfg); err != nil {
		return fmt.Errorf("encoding default config `%s`: %s", userCfgPath, err)
	}

	return nil
}
