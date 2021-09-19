package goqr

// qrCode is used to return information about detected QR codes
// in the input image.
type qrCode struct {
	// The four corners of the QR-code, from top left, clockwise
	corners [4]point

	// The number of cells across in the QR-code. The cell bitmap
	// is a bitmask giving the actual values of cells. If the cell
	// at (x, y) is black, then the following bit is set:
	//
	//    cell_bitmap[i >> 3] & (1 << (i & 7))
	//
	// where i = (y * size) + x.
	//

	size       int
	cellBitmap [qrMaxBimap]uint8
}

// QRData holds the decoded QR-code data
type QRData struct {
	// Various parameters of the QR-code. These can mostly be
	// ignored if you only care about the data.
	Version  int
	EccLevel int
	Mask     int

	// This field is the highest-valued data type found in the QR code.
	DataType int

	// Data Payload. For the Kanji datatype, Payload is encoded as
	// Shift-JIS. For all other datatypes, Payload is ASCII text.
	Payload []uint8

	// ECI assignment number
	Eci uint32
}

func gridBit(code *qrCode, x, y int) int {
	p := uint(y*code.size + x)
	return int((code.cellBitmap[p>>3])>>(p&7)) & 1
}

func readFormat(code *qrCode, data *QRData, which int) error {
	format := uint16(0)
	if which != 0 {
		for i := 0; i < 7; i++ {
			format = (format << 1) | uint16(gridBit(code, 8, code.size-1-i))
		}
		for i := 0; i < 8; i++ {
			format = (format << 1) | uint16(gridBit(code, code.size-8+i, 8))
		}
	} else {
		xs := [15]int{8, 8, 8, 8, 8, 8, 8, 8, 7, 5, 4, 3, 2, 1, 0}
		ys := [15]int{0, 1, 2, 3, 4, 5, 7, 8, 8, 8, 8, 8, 8, 8, 8}
		for i := 14; i >= 0; i-- {
			format = (format << 1) | uint16(gridBit(code, xs[i], ys[i]))
		}
	}

	format ^= 0x5412
	err := correctFormat(&format)
	if err != nil {
		return err
	}

	fdata := format >> 10
	data.EccLevel = int(fdata) >> 3
	data.Mask = int(fdata) & 7
	return nil
}

func readBit(code *qrCode, data *QRData, ds *datastream, i, j int) {
	bitpos := ds.dataBits & 7
	bytepos := ds.dataBits >> 3
	v := gridBit(code, j, i)

	if maskBit(data.Mask, i, j) != 0 {
		v ^= 1
	}

	if v != 0 {
		ds.raw[bytepos] |= 0x80 >> uint32(bitpos)
	}
	ds.dataBits++
}

func readData(code *qrCode, data *QRData, ds *datastream) {
	y := code.size - 1
	x := code.size - 1
	dir := -1
	for x > 0 {
		if x == 6 {
			x--
		}

		if 0 == reservedCell(data.Version, y, x) {
			readBit(code, data, ds, y, x)
		}

		if 0 == reservedCell(data.Version, y, x-1) {
			readBit(code, data, ds, y, x-1)
		}
		y += dir
		if y < 0 || y >= code.size {
			dir = -dir
			x -= 2
			y += dir
		}
	}
}

func decode(code *qrCode, data *QRData) error {
	var ds datastream
	if (code.size-17)%4 != 0 {
		return ErrInvalidGridSize
	}
	data.Version = (code.size - 17) / 4

	if data.Version < 1 || data.Version > qrMaxVersion {
		return ErrInvalidVersion
	}

	// Read format information -- try both locations
	err := readFormat(code, data, 0)
	if err != nil {
		err = readFormat(code, data, 1)
	}
	if err != nil {
		return err
	}

	readData(code, data, &ds)

	err = codestreamEcc(data, &ds)
	if err != nil {
		return err
	}
	err = decodePayload(data, &ds)
	if err != nil {
		return err
	}
	return nil
}
