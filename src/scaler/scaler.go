package scaler

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"runtime"

	// The following are all image formats supported for converting
	// to other image sizes.
	_ "image/gif"
	_ "image/png"

	// Additional image formats from the x repository.
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/vp8"
	_ "golang.org/x/image/webp"

	"golang.org/x/image/draw"
	"golang.org/x/sync/errgroup"
)

// ErrCancelled is returned when one is trying to interact with an stopped
// scaler.
var ErrCancelled = fmt.Errorf("scale operation on cancelled Scaler")

// description is a scaling instruction.
type description struct {

	// ToWidth tells instructs the scaling to produce an image
	// with this width.
	ToWidth int

	// ImgR is the source of the image which will be scaled.
	ImgR io.Reader

	// Result is the channel on which the result image is
	// returned.
	Result chan Result
}

// Result is a type which encapsulates a result from an image
// conversion.
type Result struct {
	ImgData []byte
	Err     error
}

// Scaler is a utility type which could be used for scaling
// images.
type Scaler struct {
	cancelContext context.CancelFunc
	stopped       bool

	work chan description
}

// Scale converts the image (img) to have width toWidth in pixels while
// preserving its aspect ratio.
func (s *Scaler) Scale(
	ctx context.Context,
	img io.Reader,
	toWidth int,
) ([]byte, error) {
	if s.stopped {
		return nil, ErrCancelled
	}

	desc := description{
		ImgR:    img,
		ToWidth: toWidth,
		Result:  make(chan Result),
	}

	select {
	case s.work <- desc:
	case <-ctx.Done():
		return nil, fmt.Errorf("ctx done while waiting to send scale op: %w", ctx.Err())
	}

	res := <-desc.Result
	if res.Err != nil {
		return nil, res.Err
	}

	return res.ImgData, nil
}

func (s *Scaler) worker() error {
	for desc := range s.work {
		imgData, err := s.scaleImage(desc.ImgR, desc.ToWidth)
		desc.Result <- Result{
			ImgData: imgData,
			Err:     err,
		}
	}

	return nil
}

func (s *Scaler) scaleImage(imgReader io.Reader, toWidth int) ([]byte, error) {
	img, _, err := image.Decode(imgReader)
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %w", err)
	}

	toHeight := toWidth
	imgRect := img.Bounds()
	imgw := imgRect.Max.X - imgRect.Min.X
	imgh := imgRect.Max.Y - imgRect.Min.Y
	if imgw != imgh {
		toHeight = int((float32(imgh) / float32(imgw)) * float32(toWidth))
	}

	dst := image.NewRGBA(image.Rect(0, 0, toWidth, toHeight))

	draw.CatmullRom.Scale(
		dst,
		dst.Bounds(),
		img,
		img.Bounds(),
		draw.Over,
		nil,
	)

	var dstJPEG bytes.Buffer
	if err := jpeg.Encode(&dstJPEG, dst, nil); err != nil {
		return nil, fmt.Errorf("encoding image: %w", err)
	}

	return dstJPEG.Bytes(), nil
}

func (s *Scaler) watchCtx(ctx context.Context) func() error {
	// This function is meant to continuously watch the incoming context
	// and when it is done to close the work channel. This will cause all
	// worker go routines to stop.
	return func() error {
		<-ctx.Done()
		s.stopped = true
		close(s.work)
		return nil
	}
}

// Cancel stops the scaler and of its operations. Users may not use
// any further methods on cancelled scalers.
func (s *Scaler) Cancel() {
	s.stopped = true
	s.cancelContext()
}

// New returns a new scaler, ready for use.
func New(ctx context.Context) *Scaler {
	ctx, cancel := context.WithCancel(ctx)

	s := &Scaler{
		cancelContext: cancel,
		work:          make(chan description),
	}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(s.watchCtx(gctx))
	for i := 0; i < runtime.NumCPU(); i++ {
		g.Go(s.worker)
	}

	return s
}
