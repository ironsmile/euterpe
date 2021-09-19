package goqr

import (
	"errors"
	"math"
)

const (
	thresholdSMin = 1
	thresholdSDen = 8
	thresholdT    = 5
)

// Error definition
var (
	ErrOutOfRange = errors.New("out of range")
)

// Recognizer is a Qr Code recognizer interface
type Recognizer interface {
	SetPixel(x, y int, val uint8)
	Begin()
	End()
	Count() int
	Decode(index int) (*QRData, error)
}

type recognizer struct {
	pixels    []uint8
	w         int
	h         int
	regions   []qrRegion
	capstones []qrCapstone
	grids     []qrGrid
}

// NewRecognizer news a recognizer
func NewRecognizer(w, h int) Recognizer {
	if w <= 0 || h <= 0 {
		return nil
	}
	return &recognizer{
		h:      h,
		w:      w,
		pixels: make([]qrPixelType, w*h),
	}
}

func (q *recognizer) SetPixel(x, y int, val uint8) {
	q.pixels[x+y*q.w] = val
}

func (q *recognizer) Count() int {
	return len(q.grids)
}

func (q *recognizer) Begin() {
	q.regions = make([]qrRegion, qrPixelRegion)
}

func (q *recognizer) End() {
	q.threshold()
	for i := 0; i < q.h; i++ {
		q.finderScan(i)
	}
	for i := 0; i < len(q.capstones); i++ {
		q.testGrouping(i)
	}
}

func (q *recognizer) extract(index int) (*qrCode, error) {
	code := &qrCode{}

	qr := &q.grids[index]

	if index < 0 || index >= len(q.grids) {
		return nil, ErrOutOfRange
	}

	perspectiveMap(qr.c[:], 0.0, 0.0, &code.corners[0])
	perspectiveMap(qr.c[:], float64(qr.gridSize), 0.0, &code.corners[1])
	perspectiveMap(qr.c[:], float64(qr.gridSize), float64(qr.gridSize), &code.corners[2])
	perspectiveMap(qr.c[:], 0.0, float64(qr.gridSize), &code.corners[3])

	code.size = qr.gridSize
	i := uint(0)
	for y := 0; y < qr.gridSize; y++ {
		for x := 0; x < qr.gridSize; x++ {
			if q.readCell(index, x, y) > 0 {
				code.cellBitmap[i>>3] |= uint8(1 << (i & 7))
			}
			i++
		}
	}
	return code, nil
}

func (q *recognizer) Decode(index int) (*QRData, error) {
	code, err := q.extract(index)
	if err != nil {
		return nil, err
	}
	var data QRData
	data.Payload = make([]uint8, 0)

	err = decode(code, &data)
	return &data, err
}

func (q *recognizer) threshold() {
	var x, y int
	avgW := 0
	avgU := 0
	thresholds := q.w / thresholdSDen

	// Ensure a sane, non-zero value for threshold_s.
	// threshold_s can be zero if the image width is small. We need to avoid
	// SIGFPE as it will be used as divisor.
	if thresholds < thresholdSMin {
		thresholds = thresholdSMin
	}

	for y = 0; y < q.h; y++ {
		row := q.pixels[q.w*y : q.w*(y+1)]
		rowAverage := make([]int, q.w)
		for x = 0; x < q.w; x++ {
			var w, u int
			if y&1 == 1 {
				w = x
				u = q.w - 1 - x
			} else {
				w = q.w - 1 - x
				u = x
			}

			avgW = (avgW*(thresholds-1))/thresholds + int(row[w])
			avgU = (avgU*(thresholds-1))/thresholds + int(row[u])

			rowAverage[w] += avgW
			rowAverage[u] += avgU
		}

		for x = 0; x < q.w; x++ {
			if int(row[x]) < rowAverage[x]*(100-thresholdT)/(200*thresholds) {
				row[x] = qrPixelBlack
			} else {
				row[x] = qrPixelWhite
			}
		}
	}
}

const floodFileMaxDepth = 4096

type spanFunc func(userData interface{}, y, left, right int)

func (q *recognizer) floodFillSeed(x, y, from, to int, span spanFunc, userData interface{}, depth int) {
	left := x
	right := x
	row := q.pixels[y*q.w : (y+1)*q.w]

	if depth >= floodFileMaxDepth {
		return
	}

	for left > 0 && int(row[left-1]) == from {
		left--
	}

	for right < q.w-1 && int(row[right+1]) == from {
		right++
	}

	// Fill the extent
	for i := left; i <= right; i++ {
		row[i] = qrPixelType(to)
	}

	if span != nil {
		span(userData, y, left, right)
	}

	// Seed new flood-fills
	if y > 0 {
		row = q.pixels[(y-1)*q.w : (y)*q.w]
		for i := left; i <= right; i++ {
			if int(row[i]) == from {
				q.floodFillSeed(i, y-1, from, to, span, userData, depth+1)
			}
		}
	}

	if y < q.h-1 {
		row = q.pixels[(y+1)*q.w : (y+2)*q.w]
		for i := left; i <= right; i++ {
			if int(row[i]) == from {
				q.floodFillSeed(i, y+1, from, to, span, userData, depth+1)
			}
		}
	}
}

func areaCount(userData interface{}, y, left, right int) {
	region := userData.(*qrRegion)
	region.count += right - left + 1
}

func (q *recognizer) regionCode(x, y int) int {
	if x < 0 || y < 0 || x >= q.w || y >= q.h {
		return -1
	}
	pixel := int(q.pixels[y*q.w+x])
	if pixel >= qrPixelRegion {
		return pixel
	}
	if pixel == qrPixelWhite {
		return -1
	}
	if len(q.regions) >= qrMaxRegion {
		return -1
	}

	region := len(q.regions)
	q.regions = append(q.regions, qrRegion{})
	box := &q.regions[region]

	box.seed.x = x
	box.seed.y = y
	box.count = 0
	box.capstone = -1

	q.floodFillSeed(x, y, pixel, region, areaCount, box, 0)
	return region
}

func findOneCorner(userData interface{}, y, left, right int) {
	psd := userData.(*polygonScoreData)
	xs := [2]int{left, right}
	dy := y - psd.ref.y

	for i := 0; i < 2; i++ {
		dx := xs[i] - psd.ref.x
		d := dx*dx + dy*dy
		if d > psd.scores[0] {
			psd.scores[0] = d
			psd.corners[0].x = xs[i]
			psd.corners[0].y = y
		}
	}
}

func findOtherCorners(userData interface{}, y, left, right int) {
	psd := userData.(*polygonScoreData)
	xs := [2]int{left, right}

	for i := 0; i < 2; i++ {
		up := xs[i]*psd.ref.x + y*psd.ref.y
		right := xs[i]*-psd.ref.y + y*psd.ref.x
		scores := [4]int{up, right, -up, -right}
		for j := 0; j < 4; j++ {
			if scores[j] > psd.scores[j] {
				psd.scores[j] = scores[j]
				psd.corners[j].x = xs[i]
				psd.corners[j].y = y
			}
		}
	}
}

func (q *recognizer) findRegionCorners(rcode int, ref *point, corners []point) {
	region := &q.regions[rcode]
	psd := polygonScoreData{}
	psd.corners = corners[:]

	psd.ref = *ref
	psd.scores[0] = -1

	q.floodFillSeed(region.seed.x, region.seed.y, rcode, qrPixelBlack, findOneCorner, &psd, 0)

	psd.ref.x = psd.corners[0].x - psd.ref.x
	psd.ref.y = psd.corners[0].y - psd.ref.y

	for i := 0; i < 4; i++ {
		psd.corners[i] = region.seed
	}

	i := region.seed.x*psd.ref.x + region.seed.y*psd.ref.y
	psd.scores[0] = i
	psd.scores[2] = -i

	i = region.seed.x*-psd.ref.y + region.seed.y*psd.ref.x
	psd.scores[1] = i
	psd.scores[3] = -i

	q.floodFillSeed(region.seed.x, region.seed.y, qrPixelBlack, rcode, findOtherCorners, &psd, 0)
}

func (q *recognizer) recordCapstone(ring, stone int) {

	stoneReg := &q.regions[stone]
	ringReg := &q.regions[ring]

	if len(q.capstones) >= qrMaxCastones {
		return
	}

	csIndex := len(q.capstones)
	q.capstones = append(q.capstones, qrCapstone{})
	capstone := &q.capstones[csIndex]

	capstone.qrGrid = -1
	capstone.ring = ring
	capstone.stone = stone
	stoneReg.capstone = csIndex
	ringReg.capstone = csIndex

	// Find the corners of the ring
	q.findRegionCorners(ring, &stoneReg.seed, capstone.corners[:])

	// Set up the perspective transform and find the center
	perspectiveSetup(capstone.c[:], capstone.corners[:], 7.0, 7.0)
	perspectiveMap(capstone.c[:], 3.5, 3.5, &capstone.center)

}

func perspectiveSetup(c []float64, rect []point, w, h float64) {
	x0 := float64(rect[0].x)
	y0 := float64(rect[0].y)
	x1 := float64(rect[1].x)
	y1 := float64(rect[1].y)
	x2 := float64(rect[2].x)
	y2 := float64(rect[2].y)
	x3 := float64(rect[3].x)
	y3 := float64(rect[3].y)
	wden := w * (x2*y3 - x3*y2 + (x3-x2)*y1 + x1*(y2-y3))
	hden := h * (x2*y3 + x1*(y2-y3) - x3*y2 + (x3-x2)*y1)
	c[0] = (x1*(x2*y3-x3*y2) + x0*(-x2*y3+x3*y2+(x2-x3)*y1) + x1*(x3-x2)*y0) / wden
	c[1] = -(x0*(x2*y3+x1*(y2-y3)-x2*y1) - x1*x3*y2 + x2*x3*y1 + (x1*x3-x2*x3)*y0) / hden
	c[2] = x0
	c[3] = (y0*(x1*(y3-y2)-x2*y3+x3*y2) + y1*(x2*y3-x3*y2) + x0*y1*(y2-y3)) / wden
	c[4] = (x0*(y1*y3-y2*y3) + x1*y2*y3 - x2*y1*y3 + y0*(x3*y2-x1*y2+(x2-x3)*y1)) / hden
	c[5] = y0
	c[6] = (x1*(y3-y2) + x0*(y2-y3) + (x2-x3)*y1 + (x3-x2)*y0) / wden
	c[7] = (-x2*y3 + x1*y3 + x3*y2 + x0*(y1-y2) - x3*y1 + (x2-x1)*y0) / hden
}

func perspectiveMap(c []float64, u, v float64, ret *point) {
	den := c[6]*u + c[7]*v + 1.0
	x := (c[0]*u + c[1]*v + c[2]) / den
	y := (c[3]*u + c[4]*v + c[5]) / den
	ret.x = int(x + 0.5)
	ret.y = int(y + 0.5)
}

func perspectiveUnmap(c []float64, in *point, u, v *float64) {
	x := float64(in.x)
	y := float64(in.y)
	den := -c[0]*c[7]*y + c[1]*c[6]*y + (c[3]*c[7]-c[4]*c[6])*x + c[0]*c[4] - c[1]*c[3]
	*u = -(c[1]*(y-c[5]) - c[2]*c[7]*y + (c[5]*c[7]-c[4])*x + c[2]*c[4]) / den
	*v = (c[0]*(y-c[5]) - c[2]*c[6]*y + (c[5]*c[6]-c[3])*x + c[2]*c[3]) / den
}

func (q *recognizer) testCapstone(x, y int, pb []int) {

	ringRight := q.regionCode(x-pb[4], y)
	stone := q.regionCode(x-pb[4]-pb[3]-pb[2], y)
	ringLeft := q.regionCode(x-pb[4]-pb[3]-pb[2]-pb[1]-pb[0], y)

	if ringLeft < 0 || ringRight < 0 || stone < 0 {
		return
	}
	// Left and ring of ring should be connected
	if ringLeft != ringRight {
		return
	}
	// Ring should be disconnected from stone
	if ringLeft == stone {
		return
	}
	stoneReg := &q.regions[stone]
	ringReg := &q.regions[ringLeft]
	/* Already detected */
	if stoneReg.capstone >= 0 || ringReg.capstone >= 0 {
		return
	}
	// Ratio should ideally be 37.5
	ratio := stoneReg.count * 100 / ringReg.count
	if ratio < 10 || ratio > 70 {
		return
	}
	q.recordCapstone(ringLeft, stone)
}

func (q *recognizer) finderScan(y int) {

	row := q.pixels[y*q.w : (y+1)*q.w]
	x := 0
	lastColor := 0
	runLength := 0
	runCount := 0
	pb := make([]int, 5)
	check := [5]int{1, 1, 3, 1, 1}

	for x = 0; x < q.w; x++ {
		color := 0
		if row[x] > 0 {
			color = 1
		}

		if x > 0 && color != lastColor {
			for i := 0; i < 4; i++ {
				pb[i] = pb[i+1]
			}
			pb[4] = runLength
			runLength = 0
			runCount++

			if color == 0 && runCount >= 5 {
				var avg, err int
				ok := true
				avg = (pb[0] + pb[1] + pb[3] + pb[4]) / 4
				err = avg * 3 / 4
				for i := 0; i < 5; i++ {
					if pb[i] < check[i]*avg-err || pb[i] > check[i]*avg+err {
						ok = false
						break
					}
				}
				if ok {
					q.testCapstone(x, y, pb)
				}
			}
		}
		runLength++
		lastColor = color
	}
}

func (q *recognizer) findAlignmentPattern(index int) {
	qr := &q.grids[index]
	c0 := &q.capstones[qr.caps[0]]
	c2 := &q.capstones[qr.caps[2]]

	var a, b, c point
	stepSize := 1
	dir := 0
	var u, v float64

	// Grab our previous estimate of the alignment pattern corner
	b = qr.align

	// Guess another two corners of the alignment pattern so that we
	// can estimate its size.

	perspectiveUnmap(c0.c[:], &b, &u, &v)
	perspectiveMap(c0.c[:], u, v+1.0, &a)
	perspectiveUnmap(c2.c[:], &b, &u, &v)
	perspectiveMap(c2.c[:], u+1.0, v, &c)
	sizeEstimate := int(math.Abs(float64((a.x-b.x)*-(c.y-b.y) + (a.y-b.y)*(c.x-b.x))))

	// Spiral outwards from the estimate point until we find something
	// roughly the right size. Don't look too far from the estimate point

	for stepSize*stepSize < sizeEstimate*100 {
		dxMap := []int{1, 0, -1, 0}
		dyMap := []int{0, -1, 0, 1}

		for i := 0; i < stepSize; i++ {
			code := q.regionCode(b.x, b.y)

			if code >= 0 {
				reg := &q.regions[code]

				if reg.count >= sizeEstimate/2 && reg.count <= sizeEstimate*2 {
					qr.alignRegion = code
					return
				}
			}

			b.x += dxMap[dir]
			b.y += dyMap[dir]
		}

		dir = (dir + 1) % 4
		if (dir & 1) == 1 {
			stepSize++
		}
	}
}

// readCell read a cell from a grid using the currently set perspective
// transform. Returns +/- 1 for black/white, 0 for cells which are
// out of image bounds.
func (q *recognizer) readCell(index, x, y int) int {
	qr := &q.grids[index]
	var p point

	perspectiveMap(qr.c[:], float64(x)+0.5, float64(y)+0.5, &p)
	if p.y < 0 || p.y >= q.h || p.x < 0 || p.x >= q.w {
		return 0
	}
	if q.pixels[p.y*q.w+p.x] != 0 {
		return 1
	}
	return -1
}

func (q *recognizer) fitnessCell(index, x, y int) int {
	qr := &q.grids[index]
	score := 0
	offsets := []float64{0.3, 0.5, 0.7}
	for v := 0; v < 3; v++ {
		for u := 0; u < 3; u++ {
			var p point
			perspectiveMap(qr.c[:], float64(x)+offsets[u], float64(y)+offsets[v], &p)

			if p.y < 0 || p.y >= q.h || p.x < 0 || p.x >= q.w {
				continue
			}
			if q.pixels[p.y*q.w+p.x] != 0 {
				score++
			} else {
				score--
			}
		}
	}
	return score
}

func (q *recognizer) fitnessRing(index, cx, cy, radius int) int {
	score := 0
	for i := 0; i < radius*2; i++ {
		score += q.fitnessCell(index, cx-radius+i, cy-radius)
		score += q.fitnessCell(index, cx-radius, cy+radius-i)
		score += q.fitnessCell(index, cx+radius, cy-radius+i)
		score += q.fitnessCell(index, cx+radius-i, cy+radius)
	}
	return score
}

func (q *recognizer) fitnessApat(index, cx, cy int) int {
	return q.fitnessCell(index, cx, cy) -
		q.fitnessRing(index, cx, cy, 1) +
		q.fitnessRing(index, cx, cy, 2)
}

func (q *recognizer) fitnessCapstone(index, x, y int) int {
	x += 3
	y += 3
	return q.fitnessCell(index, x, y) +
		q.fitnessRing(index, x, y, 1) -
		q.fitnessRing(index, x, y, 2) +
		q.fitnessRing(index, x, y, 3)
}

// fitnessAll compute a fitness score for the currently configured perspective
// transform, using the features we expect to find by scanning the
// grid.
func (q *recognizer) fitnessAll(index int) int {
	qr := &q.grids[index]
	version := (qr.gridSize - 17) / 4
	info := &qrVersionDb[version]
	score := 0

	// Check the timing pattern
	for i := 0; i < qr.gridSize-14; i++ {
		expect := 1
		if i&1 == 0 {
			expect = -1
		}
		score += q.fitnessCell(index, i+7, 6) * expect
		score += q.fitnessCell(index, 6, i+7) * expect
	}

	// Check capstones
	score += q.fitnessCapstone(index, 0, 0)
	score += q.fitnessCapstone(index, qr.gridSize-7, 0)
	score += q.fitnessCapstone(index, 0, qr.gridSize-7)

	if version < 0 || version > qrMaxVersion {
		return score
	}

	// Check alignment patterns
	apCount := 0
	for (apCount < qrMaxAliment) && info.apat[apCount] != 0 {
		apCount++
	}

	for i := 1; i+1 < apCount; i++ {
		score += q.fitnessApat(index, 6, info.apat[i])
		score += q.fitnessApat(index, info.apat[i], 6)
	}

	for i := 1; i < apCount; i++ {
		for j := 1; j < apCount; j++ {
			score += q.fitnessApat(index, info.apat[i], info.apat[j])
		}
	}

	return score
}

func (q *recognizer) jigglePerspective(index int) {
	qr := &q.grids[index]
	best := q.fitnessAll(index)

	adjustments := make([]float64, 8)
	for i := 0; i < 8; i++ {
		adjustments[i] = qr.c[i] * 0.02
	}

	for pass := 0; pass < 5; pass++ {
		for i := 0; i < 16; i++ {
			j := i >> 1
			old := qr.c[j]
			step := adjustments[j]
			var new float64

			if i&1 == 1 {
				new = old + step
			} else {
				new = old - step
			}
			qr.c[j] = new
			test := q.fitnessAll(index)

			if test > best {
				best = test
			} else {
				qr.c[j] = old
			}
		}
		for i := 0; i < 8; i++ {
			adjustments[i] *= 0.5
		}
	}
}

// Once the capstones are in place and an alignment point has been chosen,
// we call this function to set up a grid-reading perspective transform.
func (q *recognizer) setupQrPerspective(index int) {
	qr := &q.grids[index]
	var rect [4]point

	/* Set up the perspective map for reading the grid */
	rect[0] = q.capstones[qr.caps[1]].corners[0]
	rect[1] = q.capstones[qr.caps[2]].corners[0]
	rect[2] = qr.align
	rect[3] = q.capstones[qr.caps[0]].corners[0]

	perspectiveSetup(qr.c[:], rect[:], float64(qr.gridSize-7), float64(qr.gridSize-7))
	q.jigglePerspective(index)
}

func rotateCapstone(cap *qrCapstone, h0, hd *point) {
	copy := [4]point{}

	var best int
	var bestScore int

	for j := 0; j < 4; j++ {
		p := &cap.corners[j]
		score := (p.x-h0.x)*-hd.y + (p.y-h0.y)*hd.x
		if j == 0 || score < bestScore {
			best = j
			bestScore = score
		}
	}
	///* Rotate the capstone */
	for j := 0; j < 4; j++ {
		copy[j] = cap.corners[(j+best)%4]
	}
	for j := 0; j < 4; j++ {
		cap.corners[j] = copy[j]
	}
	perspectiveSetup(cap.c[:], cap.corners[:], 7.0, 7.0)
}

func (q *recognizer) timingScan(p0, p1 *point) int {
	n := p1.x - p0.x
	d := p1.y - p0.y
	x := p0.x
	y := p0.y
	a := 0
	runlength := 0
	count := 0

	var dom, nondom *int
	var domStep int
	var nondomStep int

	if p0.x < 0 || p0.y < 0 || p0.x >= q.w || p0.y >= q.h {
		return -1
	}

	if p1.x < 0 || p1.y < 0 || p1.x >= q.w || p1.y >= q.h {
		return -1
	}

	if math.Abs(float64(n)) > math.Abs(float64(d)) {
		n, d = d, n
		dom = &x
		nondom = &y
	} else {
		dom = &y
		nondom = &x
	}

	if n < 0 {
		n = -n
		nondomStep = -1
	} else {
		nondomStep = 1
	}

	if d < 0 {
		d = -d
		domStep = -1
	} else {
		domStep = 1
	}

	x = p0.x
	y = p0.y
	for i := 0; i <= d; i++ {
		if y < 0 || y >= q.h || x < 0 || x >= q.w {
			break
		}
		pixel := q.pixels[y*q.w+x]

		if pixel > 0 {
			if runlength >= 2 {
				count++
			}
			runlength = 0
		} else {
			runlength++
		}

		a += n
		*dom += domStep
		if a >= d {
			*nondom += nondomStep
			a -= d
		}
	}
	return count
}

func findLeftMostToLine(userData interface{}, y, left, right int) {
	psd := userData.(*polygonScoreData)
	xs := []int{left, right}
	for i := 0; i < 2; i++ {
		d := -psd.ref.y*xs[i] + psd.ref.x*y
		if d < psd.scores[0] {
			psd.scores[0] = d
			psd.corners[0].x = xs[i]
			psd.corners[0].y = y
		}
	}
}

// Try the measure the timing pattern for a given QR code. This does
// not require the global perspective to have been set up, but it
// does require that the capstone corners have been set to their
// canonical rotation.
//
// For each capstone, we find a point in the middle of the ring band
// which is nearest the centre of the code. Using these points, we do
// a horizontal and a vertical timing scan.
func (q *recognizer) measureTimingPattern(index int) int {
	qr := &q.grids[index]
	for i := 0; i < 3; i++ {
		us := []float64{6.5, 6.5, 0.5}
		vs := []float64{0.5, 6.5, 6.5}
		cap := &q.capstones[qr.caps[i]]
		perspectiveMap(cap.c[:], us[i], vs[i], &qr.tpep[i])
	}

	qr.hscan = q.timingScan(&qr.tpep[1], &qr.tpep[2])
	qr.vscan = q.timingScan(&qr.tpep[1], &qr.tpep[0])
	scan := qr.hscan
	if qr.vscan > scan {
		scan = qr.vscan
	}

	// If neither scan worked, we can't go any further.
	if scan < 0 {
		return -1
	}

	// Choose the nearest allowable grid size
	size := scan*2 + 13
	ver := (size - 15) / 4
	qr.gridSize = ver*4 + 17

	return 0
}

func (q *recognizer) recordQrGrid(a, b, c int) {

	if len(q.grids) >= qrMaxGrids {
		return
	}

	// Construct the hypotenuse line from A to C. B should be tothe left of this line.

	h0 := q.capstones[a].center
	var hd point
	hd.x = q.capstones[c].center.x - q.capstones[a].center.x
	hd.y = q.capstones[c].center.y - q.capstones[a].center.y

	// Make sure A-B-C is clockwise
	if (q.capstones[b].center.x-h0.x)*-hd.y+(q.capstones[b].center.y-h0.y)*hd.x > 0 {
		a, c = c, a
		hd.x = -hd.x
		hd.y = -hd.y
	}

	qrIndex := len(q.grids)
	q.grids = append(q.grids, qrGrid{})
	qr := &q.grids[qrIndex]

	qr.caps[0] = a
	qr.caps[1] = b
	qr.caps[2] = c
	qr.alignRegion = -1

	// Rotate each capstone so that corner 0 is top-left with respect
	// to the grid.

	for i := 0; i < 3; i++ {
		cap := &q.capstones[qr.caps[i]]
		rotateCapstone(cap, &h0, &hd)
		cap.qrGrid = qrIndex
	}

	// Check the timing pattern. This doesn't require a perspective transform.

	if q.measureTimingPattern(qrIndex) < 0 {
		goto fail
	}

	// Make an estimate based for the alignment pattern based on extending lines from capstones A and C.
	if !lineIntersect(&q.capstones[a].corners[0],
		&q.capstones[a].corners[1],
		&q.capstones[c].corners[0],
		&q.capstones[c].corners[3],
		&qr.align) {

		goto fail
	}

	// On V2+ grids, we should use the alignment pattern.

	if qr.gridSize > 21 {
		// Try to find the actual location of the alignment pattern.

		q.findAlignmentPattern(qrIndex)

		// Find the point of the alignment pattern closest to the
		// top-left of the QR grid.

		if qr.alignRegion >= 0 {
			var psd polygonScoreData
			psd.corners = make([]point, 1)
			reg := &q.regions[qr.alignRegion]

			// Start from some point inside the alignment pattern
			qr.align = reg.seed
			psd.ref = hd
			psd.corners[0] = qr.align

			psd.scores[0] = -hd.y*qr.align.x + hd.x*qr.align.y

			q.floodFillSeed(reg.seed.x, reg.seed.y, qr.alignRegion, qrPixelBlack, nil, nil, 0)

			q.floodFillSeed(reg.seed.x, reg.seed.y, qrPixelBlack, qr.alignRegion, findLeftMostToLine, &psd, 0)
			qr.align = psd.corners[0]

		}
	}

	q.setupQrPerspective(qrIndex)
	return

	// We've been unable to complete setup for this grid. Undo what we've
	// recorded and pretend it never happened.

fail:
	for i := 0; i < 3; i++ {
		q.capstones[qr.caps[i]].qrGrid = -1
	}
	q.grids = q.grids[:len(q.grids)-1]

}

func (q *recognizer) testNeighbours(i int, hlist []*neighbour, vlist []*neighbour) {
	bestScore := 0.0
	bestH := -1
	bestV := -1

	// Test each possible grouping

	for j := 0; j < len(hlist); j++ {
		hn := hlist[j]

		for k := 0; k < len(vlist); k++ {
			vn := vlist[k]
			score := math.Abs(1.0 - hn.distance/vn.distance)

			if score > 2.5 {
				continue
			}

			if bestH < 0 || score < bestScore {
				bestH = hn.index
				bestV = vn.index
				bestScore = score
			}
		}
	}

	if bestH < 0 || bestV < 0 {
		return
	}

	q.recordQrGrid(bestH, i, bestV)
}

func (q *recognizer) testGrouping(i int) {
	c1 := &q.capstones[i]

	hlist := make([]*neighbour, 0)
	vlist := make([]*neighbour, 0)

	if c1.qrGrid >= 0 {
		return
	}

	// Look for potential neighbours by examining the relative gradients
	// from this capstone to others.

	for j := 0; j < len(q.capstones); j++ {
		c2 := &q.capstones[j]

		if i == j || c2.qrGrid >= 0 {
			continue
		}
		var u, v float64

		perspectiveUnmap(c1.c[:], &c2.center, &u, &v)

		u = math.Abs(u - 3.5)
		v = math.Abs(v - 3.5)

		if u < 0.2*v {
			n := &neighbour{}
			n.index = j
			n.distance = v
			hlist = append(hlist, n)
		}
		if v < 0.2*u {
			n := &neighbour{}
			n.index = j
			n.distance = u
			vlist = append(vlist, n)
		}
	}

	if !(len(hlist) > 0 && len(vlist) > 0) {
		return
	}
	q.testNeighbours(i, hlist, vlist)
}
