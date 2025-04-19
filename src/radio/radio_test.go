package radio_test

import (
	"context"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/radio"
)

// TestRadioManager uses the Stations interface to test operations with the manager.
func TestRadioManager(t *testing.T) {
	ctx := context.Background()

	lib := getLibrary(ctx, t)
	defer func() {
		_ = lib.Truncate()
	}()
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

// TestRadioManagerErrors checks that various errors are returned on invalid input.
func TestRadioManagerErrors(t *testing.T) {
	ctx := context.Background()

	lib := getLibrary(ctx, t)
	defer func() {
		_ = lib.Truncate()
	}()
	radios := radio.NewManager(lib.ExecuteDBJobAndWait)

	playURL, _ := url.Parse("http://test-radio.example.com/play.mp3")
	homepageURL, _ := url.Parse("http://test-radio.example.com")

	goodStation := radio.Station{
		Name:      "Test Radio",
		StreamURL: *playURL,
		HomePage:  homepageURL,
	}

	noName := goodStation
	noName.Name = ""

	if _, err := radios.Create(ctx, noName); err == nil {
		t.Errorf("Expected error for radio without name but got none")
	} else if !strings.Contains(err.Error(), "name") {
		t.Logf("Error returned: %s\n", err)
		t.Errorf("Expected the error to mention that no name was set")
	}

	ftpURL, _ := url.Parse("ftp://test-radio.example.com")

	badPlaySchema := goodStation
	badPlaySchema.StreamURL = *ftpURL
	if _, err := radios.Create(ctx, badPlaySchema); err == nil {
		t.Errorf("Expected error for radio FTP play URL but got none")
	} else if !strings.Contains(err.Error(), "scheme") {
		t.Logf("Error returned for FTP stream URL: %s\n", err)
		t.Errorf("Expected the error to mention that URL scheme is not supported")
	}

	badHomeSchema := goodStation
	badHomeSchema.HomePage = ftpURL
	if _, err := radios.Create(ctx, badHomeSchema); err == nil {
		t.Errorf("Expected error for radio FTP home URL but got none")
	} else if !strings.Contains(err.Error(), "scheme") {
		t.Logf("Error returned for FTP home: %s\n", err)
		t.Errorf("Expected the error to mention that URL scheme is not supported")
	}

	if err := radios.Delete(ctx, 5823); !errors.Is(err, radio.ErrNotFound) {
		t.Errorf("Expected 'not found' error but got %s", err)
	}

	stationID, err := radios.Create(ctx, goodStation)
	if err != nil {
		t.Fatalf("Cannot create a station for testing: %s", err)
	}

	goodStation.ID = stationID

	replaced := goodStation
	replaced.Name = "Replaced Station"
	replaced.ID = goodStation.ID + 1

	if err := radios.Replace(ctx, replaced); !errors.Is(err, radio.ErrNotFound) {
		t.Errorf("Expected 'not found' error but got %s", err)
	}

	replaced = goodStation
	replaced.Name = ""
	if err := radios.Replace(ctx, replaced); err == nil {
		t.Errorf("Expected error for replaced radio without name but got none")
	} else if !strings.Contains(err.Error(), "name") {
		t.Logf("Error returned: %s\n", err)
		t.Errorf("Expected the error for replaced radio to mention that no name was set")
	}

	replaced = goodStation
	replaced.StreamURL = *ftpURL
	if err := radios.Replace(ctx, replaced); err == nil {
		t.Errorf("Expected error for replaced radio FTP play URL but got none")
	} else if !strings.Contains(err.Error(), "scheme") {
		t.Logf("Error returned for FTP stream URL: %s\n", err)
		t.Errorf("Expected the error to mention that URL scheme is not supported")
	}

	replaced = goodStation
	replaced.HomePage = ftpURL
	if err := radios.Replace(ctx, replaced); err == nil {
		t.Errorf("Expected error for replaced radio home page URL but got none")
	} else if !strings.Contains(err.Error(), "scheme") {
		t.Logf("Error returned for home page URL: %s\n", err)
		t.Errorf("Expected the error to mention that URL scheme is not supported")
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
