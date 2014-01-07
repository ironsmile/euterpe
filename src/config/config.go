// The module is resposible for findig, parsing and merging the HTTPMS configuration
// with the default. Configuration locations should be different depending on the
// host OS.
//
// Linux/BSD configurations should be in $HOME/.httpms/config.json
// Windows configurations should be in %APPDATA%/httpms/config.json
package config

// The configuration type. Should contain representation for everything in config.json
type Config struct {
	Listen         string     `json:"listen"`
	SSL            bool       `json:"ssl"`
	SSLCertificate ConfigCert `json:"ssl_certificate"`
	Authenticate   ConfigAuth `json:"authenticates"`
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
	return nil
}

// Returns the full path to the place where the user's configuration file should be
func (cfg *Config) UserPath() string {
	return ""
}

// Returns the full path to the default configuration file
func (cfg *Config) DefaultPath() string {
	return ""
}

// Returns true if the user configuration is present and in order. Otherwise false.
func (cfg *Config) UserConfigExists() bool {
	return false
}

// Will create (or replace if neccessery) the user configuration using the default
// config file supplied with the installation.
func (cfg *Config) CopyDefaultOverUser() error {
	return nil
}
