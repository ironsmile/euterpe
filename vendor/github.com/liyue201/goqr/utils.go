package goqr

func lineIntersect(p0, p1, q0, q1, r *point) bool {
	// (a, b) is perpendicular to line p
	a := -(p1.y - p0.y)
	b := p1.x - p0.x

	// (c, d) is perpendicular to line q
	c := -(q1.y - q0.y)
	d := q1.x - q0.x

	// e and f are dot products of the respective vectors with p and q
	e := a*p1.x + b*p1.y
	f := c*q1.x + d*q1.y

	// Now we need to solve:
	//     [a b] [rx]   [e]
	//     [c d] [ry] = [f]
	//
	// We do this by inverting the matrix and applying it to (e, f):
	//       [ d -b] [e]   [rx]
	// 1/det [-c  a] [f] = [ry]
	//
	det := (a * d) - (b * c)
	if det == 0 {
		return false
	}
	r.x = (d*e - b*f) / det
	r.y = (-c*e + a*f) / det
	return true
}
