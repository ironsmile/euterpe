package goqr

const (
	qrPixelWhite        = 0
	qrPixelBlack        = 1
	qrPixelRegion       = 2
	qrMaxRegion         = 254
	qrMaxCastones       = 32
	qrMaxGrids          = 8
	qrPerspectiveParams = 8
)

type qrPixelType = uint8

type point struct {
	x int
	y int
}

type qrRegion struct {
	seed     point
	count    int
	capstone int
}

type qrCapstone struct {
	ring    int
	stone   int
	corners [4]point
	center  point
	c       [qrPerspectiveParams]float64
	qrGrid  int
}

type qrGrid struct {
	// Capstone indices
	caps [3]int

	// Alignment pattern region and corner
	alignRegion int
	align       point

	// Timing pattern endpoints
	tpep  [3]point
	hscan int
	vscan int

	// Grid size and perspective transform
	gridSize int
	c        [qrPerspectiveParams]float64
}

/************************************************************************
 * QR-code Version information database
 */

const (
	qrMaxVersion = 40
	qrMaxAliment = 7
)

type qrRsParams struct {
	bs int // Small block size
	dw int // Small data words
	ns int // Number of small blocks
}

type qrVersionInfo struct {
	dataBytes int
	apat      [qrMaxAliment]int
	ecc       [4]qrRsParams
}

type polygonScoreData struct {
	ref     point
	scores  [4]int
	corners []point
}

type neighbour struct {
	index    int
	distance float64
}
