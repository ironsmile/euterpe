package radio_test

import (
	"context"
	"io/fs"
	"net/url"
	"os"
	"testing"

	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/radio"
)

// TestRadioManager uses the Stations interface to test operations with the manager.
func TestRadioManager(t *testing.T) {
	ctx := context.Background()

	lib := getLibrary(ctx, t)
	defer lib.Truncate()
	radios := radio.NewManager(lib.ExecuteDBJobAndWait)

	allRadios, err := radios.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all radios: %s", err)
	}

	if len(allRadios) != 0 {
		t.Errorf("Expected no radios in the DB but there were %d", len(allRadios))
	}

	playURL, _ := url.Parse("http://test-radio.example.com/play.mp3")
	homepageURL, _ := url.Parse("http://test-radio.example.com")

	expected := radio.Station{
		Name:      "Test Radio",
		StreamURL: *playURL,
		HomePage:  homepageURL,
	}

	newID, err := radios.Create(ctx, expected)
	if err != nil {
		t.Fatalf("Failed to create a radio: %s", err)
	}

	expected.ID = newID

	allRadios, err = radios.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed while getting all radios: %s", err)
	}
	if len(allRadios) != 1 {
		t.Fatalf("Expected one radio to be stored in the DB but got none")
	}

	compareStations(t, expected, allRadios[0])

	replacedURL, _ := url.Parse("http://replaced-radio.example.com/play.mp3")
	replacedHomepageURL, _ := url.Parse("http://replaced-radio.example.com")

	replaced := expected
	replaced.Name = "Replaced Test Radio"
	replaced.StreamURL = *replacedURL
	replaced.HomePage = replacedHomepageURL

	if err := radios.Replace(ctx, replaced); err != nil {
		t.Fatalf("Failed to replace a radio with error: %s", err)
	}

	allRadios, err = radios.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed while getting all radios: %s", err)
	}
	if len(allRadios) != 1 {
		t.Fatalf("Expected one radio to be stored in the DB but got none")
	}

	compareStations(t, replaced, allRadios[0])

	if err := radios.Delete(ctx, replaced.ID); err != nil {
		t.Fatalf("Failed to delete a radio with error: %s", err)
	}

	allRadios, err = radios.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed while getting all radios: %s", err)
	}
	if len(allRadios) != 0 {
		t.Fatalf("Expected no radios to be returned but there were %d", len(allRadios))
	}
}

func compareStations(t *testing.T, expected radio.Station, actual radio.Station) {
	if expected.ID != actual.ID {
		t.Errorf("Expected radio ID `%d` but got `%d`", expected.ID, actual.ID)
	}
	if expected.Name != actual.Name {
		t.Errorf("Expected radio name `%s` but got `%s`", expected.Name, actual.Name)
	}
	if expected.StreamURL.String() != actual.StreamURL.String() {
		t.Errorf("Expected radio stream URL `%s` but got `%s`",
			expected.StreamURL.String(),
			actual.StreamURL.String(),
		)
	}
	if expected.HomePage != nil && expected.HomePage.String() != actual.HomePage.String() {
		t.Errorf("Expected radio stream home page `%s` but got `%s`",
			expected.HomePage.String(),
			actual.HomePage.String(),
		)
	}
	if expected.HomePage == nil && actual.HomePage != nil {
		t.Errorf("Expected empty home page but got %s", actual.HomePage.String())
	}
}

// getTestMigrationFiles returns the SQLs directory used by the application itself
// normally. This way tests will be done with the exact same files which will be
// bundled into the binary on build.
func getTestMigrationFiles() fs.FS {
	return os.DirFS("../../sqls")
}

// It is the caller's responsibility to remove the library SQLite database file
func getLibrary(ctx context.Context, t *testing.T) *library.LocalLibrary {
	lib, err := library.NewLocalLibrary(ctx, library.SQLiteMemoryFile, getTestMigrationFiles())
	if err != nil {
		t.Fatal(err.Error())
	}

	err = lib.Initialize()
	if err != nil {
		t.Fatalf("Initializing library: %s", err)
	}

	return lib
}
