package scaler_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ironsmile/euterpe/src/scaler"
)

// TestScalerSimpleImage creates a very simple image and uses the scaler to reduce it
// in size. Then checks whether it is the desired size.
func TestScalerSimpleImage(t *testing.T) {

}

// TestScalerCancel makes sure that the Scaler is not usable after cancel and that
// cancel actually stops its workers.
func TestScalerCancel(t *testing.T) {
	tests := []struct {
		desc            string
		cancelledScaler func() *scaler.Scaler
	}{
		{
			desc: "cancelled after using its own cancel func",
			cancelledScaler: func() *scaler.Scaler {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()

				sclr := scaler.New(ctx)
				sclr.Cancel()
				return sclr
			},
		},
		{
			desc: "cancelled after its context is cancelled",
			cancelledScaler: func() *scaler.Scaler {
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
