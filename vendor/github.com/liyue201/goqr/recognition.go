package goqr

import (
	"image"
	"image/color"
	"math"
)

// Recognize recognizes the passed image and returns a slice of QRData.
func Recognize(img image.Image) ([]*QRData, error) {
	b := img.Bounds()

	r := NewRecognizer(b.Max.X, b.Max.Y)
	r.Begin()
	switch m := img.(type) {
	case *image.Gray:
		off := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				gray := m.GrayAt(x, y)
				r.SetPixel(x-b.Min.X, y-b.Min.Y, byte(gray.Y))
				off++
			}
		}
	case *image.RGBA:
		off := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				pix := toGrayLuminance(m.At(x, y))
				r.SetPixel(x-b.Min.X, y-b.Min.Y, byte(pix))
				off++
			}
		}
	default:
		off := 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				rgba := color.RGBAModel.Convert(m.At(x, y)).(color.RGBA)
				pix := toGrayLuminance(rgba)
				r.SetPixel(x-b.Min.X, y-b.Min.Y, byte(pix))
				off++
			}
		}
	}
	r.End()

	count := r.Count()
	if count == 0 {
		return nil, ErrNoQRCode
	}

	qrCodes := make([]*QRData, 0)
	for i := 0; i < count; i++ {
		code, err := r.Decode(i)
		if err != nil {
			continue
		}
		qrCodes = append(qrCodes, code)
	}
	if len(qrCodes) == 0 {
		return nil, ErrNoQRCode
	}
	return qrCodes, nil
}

func toGrayLuminance(c color.Color) uint8 {
	rr, gg, bb, _ := c.RGBA()
	r := math.Pow(float64(rr), 2.2)
	g := math.Pow(float64(gg), 2.2)
	b := math.Pow(float64(bb), 2.2)
	y := math.Pow(0.2125*r+0.7154*g+0.0721*b, 1/2.2)
	Y := uint16(y + 0.5)
	return uint8(Y >> 8)
}
