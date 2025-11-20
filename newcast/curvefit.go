package newcast

import (
	"errors"
)

// Polynomial represents the coefficients of a degree 2 polynomial: a*t^2 + b*t + c
type Polynomial struct {
	A, B, C float64
}

// FitQuadratic fits a degree 2 polynomial to the X and Y coordinates of the track points.
// It returns two Polynomials, one for the X dimension and one for the Y dimension.
// This is a direct implementation of a least-squares fit for a quadratic curve.
func FitQuadratic(points []Point) (polyX, polyY Polynomial, err error) {
	n := len(points)
	if n < 3 {
		return Polynomial{}, Polynomial{}, errors.New("not enough points to fit quadratic")
	}

	var sumT, sumT2, sumT3, sumT4 float64
	var sumX, sumTX, sumT2X float64
	var sumY, sumTY, sumT2Y float64

	for i, p := range points {
		t := float64(i)
		t2 := t * t
		t3 := t2 * t
		t4 := t3 * t

		sumT += t
		sumT2 += t2
		sumT3 += t3
		sumT4 += t4

		x := p.X
		y := p.Y

		sumX += x
		sumTX += t * x
		sumT2X += t2 * x

		sumY += y
		sumTY += t * y
		sumT2Y += t2 * y
	}

	// We need to solve the 3x3 system of normal equations:
	// |   n    sumT   sumT2 | | c |   | sumX  |
	// | sumT   sumT2  sumT3 | | b | = | sumTX |
	// | sumT2  sumT3  sumT4 | | a |   | sumT2X|

	N := float64(n)
	A := [][]float64{
		{N, sumT, sumT2},
		{sumT, sumT2, sumT3},
		{sumT2, sumT3, sumT4},
	}

	bX := []float64{sumX, sumTX, sumT2X}
	bY := []float64{sumY, sumTY, sumT2Y}

	// Solve using Cramer's rule (for a 3x3 system)
	detA := determinant(A)
	if detA == 0 {
		return Polynomial{}, Polynomial{}, errors.New("failed to fit curve (singular matrix)")
	}

	// Solve for X
	Ax := copyMatrix(A)
	Ax[0][0], Ax[1][0], Ax[2][0] = bX[0], bX[1], bX[2]
	polyX.C = determinant(Ax) / detA

	Ax = copyMatrix(A)
	Ax[0][1], Ax[1][1], Ax[2][1] = bX[0], bX[1], bX[2]
	polyX.B = determinant(Ax) / detA

	Ax = copyMatrix(A)
	Ax[0][2], Ax[1][2], Ax[2][2] = bX[0], bX[1], bX[2]
	polyX.A = determinant(Ax) / detA

	// Solve for Y
	Ay := copyMatrix(A)
	Ay[0][0], Ay[1][0], Ay[2][0] = bY[0], bY[1], bY[2]
	polyY.C = determinant(Ay) / detA

	Ay = copyMatrix(A)
	Ay[0][1], Ay[1][1], Ay[2][1] = bY[0], bY[1], bY[2]
	polyY.B = determinant(Ay) / detA

	Ay = copyMatrix(A)
	Ay[0][2], Ay[1][2], Ay[2][2] = bY[0], bY[1], bY[2]
	polyY.A = determinant(Ay) / detA

	return polyX, polyY, nil
}

func determinant(m [][]float64) float64 {
	return m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
}

func copyMatrix(m [][]float64) [][]float64 {
	c := make([][]float64, 3)
	for i := range c {
		c[i] = make([]float64, 3)
		copy(c[i], m[i])
	}
	return c
}

// Eval evaluates the polynomial at a given time t.
func (p *Polynomial) Eval(t float64) float64 {
	return p.A*t*t + p.B*t + p.C
}

// Velocity evaluates the velocity (first derivative) of the polynomial at time t.
func (p *Polynomial) Velocity(t float64) float64 {
	return 2*p.A*t + p.B
}

// Acceleration evaluates the acceleration (second derivative) of the polynomial.
func (p *Polynomial) Acceleration() float64 {
	return 2 * p.A
}

// ReplaceWithFittedPoints replaces the raw tracked points with polynomial-fitted points.
// This reduces tracking noise and produces smoother trajectories.
// The polynomial must already be fitted (PolyX and PolyY must be set).
func ReplacWithFittedPoints(tracks []*Track) []*Track {
	fittedTracks := make([]*Track, 0, len(tracks))

	for _, track := range tracks {
		// Fit polynomial if not already done
		polyX := track.PolyX
		polyY := track.PolyY

		if polyX.A == 0 && polyX.B == 0 && polyX.C == 0 {
			// Polynomial not fitted yet, fit it now
			var err error
			polyX, polyY, err = FitQuadratic(track.Points)
			if err != nil {
				// Can't fit, skip this track
				continue
			}
		}

		// Create new track with fitted points
		fittedTrack := &Track{
			ID:     track.ID,
			Points: make([]Point, len(track.Points)),
			PolyX:  polyX,
			PolyY:  polyY,
		}

		// Replace each point with its fitted position
		for i := range track.Points {
			t := float64(i)
			fittedTrack.Points[i] = Point{
				X: polyX.Eval(t),
				Y: polyY.Eval(t),
			}
		}

		fittedTracks = append(fittedTracks, fittedTrack)
	}

	return fittedTracks
}
