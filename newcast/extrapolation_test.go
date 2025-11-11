package newcast

import (
	"gocv.io/x/gocv"
	"testing"
	"time"
)

func TestExtrapolationTimeCalculation(t *testing.T) {
	// Create a track with known time intervals
	t0 := time.Now()
	points := []Point{
		{Time: t0, Vec: gocv.Point2f{X: 0, Y: 0}},
		{Time: t0.Add(2 * time.Second), Vec: gocv.Point2f{X: 2, Y: 1}}, // 2s interval
		{Time: t0.Add(4 * time.Second), Vec: gocv.Point2f{X: 4, Y: 2}}, // 2s interval
		{Time: t0.Add(6 * time.Second), Vec: gocv.Point2f{X: 6, Y: 3}}, // 2s interval
		{Time: t0.Add(8 * time.Second), Vec: gocv.Point2f{X: 8, Y: 4}}, // 2s interval
	}

	// Manually create a track to test the time interval calculation
	track := &Track{
		ID:     1,
		Points: points,
	}

	// The average time interval should be 2 seconds (8 seconds total / 4 intervals)
	var avgDt float64
	if len(track.Points) > 1 {
		totalDuration := track.Points[len(track.Points)-1].Time.Sub(track.Points[0].Time).Seconds()
		avgDt = totalDuration / float64(len(track.Points)-1)
	}
	
	expectedAvgDt := 2.0 // 8 seconds total / 4 intervals = 2 seconds per interval
	if avgDt != expectedAvgDt {
		t.Errorf("Expected average time interval to be %.1f, got %.1f", expectedAvgDt, avgDt)
	}
	
	// Now test the polynomial fitting
	polyX, polyY, err := FitQuadratic(points)
	if err != nil {
		t.Fatalf("FitQuadratic failed: %v", err)
	}
	
	track.PolyX = polyX
	track.PolyY = polyY
	
	// Simulate the extrapolation calculation
	lastT := points[len(points)-1].Time.Sub(points[0].Time).Seconds() // Should be 8.0
	numFuturePoints := 4
	
	// Calculate where the extrapolated points should be
	extrapolatedTimes := make([]float64, numFuturePoints)
	for j := 1; j <= numFuturePoints; j++ {
		extrapolatedTimes[j-1] = lastT + float64(j)*avgDt // 8 + j*2
	}
	
	// The extrapolated times should be: 10, 12, 14, 16 seconds
	expectedTimes := []float64{10.0, 12.0, 14.0, 16.0}
	for i, expected := range expectedTimes {
		if extrapolatedTimes[i] != expected {
			t.Errorf("Expected extrapolated time %d to be %.1f, got %.1f", i, expected, extrapolatedTimes[i])
		}
	}
}