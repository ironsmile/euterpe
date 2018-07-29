// Package config is resposible for finding, parsing and merging the HTTPMS user
// configuration  with the default. Configuration locations should be different
// depending on the host OS.
//
// Linux/BSD configurations should be in $HOME/.httpms/config.json
// Windows configurations should be in %APPDATA%/httpms/config.json
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/ironsmile/httpms/src/helpers"
)

// ConfigName contains the name of the actual configuration file. This is the one
// the user is supposed to change.
const ConfigName = "config.json"

// DefaultConfigName contains the name of the file which contains the default
// configuration values for HTTPMS. It shouldn't be edited by a user and it is part
// of the app installation. Preferably, it should be hidden away.
const DefaultConfigName = "config.default.json"

// Config contains representation for everything in config.json
type Config struct {
	Listen          string      `json:"listen"`
	SSL             bool        `json:"ssl"`
	SSLCertificate  Cert        `json:"ssl_certificate"`
	Auth            bool        `json:"basic_authenticate"`
	Authenticate    Auth        `json:"authentication"`
	Libraries       []string    `json:"libraries"`
	LibraryScan     ScanSection `json:"library_scan"`
	UserPath        string      `json:"user_path"`
	LogFile         string      `json:"log_file"`
	SqliteDatabase  string      `json:"sqlite_database"`
	Gzip            bool        `json:"gzip"`
	ReadTimeout     int         `json:"read_timeout"`
	WriteTimeout    int         `json:"write_timeout"`
	MaxHeadersSize  int         `json:"max_header_bytes"`
	DownloadArtwork bool        `json:"download_artwork"`
}

// MergedConfig is used for merging one config over the other. I need the zero value
// for every Field to be nil so that I can destinguish if it has been in the json file
// or not. If I did not use pointers I would not have been able to do that.
// That way the merged (user) json can contain a subset of all fields and everything
// else will be used from the default json.
// Unfortunately this leads to repetition since MergedConfig must have the same
// fields in the same order as Config.
type MergedConfig struct {
	Listen          *string      `json:"listen"`
	SSL             *bool        `json:"ssl"`
	SSLCertificate  *Cert        `json:"ssl_certificate"`
	Auth            *bool        `json:"basic_authenticate"`
	Authenticate    *Auth        `json:"authentication"`
	Libraries       *[]string    `json:"libraries"`
	LibraryScan     *ScanSection `json:"library_scan"`
	UserPath        *string      `json:"user_path"`
	LogFile         *string      `json:"log_file"`
	SqliteDatabase  *string      `json:"sqlite_database"`
	Gzip            *bool        `json:"gzip"`
	ReadTimeout     *int         `json:"read_timeout"`
	WriteTimeout    *int         `json:"write_timeout"`
	MaxHeadersSize  *int         `json:"max_header_bytes"`
	DownloadArtwork *bool        `json:"download_artwork"`
}

// ScanSection is used for merging the two configs. Its purpose is to essentially
// hold the default values for its properties.
type ScanSection struct {
	FilesPerOperation int64         `json:"files_per_operation"`
	SleepPerOperation time.Duration `json:"sleep_after_operation"`
	InitialWait       time.Duration `json:"initial_wait_duration"`
}

// UnmarshalJSON parses a JSON and populets its ScanSection. Satisfies the
// Unmrashaller interface.
func (ss *ScanSection) UnmarshalJSON(input []byte) error {
	ssProxy := &struct {
		FilesPerOperation int64  `json:"files_per_operation"`
		SleepPerOperation string `json:"sleep_after_operation"`
		InitialWait       string `json:"initial_wait_duration"`
	}{}

	if err := json.Unmarshal(input, ssProxy); err != nil {
		return err
	}

	ss.FilesPerOperation = ssProxy.FilesPerOperation

	if ssProxy.SleepPerOperation != "" {
		spo, err := time.ParseDuration(ssProxy.SleepPerOperation)
		if err != nil {
			return err
		}
		ss.SleepPerOperation = spo
	}

	if ssProxy.InitialWait != "" {
		iwd, err := time.ParseDuration(ssProxy.InitialWait)
		if err != nil {
			return err
		}
		ss.InitialWait = iwd
	}

	if ss.FilesPerOperation <= 0 {
		return errors.New("files_per_operation must be a positive integer")
	}

	return nil
}

// Cert represents a configuration for TLS certificate
type Cert struct {
	Crt string `json:"crt"`
	Key string `json:"key"`
}

// Auth represents a configuration HTTP Basic authentication
type Auth struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// FindAndParse actually finds the configuration file, parsing it and merging it on
// top the default configuration.
func (cfg *Config) FindAndParse() error {
	if !cfg.UserConfigExists() {
		err := cfg.CopyDefaultOverUser()
		if err != nil {
			return err
		}
	}

	defaultPath := cfg.DefaultConfigPath()
	err := cfg.parse(defaultPath)

	if err != nil {
		return fmt.Errorf("Parsing %s failed: %s", defaultPath, err.Error())
	}

	userPath := cfg.UserConfigPath()
	defaultConfig, err := ioutil.ReadFile(userPath)

	if err != nil {
		return fmt.Errorf("Parsing %s failed: %s", userPath, err.Error())
	}

	return cfg.mergeJSON(defaultConfig)
}

// The config object parses an json file and populates its fields.
// The json file is specified by the finame argument.
func (cfg *Config) parse(filename string) error {
	jsonContents, err := ioutil.ReadFile(filename)

	if err != nil {
		return err
	}

	return json.Unmarshal(jsonContents, cfg)
}

// Parses the json buffer jsonBuffer into a MergedConfig and uses it
// for cfg.merge
func (cfg *Config) mergeJSON(jsonBuffer []byte) error {

	usrCfg := new(MergedConfig)

	err := json.Unmarshal(jsonBuffer, usrCfg)

	if err != nil {
		return err
	}

	cfg.merge(usrCfg)
	return nil
}

// Merges an MergedConfig on top of itself. Only non-zero values will be merged.
func (cfg *Config) merge(merged *MergedConfig) {
	cfgVal := reflect.ValueOf(cfg).Elem()
	mergedVal := reflect.ValueOf(merged).Elem()

	for i := 0; i < mergedVal.NumField(); i++ {
		mergedField := mergedVal.Field(i)
		if !mergedField.IsValid() || mergedField.IsNil() {
			continue
		}
		cfgField := cfgVal.Field(i)
		if !cfgField.CanSet() {
			continue
		}
		if mergedField.Kind() != reflect.Ptr {
			cfgField.Set(mergedField)
			continue
		}
		cfgField.Set(reflect.Indirect(mergedField))
	}
}

// UserConfigPath returns the full path to the place where the user's configuration
// file should be
func (cfg *Config) UserConfigPath() string {
	if len(cfg.UserPath) > 0 {
		if filepath.IsAbs(cfg.UserPath) {
			return filepath.Join(cfg.UserPath, ConfigName)
		}
		log.Printf("User path %s was invalid as it was not rooted", cfg.UserPath)
	}
	path, err := helpers.ProjectUserPath()
	if err != nil {
		log.Println(err)
		return ""
	}
	return filepath.Join(path, ConfigName)
}

// DefaultConfigPath returns the full path to the default configuration file
func (cfg *Config) DefaultConfigPath() string {
	path, err := helpers.ProjectRoot()
	if err != nil {
		log.Println(err)
		return ""
	}
	return filepath.Join(path, DefaultConfigName)
}

// UserConfigExists returns true if the user configuration is present and in order.
// Otherwise false.
func (cfg *Config) UserConfigExists() bool {
	path := cfg.UserConfigPath()
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !st.IsDir()
}

// CopyDefaultOverUser will create (or replace if neccessery) the user configuration
// using the default config file supplied with the installation.
func (cfg *Config) CopyDefaultOverUser() error {
	userConfig := cfg.UserConfigPath()
	defaultConfigDir := filepath.Dir(cfg.DefaultConfigPath())
	defaultConfig := filepath.Join(defaultConfigDir, ConfigName)
	return helpers.Copy(defaultConfig, userConfig)
}
