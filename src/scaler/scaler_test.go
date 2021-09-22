package scaler_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ironsmile/euterpe/src/scaler"
)

// TestScalerSimpleImage creates a very simple image and uses the scaler to reduce it
// in size. Then checks whether it is the desired size.
func TestScalerSimpleImage(t *testing.T) {

	// First, creating a test image.
	testImg := image.NewRGBA(image.Rect(
		0, 0, 150, 200,
	))

	for x := 0; x < 150; x++ {
		for y := 0; y < 200; y++ {
			testImg.Set(x, y, color.RGBA{
				R: 100,
				G: 100,
				B: 100,
				A: 255,
			})
		}
	}

	imgBuf := new(bytes.Buffer)
	if err := png.Encode(imgBuf, testImg); err != nil {
		t.Fatalf("could not encode test image: %s", err)
	}

	// The use the created image in order to scale it down.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sclr := scaler.New(ctx)
	defer sclr.Cancel()

	imgBytes, err := sclr.Scale(ctx, imgBuf, 50)
	if err != nil {
		t.Fatalf("scaling the test image failed: %s", err)
	}

	scaledImage := bytes.NewBuffer(imgBytes)
	img, _, err := image.Decode(scaledImage)
	if err != nil {
		t.Fatalf("the scaled image cannot be decoded: %s", err)
	}

	imgBounds := img.Bounds()
	if imgBounds.Max.X != 50 {
		t.Errorf("expected image width 50 but got %d", imgBounds.Max.X)
	}
}

// TestScalingNonImageCausesAnError makes sure that trying to scale a non-image will
// cause an error.
func TestScalingNonImageCausesAnError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sclr := scaler.New(ctx)
	defer sclr.Cancel()

	notImage := bytes.NewBufferString("definitely not an image")

	_, err := sclr.Scale(ctx, notImage, 100)
	if err == nil {
		t.Fatalf("scaling an non-image did not cause an error")
	}

	if !strings.Contains(err.Error(), "decoding image") {
		t.Errorf("scaling error did not mention that the problem was decoding")
	}
}

// TestScalerCancel makes sure that the Scaler is not usable after cancel and that
// cancel actually stops its workers.
func TestScalerCancel(t *testing.T) {
	tests := []struct {
		desc            string
		cancelledScaler func() scaler.Scaler
	}{
		{
			desc: "cancelled after using its own cancel func",
			cancelledScaler: func() scaler.Scaler {
				sclr := scaler.New(context.Background())
				sclr.Cancel()
				return sclr
			},
		},
		{
			desc: "cancelled after its context is cancelled",
			cancelledScaler: func() scaler.Scaler {
				ctx, cancel := context.WithCancel(context.Background())

				sclr := scaler.New(ctx)
				cancel()
				time.Sleep(5 * time.Millisecond)
				return sclr
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			sclr := test.cancelledScaler()
			testImgStr := "not actually an image but OK"
			testImg := bytes.NewBufferString(testImgStr)

			ctx := context.Background()
			_, err := sclr.Scale(ctx, testImg, 200)
			if !errors.Is(err, scaler.ErrCancelled) {
				t.Errorf("using cancelled scaler did not cause scaler.ErrCancelled")
			}

			readTestImg, err := io.ReadAll(testImg)
			if err != nil {
				t.Errorf("error while reading from test image: %s", err)
			}

			if string(readTestImg) != testImgStr {
				t.Errorf("scaler was reading from the test image even though it is cancelled")
			}
		})
	}

}
