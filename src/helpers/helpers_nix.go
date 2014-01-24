// +build linux darwin bsd

/*
   Helpers for windows machines
*/

package helpers

import (
	"os"
	"path/filepath"
)

func ProjectUserPath() (string, error) {
	home := os.ExpandEnv("$HOME")
	if len(home) < 1 {
		home = "/opt"
	}

	path := filepath.Join(home, ".httpms")

	_, err := os.Stat(path)

	if err == nil {
		return path, nil
	}

	err = os.MkdirAll(path, os.ModeDir|0750)

	if err != nil {
		return "", err
	}

	return path, nil
}
