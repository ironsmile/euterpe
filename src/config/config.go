// The module is resposible for findig, parsing and merging the HTTPMS configuration
// with the default. Configuration locations should be different depending on the
// host OS.
//
// Linux/BSD configurations should be in $HOME/.httpms/config.json
// Windows configurations should be in %APPDATA%/httpms/config.json
package config

// The configuration type. Should contain representation for everything in config.json
type Config struct {
}

// Actually finds the configuration file, parsing it and merging in on top the default
// configuration.
func (cfg *Config) FindAndParse() error {
	return nil
}
