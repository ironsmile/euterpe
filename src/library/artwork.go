package library

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GetAlbumArtwork implements the ArtworkFinder interface for the local library.
// It would return a previously found artwork if any or try to find one in the
// filesystem or _on the internet_! This function returns ReadCloser and the caller
// is resposible for freeing the used resources by calling Close().
func (lib *LocalLibrary) GetAlbumArtwork(albumID int64) (io.ReadCloser, error) {
	albumPath, err := lib.GetAlbumFSPathByID(albumID)

	if err != nil {
		return nil, err
	}

	imagesRegexp := regexp.MustCompile(`(?i).*\.(png|gif|jpeg|jpg)$`)
	var possibleArtworks []string

	err = filepath.Walk(albumPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip directories
			return nil
		}
		if imagesRegexp.MatchString(path) {
			possibleArtworks = append(possibleArtworks, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(possibleArtworks) < 1 {
		return nil, ErrArtworkNotFound
	}

	var (
		selectedArtwork string
		score           int
	)

	for _, path := range possibleArtworks {
		pathScore := 5

		fileBase := strings.ToLower(filepath.Base(path))

		if strings.HasPrefix(fileBase, "cover.") || strings.HasPrefix(fileBase, "front.") {
			pathScore = 15
		}

		if strings.Contains(fileBase, "cover") || strings.Contains(fileBase, "front") {
			pathScore = 10
		}

		if strings.Contains(fileBase, "artwork") {
			pathScore = 8
		}

		if pathScore > score {
			selectedArtwork = path
			score = pathScore
		}
	}

	return os.Open(selectedArtwork)
}
