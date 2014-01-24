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

// The configuration type. Should contain representation for everything in config.json
type Config struct {
	Listen         string     `json:"listen"`
	SSL            bool       `json:"ssl"`
	SSLCertificate ConfigCert `json:"ssl_certificate"`
	Auth           bool       `json:"basic_authenticate"`
	Authenticate   ConfigAuth `json:"authenticate"`
	Libraries      []string   `json:"libraries"`
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

	defaultConfig, err := ioutil.ReadFile(cfg.DefaultPath())

	if err != nil {
		return err
	}

	err = json.Unmarshal(defaultConfig, cfg)

	if err != nil {
		return err
	}

	usrCfg := new(Config)

	userConfig, err := ioutil.ReadFile(cfg.UserPath())

	if err != nil {
		return err
	}

	err = json.Unmarshal(userConfig, usrCfg)

	if err != nil {
		return err
	}

	cfgVal := reflect.ValueOf(cfg).Elem()
	userVal := reflect.ValueOf(usrCfg).Elem()

	for i := 0; i < userVal.NumField(); i++ {
		usrField := userVal.Field(i)
		if !usrField.IsValid() {
			continue
		}

		if reflect.Zero(reflect.TypeOf(usrField)) == usrField {
			continue
		}

		cfgField := cfgVal.Field(i)

		if !cfgField.CanSet() {
			continue
		}

		cfgField.Set(usrField)
	}
	return nil
}

// Returns the full path to the place where the user's configuration file should be
func (cfg *Config) UserPath() string {
	path, err := helpers.ProjectUserPath()
	if err != nil {
		log.Println(err)
		return ""
	}
	return filepath.Join(path, CONFIG_NAME)
}

// Returns the full path to the default configuration file
func (cfg *Config) DefaultPath() string {
	path, err := helpers.ProjectRoot()
	if err != nil {
		log.Println(err)
		return ""
	}
	return filepath.Join(path, CONFIG_NAME)
}

// Returns true if the user configuration is present and in order. Otherwise false.
func (cfg *Config) UserConfigExists() bool {
	path := cfg.UserPath()
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !st.IsDir()
}

// Will create (or replace if neccessery) the user configuration using the default
// config file supplied with the installation.
func (cfg *Config) CopyDefaultOverUser() error {
	userConfig := cfg.UserPath()
	defaultConfig := cfg.DefaultPath()
	return helpers.Copy(defaultConfig, userConfig)
}
