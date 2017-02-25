// Package helpers contains few helpers functions which are used throughout the project
package helpers

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var projectRoot string

func cacheProjRoot(path string) string {
	projectRoot = path
	return path
}

// ProjectRoot returns the root directory. This is the place where the app is installed
// or the place where the source is stored if in development or installed with go get
func ProjectRoot() (string, error) {

	if len(projectRoot) > 0 {
		return projectRoot, nil
	}

	// first trying the gopath
	gopath := os.ExpandEnv("$GOPATH")
	relPath := filepath.FromSlash("src/github.com/ironsmile/httpms")
	for _, path := range strings.Split(gopath, ":") {
		rootPath := filepath.Join(path, relPath)
		entry, err := os.Stat(rootPath)
		if err != nil {
			continue
		}

		if entry.IsDir() {
			return cacheProjRoot(rootPath), nil
		}
	}

	etcPath := filepath.FromSlash("/etc/httpms")

	st, err := os.Stat(etcPath)

	if err == nil && st.IsDir() {
		return cacheProjRoot(etcPath), nil
	}

	// now we try the directory of the binary
	if len(os.Args) < 1 {
		// highly unlikely but still!
		return "", errors.New("os.Args is empty. " +
			"Cannot find the project root directory.")
	}

	noSymlinkPath, err := filepath.EvalSymlinks(os.Args[0])

	if err != nil {
		return "", err
	}

	abs, err := filepath.Abs(filepath.Dir(noSymlinkPath))

	if err != nil {
		return "", err
	}

	return cacheProjRoot(abs), nil
}

// SetLogsFile sets the logfile of the server
func SetLogsFile(logFilePath string) error {
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return err
	}
	log.SetOutput(logFile)
	return nil
}

// Copy copies a file from src to dst
func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

// AbsolutePath returns absolute path. If path is already absolute leave it be. If not
// join it with relativeRoot
func AbsolutePath(path, relativeRoot string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(relativeRoot, path)
}

// ProjectUserPath returns the directory in which user files should be stored. Creates
// it is missing. User files are thing such as sqlite files, logfiles and user configs.
func ProjectUserPath() (string, error) {
	user, err := user.Current()

	if err != nil {
		return "", err
	}

	path := filepath.Join(user.HomeDir, HttpmsDir)

	_, err = os.Stat(path)

	if err == nil {
		return path, nil
	}

	err = os.MkdirAll(path, os.ModeDir|0750)

	if err != nil {
		return "", err
	}

	return path, nil
}

// SetUpPidFile will create the pidfile and it will contain the processid of the
// current process
func SetUpPidFile(PidFile string) {
	fh, err := os.Create(PidFile)

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	_, err = fh.WriteString(fmt.Sprintf("%d", os.Getpid()))

	if err != nil {
		log.Println(err)
		fh.Close()
		_ = os.Remove(PidFile)
		os.Exit(1)
	}

	fh.Close()
}

// RemovePidFile just removes the pidFile. The argument should be file path.
func RemovePidFile(PidFile string) {
	_ = os.Remove(PidFile)
}

// GuessTrackNumber will use the file name of a particular media file to decide
// what its track number should be. This may be useful when the media file is
// missing this information in the meta data but the order is clearly marked in
// the file name. The function tries a few examples found by scanning real files
// found in the wild.
func GuessTrackNumber(trackFilePath string) int64 {
	basePath := filepath.Base(filepath.FromSlash(trackFilePath))

	if basePath == "" {
		// fast path, no need to match any rules
		return 0
	}

	matchRules := []string{
		// First, high confidence guesses. This would match file names which start with
		// the track number, followed by some punctuation.
		`^(\d+)[ \-\t\.\)\]].+`,

		// Now some lower confidence stuff. Maybe the track number is in the middle of
		// the file name. Searching for stuff like
		//
		//		Iron Maiden - 7 - Quest For Fire.mp3
		//		nightwish -10- Beauty Of The Beast.mp3
		//
		// and any variation of this.
		`.+- (\d+) -.+`,
		`.+ -(\d+)- .+`,

		// Example: [Iron Maiden] - 06__Wasting love.mp3
		`.+ - (\d+)_.+`,

		// Example: METALLICA - (04) One.mp3
		`.+ - \((\d+)\) .+`,

		// Example: Fatboy Slim - [14] Brimful Of Asha (Cornershop).mp3
		`.+ - \[(\d+)\] .+`,

		// Example: Nightwish-07-Ocean_Soul.mp3
		`[\w]+-(\d+)-[\w]+`,

		// Example: #11_12_Chelovek na Lune.mp3
		`^#\d+_(\d+)_.+`,
	}

	for _, rule := range matchRules {
		matcher := regexp.MustCompile(rule)

		if matched := matcher.FindStringSubmatch(basePath); matched != nil {
			return stringToInt64OrZero(matched[1])
		}
	}

	return 0
}

func stringToInt64OrZero(str string) int64 {
	num, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0
	}
	return num
}
