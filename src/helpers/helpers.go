// Contains few helpers functions which are used througout the project
package helpers

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Returns the root directory. This is the place where the app is installed
// or the place where the source is stored if in development or installed
// with go get
func ProjectRoot() (string, error) {

	// first trying the gopath
	gopath := os.ExpandEnv("$GOPATH")
	relPath := filepath.FromSlash("src/github.com/ironsmile/httpms")
	for _, path := range strings.Split(gopath, ":") {
		tmplPath := filepath.Join(path, relPath)
		entry, err := os.Stat(tmplPath)
		if err != nil {
			continue
		}

		if entry.IsDir() {
			return tmplPath, nil
		}
	}

	// now we try the directory of the binary
	if len(os.Args) < 1 {
		// highly unlikely but still!
		return "", errors.New("os.Args is empty. " +
			"Cannot find the project root directory.")
	}

	abs, err := filepath.Abs(os.Args[0])

	if err != nil {
		return "", err
	}

	return abs, nil
}

// Sets the logfile of the server
func SetLogsFile(logFilePath string) {
	logFile, err := os.Create(logFilePath)
	if err != nil {
		log.Println("Could not open logfile")
		os.Exit(1)
	}
	log.SetOutput(logFile)
}
