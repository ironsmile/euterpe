// The module is resposible for finding, parsing and merging the HTTPMS user
// configuration  with the default. Configuration locations should be different
// depending on the host OS.
//
// Linux/BSD configurations should be in $HOME/.httpms/config.json
// Windows configurations should be in %APPDATA%/httpms/config.json
package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/ironsmile/httpms/src/helpers"
)

const CONFIG_NAME = "config.json"
const DEFAULT_CONFIG_NAME = "config.default.json"

// The configuration type. Should contain representation for everything in config.json
type Config struct {
	Listen         string     `json:"listen"`
	SSL            bool       `json:"ssl"`
	SSLCertificate ConfigCert `json:"ssl_certificate"`
	Auth           bool       `json:"basic_authenticate"`
	Authenticate   ConfigAuth `json:"authentication"`
	Libraries      []string   `json:"libraries"`
	UserPath       string     `json:"user_path"`
	LogFile        string     `json:"log_file"`
	SqliteDatabase string     `json:"sqlite_database"`
	Gzip           bool       `json:"gzip"`
	ReadTimeout    int        `json:"read_timeout"`
	WriteTimeout   int        `json:"write_timeout"`
	MaxHeadersSize int        `json:"max_header_bytes"`
	HTTPRoot       string     `json:"http_root"`
}

type MergedConfig struct {
	Listen         *string     `json:"listen"`
	SSL            *bool       `json:"ssl"`
	SSLCertificate *ConfigCert `json:"ssl_certificate"`
	Auth           *bool       `json:"basic_authenticate"`
	Authenticate   *ConfigAuth `json:"authentication"`
	Libraries      []string    `json:"libraries"`
	UserPath       *string     `json:"user_path"`
	LogFile        *string     `json:"log_file"`
	SqliteDatabase *string     `json:"sqlite_database"`
	Gzip           *bool       `json:"gzip"`
	ReadTimeout    *int        `json:"read_timeout"`
	WriteTimeout   *int        `json:"write_timeout"`
	MaxHeadersSize *int        `json:"max_header_bytes"`
	HTTPRoot       *string     `json:"http_root"`
}

type ConfigCert struct {
	Crt string `json:"crt"`
	Key string `json:"key"`
}

type ConfigAuth struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// Actually finds the configuration file, parsing it and merging it on top the default
// configuration.
func (cfg *Config) FindAndParse() error {
	if !cfg.UserConfigExists() {
		err := cfg.CopyDefaultOverUser()
		if err != nil {
			return err
		}
	}

	err := cfg.parse(cfg.DefaultConfigPath())

	if err != nil {
		return err
	}

	usrCfg := new(Config)

	err = usrCfg.parse(cfg.UserConfigPath())

	if err != nil {
		return err
	}

	cfg.merge(usrCfg)

	return nil
}

// The config object parses an json file and populates its fields.
// The json file is specified by the finame argument.
func (cfg *Config) parse(filename string) error {
	defaultConfig, err := ioutil.ReadFile(filename)

	if err != nil {
		return err
	}

	return json.Unmarshal(defaultConfig, cfg)
}

// Merges an other config on top of itself. Only non-zero values will be merged.
func (cfg *Config) merge(merged *Config) {
	cfgVal := reflect.ValueOf(cfg).Elem()
	mergedVal := reflect.ValueOf(merged).Elem()

	for i := 0; i < mergedVal.NumField(); i++ {
		mergedField := mergedVal.Field(i)
		if !mergedField.IsValid() {
			continue
		}

		if reflect.Zero(mergedField.Type()) == mergedField {
			continue
		}

		cfgField := cfgVal.Field(i)

		if !cfgField.CanSet() {
			continue
		}

		//log.Printf("Merging %#v\n", mergedField)
		cfgField.Set(mergedField)
	}
}

// Returns the full path to the place where the user's configuration file should be
func (cfg *Config) UserConfigPath() string {
	if len(cfg.UserPath) > 0 {
		if filepath.IsAbs(cfg.UserPath) {
			return filepath.Join(cfg.UserPath, CONFIG_NAME)
		} else {
			log.Printf("User path %s was invalid as it was not rooted", cfg.UserPath)
		}
	}
	path, err := helpers.ProjectUserPath()
	if err != nil {
		log.Println(err)
		return ""
	}
	return filepath.Join(path, CONFIG_NAME)
}

// Returns the full path to the default configuration file
func (cfg *Config) DefaultConfigPath() string {
	path, err := helpers.ProjectRoot()
	if err != nil {
		log.Println(err)
		return ""
	}
	return filepath.Join(path, DEFAULT_CONFIG_NAME)
}

// Returns true if the user configuration is present and in order. Otherwise false.
func (cfg *Config) UserConfigExists() bool {
	path := cfg.UserConfigPath()
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !st.IsDir()
}

// Will create (or replace if neccessery) the user configuration using the default
// config file supplied with the installation.
func (cfg *Config) CopyDefaultOverUser() error {
	userConfig := cfg.UserConfigPath()
	defaultConfig := cfg.DefaultConfigPath()
	return helpers.Copy(defaultConfig, userConfig)
}
