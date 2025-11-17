package newcast

import (
	"errors"
)

// Polynomial represents a quadratic curve of the form f(t) = At^2 + Bt + C.
// It is used to model the motion of a feature in one dimension (either x or y)
// over time.
type Polynomial struct {
	A, B, C float64 // Coefficients of the quadratic polynomial.
}

// FitQuadratic performs a least-squares fit of a quadratic polynomial to a
// series of time-stamped points. It fits the x and y coordinates independently.
// The time 't' is measured in seconds from the timestamp of the first point.
//
// It requires at least 3 points to perform the fit.
//
// The function solves the normal equations for the polynomial coefficients
// using Cramer's rule for a 3x3 system. It returns two Polynomial objects,
// one for the x-coordinates and one for the y-coordinates.
func FitQuadratic(points []Point) (polyX, polyY Polynomial, err error) {
	n := len(points)
	if n < 3 {
		return Polynomial{}, Polynomial{}, errors.New("not enough points to fit quadratic")
	}

	t0 := points[0].Time

	var sumT, sumT2, sumT3, sumT4 float64
	var sumX, sumTX, sumT2X float64
	var sumY, sumTY, sumT2Y float64

	for _, p := range points {
		t := p.Time.Sub(t0).Seconds()
		t2 := t * t
		t3 := t2 * t
		t4 := t3 * t

		sumT += t
		sumT2 += t2
		sumT3 += t3
		sumT4 += t4

		x := float64(p.Vec.X)
		y := float64(p.Vec.Y)

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


// Eval calculates the value of the polynomial at a given time t.
// The time t is typically in seconds from the start of the observation period.
func (p *Polynomial) Eval(t float64) float64 {
	return p.A*t*t + p.B*t + p.C
}

// Velocity calculates the first derivative of the polynomial at time t,
// which represents the instantaneous velocity.
func (p *Polynomial) Velocity(t float64) float64 {
	return 2*p.A*t + p.B
}

// Acceleration calculates the second derivative of the polynomial, which
// represents the constant acceleration.
func (p *Polynomial) Acceleration() float64 {
	return 2 * p.A
}