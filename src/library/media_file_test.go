package library

import (
	"path/filepath"
	"testing"

	"github.com/ironsmile/euterpe/src/assert"
	"github.com/ironsmile/euterpe/src/helpers"
)

// TestTagParsingFallback makes sure that parsing OGG files with Vorbis tags falls
// back to the dhwden/tag library.
func TestTagParsingFallback(t *testing.T) {
	projRoot, err := helpers.ProjectRoot()
	if err != nil {
		t.Fatalf("was not able to find test_files directory: %s", err)
	}

	oggFile := filepath.Join(projRoot, "test_files", "ogg_files", "vorbis-tags.ogg")
	media, err := parseFileTags(oggFile)
	if err != nil {
		t.Fatalf("failed to parse OGG file tags: %s", err)
	}

	assert.Equal(t, "Test Artist", media.Artist(), "wrong artist name")
	assert.Equal(t, "Vorbis Album Title", media.Album(), "wrong album name")
	assert.Equal(t, "Some Track", media.Title(), "wrong track title")
	assert.Equal(t, 1, media.Track(), "wrong track number")
	assert.Equal(t, 0, media.Length(), "wrong track duration")
	assert.Equal(t, 2025, media.Year(), "wrong track year")
}
