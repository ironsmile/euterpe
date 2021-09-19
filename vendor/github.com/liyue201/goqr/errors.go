package goqr

import "errors"

// Error definition
var (
	ErrNoQRCode        = errors.New("no QR code in image")
	ErrInvalidGridSize = errors.New("invalid grid size")
	ErrInvalidVersion  = errors.New("invalid version")
	ErrFormatEcc       = errors.New("ecc format error")
	ErrDataEcc         = errors.New("ecc data error")
	ErrUnknownDataType = errors.New("unknown data type")
	ErrDataOverflow    = errors.New("data overflow")
	ErrDataUnderflow   = errors.New("data underflow")
)
