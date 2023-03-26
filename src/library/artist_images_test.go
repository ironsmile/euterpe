package library

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/ironsmile/euterpe/src/art"
	"github.com/ironsmile/euterpe/src/art/artfakes"
	"github.com/ironsmile/euterpe/src/scaler/scalerfakes"
)

// TestLocalLibraryFindAndSaveArtistImage checks that the Local library's find artist
// image method is using the library's art.Finder with the correct arguments and
// returns the desired artist image.
func TestLocalLibraryFindAndSaveArtistImage(t *testing.T) {
	var (
		bigImage       = []byte("big-image-is-really-bigger-than-the-small")
		secondBigImage = []byte("second-artist-original-image")
		smallImage     = []byte("small-image")
		ctx            = context.Background()
		mediaFile      = MockMedia{
			artist: "Testy Testov",
			album:  "The Test Strikes Back",
			title:  "One Final Bug",
			track:  1,
			length: 334,
		}
	)

	lib, err := NewLocalLibrary(ctx, SQLiteMemoryFile, getTestMigrationFiles())
	if err != nil {
		t.Fatalf(err.Error())
	}

	if err := lib.Initialize(); err != nil {
		t.Fatalf("Initializing library: %s", err)
	}

	defer func() { _ = lib.Truncate() }()

	fakeAF := &artfakes.FakeFinder{
		GetArtistImageStub: func(_ context.Context, name string) ([]byte, error) {
			if name != mediaFile.artist {
				return nil, art.ErrImageNotFound
			}

			retSlice := make([]byte, len(bigImage))
			copy(retSlice, bigImage)

			return retSlice, nil
		},
	}
	lib.SetArtFinder(fakeAF)

	fakeScaler := &scalerfakes.FakeScaler{
		ScaleStub: func(ctx context.Context, r io.Reader, toWidth int) ([]byte, error) {
			if toWidth != 60 {
				return nil, fmt.Errorf("expected to scale to size 60")
			}

			inputBytes, err := io.ReadAll(r)
			if err != nil {
				return nil, fmt.Errorf("reading input image: %s", err)
			}

			if len(inputBytes) < 1 {
				return nil, fmt.Errorf("input image is empty")
			}

			if !bytes.Equal(bigImage, inputBytes) &&
				!bytes.Equal(secondBigImage, inputBytes) {
				return nil, fmt.Errorf(
					"expected to resize one of the big images but it was `%s`",
					inputBytes,
				)
			}

			imgb := make([]byte, len(smallImage))
			copy(imgb, smallImage)
			return imgb, nil
		},
	}
	lib.SetScaler(fakeScaler)

	mediaFilePath := filepath.FromSlash("path/to/file.mp3")
	if err := lib.insertMediaIntoDatabase(&mediaFile, mediaFilePath); err != nil {
		t.Fatalf("inserting media file failed: %s", err)
	}

	firstArtistID, err := lib.GetArtistID(mediaFile.artist)
	if err != nil {
		t.Errorf("error getting first artist ID: %s", err)
	}

	// Set-up finished. Actual tests start here. First try to find an image for
	// an artist who does not have one in the database.
	assertArtistImage(t, lib, firstArtistID, SmallImage, smallImage)

	// Now search for the original image. It should have been stored in the database
	// as part of creating the small one.
	assertArtistImage(t, lib, firstArtistID, OriginalImage, bigImage)

	// Search for an image for artist who is not in the database at all.
	_, err = lib.FindAndSaveArtistImage(ctx, 42, OriginalImage)
	if !errors.Is(err, ErrArtistNotFound) {
		t.Errorf("expected error `%+v` but got `%+v`", ErrArtistNotFound, err)
	}

	// Now, create a new artist and store an image for it. Then try to get it from the
	// library right away.
	secondFile := MockMedia{
		artist: "Unit Runner",
		album:  "The Test Strikes Back",
		title:  "Good Coverage",
		track:  2,
		length: 621,
	}
	secondFilePath := filepath.FromSlash("path/to/file.mp3")
	if err := lib.insertMediaIntoDatabase(&secondFile, secondFilePath); err != nil {
		t.Fatalf("inserting second media file failed: %s", err)
	}

	secondArtistID, err := lib.GetArtistID(secondFile.artist)
	if err != nil {
		t.Errorf("error getting second artist ID: %s", err)
	}

	err = lib.SaveArtistImage(ctx, secondArtistID, bytes.NewReader(secondBigImage))
	if err != nil {
		t.Fatalf("error saving an artist image: %s", err)
	}
	assertArtistImage(t, lib, secondArtistID, OriginalImage, secondBigImage)

	// Now get the small version of this original image. This tests converting
	// a big original in the database into the desired size when this size was
	// not found.
	assertArtistImage(t, lib, secondArtistID, SmallImage, smallImage)

	// Remove this artist's image from the database and make sure it is
	// deleted.
	if err = lib.RemoveArtistImage(ctx, secondArtistID); err != nil {
		t.Fatalf("error removing artist image: %s", err)
	}

	_, err = lib.FindAndSaveArtistImage(ctx, secondArtistID, OriginalImage)
	if !errors.Is(err, ErrArtworkNotFound) {
		t.Fatalf("expected artwork not found error but got `%+v`", err)
	}

	// Insert a new artist and make sure it caches the "not-found" response at least
	// for a while.
	alwaysNotFoundFinder := &artfakes.FakeFinder{
		GetArtistImageStub: func(_ context.Context, _ string) ([]byte, error) {
			return nil, art.ErrImageNotFound
		},
	}
	lib.SetArtFinder(alwaysNotFoundFinder)

	thirdFile := MockMedia{
		artist: "Not Foundoff",
		album:  "The Test Strikes Back",
		title:  "Always Not There For You",
		track:  3,
		length: 223,
	}
	thirdFilePath := filepath.FromSlash("path/to/not-foundouff.mp3")
	if err := lib.insertMediaIntoDatabase(&thirdFile, thirdFilePath); err != nil {
		t.Fatalf("inserting third media file failed: %s", err)
	}

	thirdArtistID, err := lib.GetArtistID(thirdFile.artist)
	if err != nil {
		t.Errorf("error getting second artist ID: %s", err)
	}

	for i := 0; i < 10; i++ {
		_, err = lib.FindAndSaveArtistImage(ctx, thirdArtistID, OriginalImage)
		if !errors.Is(err, ErrArtworkNotFound) {
			t.Fatalf("expected artwork not found error but got `%+v`", err)
		}
	}

	if alwaysNotFoundFinder.GetArtistImageCallCount() != 1 {
		t.Error("expected artFinder.GetArtistImage to have been called only once")
	}
}

func assertArtistImage(
	t *testing.T,
	lib *LocalLibrary,
	artistID int64,
	size ImageSize,
	expectedImage []byte,
) {
	ctx := context.Background()

	foundImg, err := lib.FindAndSaveArtistImage(ctx, artistID, size)
	if err != nil {
		t.Fatalf("error finding artist image: %s", err)
	}

	foundImgBytes, err := io.ReadAll(foundImg)
	if err != nil {
		t.Fatalf("error reading image reader: %s", err)
	}

	if !bytes.Equal(expectedImage, foundImgBytes) {
		t.Errorf("expected image `%s` but got `%s`", expectedImage, foundImgBytes)
	}
}
