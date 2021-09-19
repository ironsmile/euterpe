package goqr

// Limits on the maximum size of QR-codes and their content
const (
	qrMaxBimap   = 3917
	qrMaxPayload = 8896

	// QR-code ECC types
	qrEccLevelM = 0
	qrEccLevelL = 1
	qrEccLevelH = 2
	qrEccLevelQ = 3

	// QR-code data types
	qrDataTypeNumeric = 1
	qrDataTypeAlpha   = 2
	qrDataTypeByte    = 4
	qrDataTypeKanji   = 8

	// Common character encodings
	qrEciIos8859_1  = 1
	qrEciIbm437     = 2
	qrEciIos8859_2  = 4
	qrEciIso8859_3  = 5
	qrEciIso8859_4  = 6
	qrEciIso8859_5  = 7
	qrEciIso8859_6  = 8
	qrEciIso8859_7  = 9
	qrEciIso8859_8  = 10
	qrEciIso8859_9  = 11
	qrEciWindows874 = 13
	qrEciIso8859_13 = 15
	qrEciIso8859_15 = 17
	qrEciShiftJis   = 20
	qrEciUtf8       = 26
)
