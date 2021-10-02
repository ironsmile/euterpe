// Package helpers contains few helpers functions which are used throughout the project.
package helpers

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"

	"github.com/spf13/afero"
)

// projectRoot is used to cache project root directory and save repeated
// calculations.
var projectRoot string

// ProjectRoot returns the source root directory. This function is useful only for
// tests which seek to find test files relative to the root of the repository.
func ProjectRoot() (rootPath string, err error) {

	if len(projectRoot) > 0 {
		return projectRoot, nil
	}

	defer func() {
		if err == nil && rootPath != "" {
			projectRoot = rootPath
		}
	}()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("no runtime.Caller information")
	}

	dirname := filepath.Dir(filename)
	return filepath.Clean(filepath.Join(dirname, "..", "..")), nil
}

// SetLogsFile sets the logfile of the server
func SetLogsFile(appfs afero.Fs, logFilePath string) error {
	logFile, err := appfs.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil && os.IsNotExist(err) {
		logFile, err = appfs.Create(logFilePath)
	}
	if err != nil {
		return err
	}
	log.SetOutput(logFile)
	return nil
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
// it if missing. User files are things such as sqlite files, logfiles and user configs.
func ProjectUserPath(appfs afero.Fs) (string, error) {
	user, err := user.Current()

	if err != nil {
		return "", err
	}

	deprecatedPath := filepath.Join(user.HomeDir, httpmsDir)
	if _, err = appfs.Stat(deprecatedPath); err == nil {
		return deprecatedPath, nil
	}

	path := filepath.Join(user.HomeDir, euterpeDir)

	if err = appfs.MkdirAll(path, os.ModeDir|0750); err != nil {
		return "", err
	}

	return path, nil
}

// SetUpPidFile will create the pidfile and it will contain the processid of the
// current process
func SetUpPidFile(appfs afero.Fs, PidFile string) error {
	fh, err := appfs.Create(PidFile)
	if err != nil {
		return err
	}

	if _, err = fh.WriteString(fmt.Sprintf("%d", os.Getpid())); err != nil {
		fh.Close()
		_ = appfs.Remove(PidFile)
		return err
	}

	fh.Close()
	return nil
}

// RemovePidFile just removes the pidFile. The argument should be file path.
func RemovePidFile(appfs afero.Fs, PidFile string) {
	_ = appfs.Remove(PidFile)
}

// GuessTrackNumber will use the file name of a particular media file to decide
// what its track number should be. This may be useful when the media file is
// missing this information in the meta data but the order is clearly marked in
// the file name. The function tries a few examples found by scanning real files
// found in the wild.
func GuessTrackNumber(trackFilePath string) int64 {
	basePath := filepath.Base(filepath.FromSlash(trackFilePath))

	if basePath == "." {
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
