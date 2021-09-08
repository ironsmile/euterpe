package library

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ironsmile/euterpe/src/helpers"
)

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
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

func TestMovingFileIntoLibrary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getScannedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()
	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	toBeMoved := filepath.Join(testFiles, "more_mp3s", "test_file_moved.mp3")
	newFile := filepath.Join(testFiles, "library", "test_file_added.mp3")

	if err := copyFile(testMp3, toBeMoved); err != nil {
		t.Fatalf("Copying file to library faild: %s", err)
	}

	if err := os.Rename(toBeMoved, newFile); err != nil {
		os.Remove(toBeMoved)
		t.Fatalf("Was not able to move new file into library: %s", err)
	}

	defer os.Remove(newFile)

	time.Sleep(100 * time.Millisecond)

	checkAddedSong(lib, t)
}

func TestRemovingFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	newFile := filepath.Join(testFiles, "library", "test_file_added.mp3")

	if err := copyFile(testMp3, newFile); err != nil {
		t.Fatalf("Copying file to library faild: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getScannedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	results := lib.Search("")

	if len(results) != 4 {
		t.Errorf("Expected 4 files in the result set but found %d", len(results))
	}

	if err := os.Remove(newFile); err != nil {
		t.Error(err)
	}

	time.Sleep(100 * time.Millisecond)

	results = lib.Search("")
	if len(results) != 3 {
		t.Errorf("Expected 3 files in the result set but found %d", len(results))
	}
}

func TestAddingAndRemovingDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testDir := filepath.Join(testFiles, "more_mp3s", "to_be_moved_directory")
	movedDir := filepath.Join(testFiles, "library", "to_be_moved_directory")
	srcTestMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	dstTestMp3 := filepath.Join(testDir, "test_file_added.mp3")

	if err := os.RemoveAll(testDir); err != nil {
		t.Fatalf("Removing directory needed for tests: %s", err)
	}

	if err := os.RemoveAll(movedDir); err != nil {
		t.Fatalf("Removing directory needed for tests: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getScannedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	if err := os.Mkdir(testDir, 0700); err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(testDir)

	if err := copyFile(srcTestMp3, dstTestMp3); err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(testDir, movedDir); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	checkAddedSong(lib, t)

	if err := os.RemoveAll(movedDir); err != nil {
		t.Error(err)
	}

	time.Sleep(100 * time.Millisecond)

	results := lib.Search("")

	if len(results) != 3 {
		t.Errorf("Expected 3 songs but found %d", len(results))
	}
}

func TestMovingDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testDir := filepath.Join(testFiles, "more_mp3s", "to_be_moved_directory")
	movedDir := filepath.Join(testFiles, "library", "to_be_moved_directory")
	secondPlace := filepath.Join(testFiles, "library", "second_place")
	srcTestMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	dstTestMp3 := filepath.Join(testFiles, "more_mp3s", "to_be_moved_directory",
		"test_file_added.mp3")

	_ = os.RemoveAll(testDir)
	_ = os.RemoveAll(movedDir)
	_ = os.RemoveAll(secondPlace)

	if err := os.Mkdir(testDir, 0700); err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(testDir)

	if err := copyFile(srcTestMp3, dstTestMp3); err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(testDir, movedDir); err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(movedDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getScannedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	checkAddedSong(lib, t)

	if err := os.Rename(movedDir, secondPlace); err != nil {
		t.Error(err)
	} else {
		defer os.RemoveAll(secondPlace)
	}

	time.Sleep(100 * time.Millisecond)

	checkAddedSong(lib, t)

	found := lib.Search("")

	if len(found) != 4 {
		t.Errorf("Expected to find 4 tracks but found %d", len(found))
	}

	found = lib.Search("Added Song")

	if len(found) != 1 {
		t.Fatalf("Did not find exactly one 'Added Song'. Found %d files", len(found))
	}

	foundPath := lib.GetFilePath(found[0].ID)
	expectedPath := filepath.Join(secondPlace, "test_file_added.mp3")

	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("File %s was not found: %s", expectedPath, err)
	}

	if foundPath != expectedPath {
		t.Errorf("File is in %s according to library. But it is actually in %s",
			foundPath, expectedPath)
	}

}

func TestAddingNewFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getScannedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()
	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	newFile := filepath.Join(testFiles, "library", "folder_one", "test_file_added.mp3")

	if err := copyFile(testMp3, newFile); err != nil {
		t.Fatalf("Copying file to library failed: %s", err)
	}

	defer os.Remove(newFile)

	time.Sleep(100 * time.Millisecond)

	checkAddedSong(lib, t)
}

func TestAddingNonRelatedFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getScannedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()
	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")
	newFile := filepath.Join(testFiles, "library", "not_related")

	testLibFiles := func() {
		results := lib.Search("")
		if len(results) != 3 {
			t.Errorf("Expected 3 files in the library but found %d", len(results))
		}
	}

	fh, err := os.Create(newFile)

	if err != nil {
		t.Fatal(err)
	}

	_, err = fh.WriteString("Some contents")
	fh.Close()
	if err != nil {
		t.Error(err)
	}

	time.Sleep(100 * time.Millisecond)
	testLibFiles()

	os.Remove(newFile)

	time.Sleep(100 * time.Millisecond)
	testLibFiles()
}
