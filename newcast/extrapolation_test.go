package newcast

import (
	"testing"
)

func TestExtrapolationCalculation(t *testing.T) {
	// Create a track with points. The time is now implicit.
	points := []Point{
		{X: 0.0, Y: 0.0}, // t=0
		{X: 2.0, Y: 1.0}, // t=1
		{X: 4.0, Y: 2.0}, // t=2
		{X: 6.0, Y: 3.0}, // t=3
		{X: 8.0, Y: 4.0}, // t=4
	}

	track := &Track{
		ID:     1,
		Points: points,
	}

	// The average time delta is implicitly 1.0
	avgDt := 1.0

	// Fit the polynomial
	polyX, polyY, err := FitQuadratic(points)
	if err != nil {
		t.Fatalf("FitQuadratic failed: %v", err)
	}
	track.PolyX = polyX
	track.PolyY = polyY

	// Simulate the extrapolation calculation
	lastT := float64(len(points) - 1) // Should be 4.0
	numFuturePoints := 4

	extrapolatedTimes := make([]float64, numFuturePoints)
	for j := 1; j <= numFuturePoints; j++ {
		extrapolatedTimes[j-1] = lastT + float64(j)*avgDt // 4.0 + j*1.0
	}

	// The extrapolated times (indices) should be: 5, 6, 7, 8
	expectedTimes := []float64{5.0, 6.0, 7.0, 8.0}
	for i, expected := range expectedTimes {
		if extrapolatedTimes[i] != expected {
			t.Errorf("Expected extrapolated time %d to be %.1f, got %.1f", i, expected, extrapolatedTimes[i])
		}
	}
}