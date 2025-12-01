package newcast

import (
	"fmt"
	"math"
	"os"
	"testing"
)

// TestRawVsFittedPoints compares flow grids generated from raw track points
// versus points from fitted polynomial curves
func TestRawVsFittedPoints(t *testing.T) {
	// Load real rainfall data
	matches, err := os.ReadDir("../rainfall_data")
	if err != nil || len(matches) == 0 {
		t.Skip("Rainfall data not found")
		return
	}

	var files []string
	for _, entry := range matches {
		if !entry.IsDir() {
			files = append(files, "../rainfall_data/"+entry.Name())
		}
	}

	if len(files) < 2 {
		t.Skip("Need at least 2 files")
		return
	}

	files = files[:min(10, len(files))]

	// Get tracks with curve-fit filtering
	config := ProcessConfig{
		MaxFeatures:     1000,
		MinTrackLength:  6,
		MinRSquared:     0.85,
		MaxRMSE:         3.0,
		MaxDeviation:    8.0,
		MaxAcceleration: 2.0,
	}

	tracks, width, height, err := ProcessFilesToTracks(files, config)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Processing %d curve-fit filtered tracks\n", len(tracks))

	// Create tracks with fitted points (replacing raw points with polynomial-fitted points)
	fittedTracks := make([]*Track, len(tracks))
	for i, track := range tracks {
		// Fit polynomial if not already done
		polyX, polyY, err := FitQuadratic(track.Points)
		if err != nil {
			continue
		}

		// Create new track with fitted points
		fittedTrack := &Track{
			ID:     track.ID,
			Points: make([]Point, len(track.Points)),
			PolyX:  polyX,
			PolyY:  polyY,
		}

		// Replace each point with its fitted position
		for j := range track.Points {
			t := float64(j)
			fittedTrack.Points[j] = Point{
				X: polyX.Eval(t),
				Y: polyY.Eval(t),
			}
		}

		fittedTracks[i] = fittedTrack
	}

	// Generate grids from both
	fmt.Println("\nGenerating flow grids...")
	gridRaw := GenerateFlowGridFromTracks(tracks, width, height, 0.0)
	gridFitted := GenerateFlowGridFromTracks(fittedTracks, width, height, 0.0)

	// Compare the grids
	fmt.Println("\n=== Comparison ===")

	var sumDiff, maxDiff float64
	var countNonZero int
	var sumMagRaw, sumMagFitted float64

	for i := 0; i < len(gridRaw.Data); i++ {
		vRaw := gridRaw.Data[i]
		vFitted := gridFitted.Data[i]

		magRaw := math.Sqrt(vRaw.Vx*vRaw.Vx + vRaw.Vy*vRaw.Vy)
		magFitted := math.Sqrt(vFitted.Vx*vFitted.Vx + vFitted.Vy*vFitted.Vy)

		if magRaw > 0.1 || magFitted > 0.1 {
			countNonZero++
			sumMagRaw += magRaw
			sumMagFitted += magFitted

			// Difference
			diff := math.Sqrt((vRaw.Vx-vFitted.Vx)*(vRaw.Vx-vFitted.Vx) +
				(vRaw.Vy-vFitted.Vy)*(vRaw.Vy-vFitted.Vy))
			sumDiff += diff
			if diff > maxDiff {
				maxDiff = diff
			}
		}
	}

	avgDiff := sumDiff / float64(countNonZero)
	avgMagRaw := sumMagRaw / float64(countNonZero)
	avgMagFitted := sumMagFitted / float64(countNonZero)

	fmt.Printf("Non-zero vectors: %d\n", countNonZero)
	fmt.Printf("\nAverage magnitude:\n")
	fmt.Printf("  Raw points: %.4f\n", avgMagRaw)
	fmt.Printf("  Fitted points: %.4f (%.2f%% difference)\n", avgMagFitted,
		100.0*math.Abs(avgMagFitted-avgMagRaw)/avgMagRaw)

	fmt.Printf("\nDifference between grids:\n")
	fmt.Printf("  Average: %.4f (%.2f%% of raw magnitude)\n", avgDiff, 100.0*avgDiff/avgMagRaw)
	fmt.Printf("  Maximum: %.4f\n", maxDiff)

	// Calculate smoothness
	smoothnessRaw := calculateSmoothness(gridRaw)
	smoothnessFitted := calculateSmoothness(gridFitted)

	fmt.Printf("\nSmoothness (gradient magnitude, lower = smoother):\n")
	fmt.Printf("  Raw points: %.4f\n", smoothnessRaw)
	fmt.Printf("  Fitted points: %.4f (%.2f%% smoother)\n", smoothnessFitted,
		100.0*(smoothnessRaw-smoothnessFitted)/smoothnessRaw)

	// Analyze point-to-point noise in tracks
	fmt.Println("\n=== Track Point Noise Analysis ===")
	analyzeTrackNoise(tracks, "Raw tracks")
	analyzeTrackNoise(fittedTracks, "Fitted tracks")

	// Conclusion
	fmt.Println("\n=== Conclusion ===")
	if avgDiff/avgMagRaw < 0.05 {
		fmt.Println("Using fitted points has MINIMAL impact (<5% change)")
		fmt.Println("The smooth fill already handles noise well.")
	} else if avgDiff/avgMagRaw < 0.15 {
		fmt.Println("Using fitted points has MODERATE impact (5-15% change)")
		if smoothnessFitted < smoothnessRaw*0.9 {
			fmt.Println("Fitted points provide SMOOTHER flow fields - RECOMMENDED")
		} else {
			fmt.Println("Benefit is marginal.")
		}
	} else {
		fmt.Println("Using fitted points has SIGNIFICANT impact (>15% change)")
		if smoothnessFitted < smoothnessRaw {
			fmt.Println("Fitted points provide much smoother results - HIGHLY RECOMMENDED")
		} else {
			fmt.Println("Raw points may be better - fitted points diverge too much.")
		}
	}
}

func analyzeTrackNoise(tracks []*Track, label string) {
	var sumJitter, maxJitter float64
	var count int

	for _, track := range tracks {
		if len(track.Points) < 3 {
			continue
		}

		// Calculate jitter as deviation from linear interpolation
		for i := 1; i < len(track.Points)-1; i++ {
			prev := track.Points[i-1]
			curr := track.Points[i]
			next := track.Points[i+1]

			// Expected position (linear interpolation)
			expectedX := (prev.X + next.X) / 2.0
			expectedY := (prev.Y + next.Y) / 2.0

			// Jitter (deviation from expected)
			jitter := math.Sqrt((curr.X-expectedX)*(curr.X-expectedX) +
				(curr.Y-expectedY)*(curr.Y-expectedY))

			sumJitter += jitter
			if jitter > maxJitter {
				maxJitter = jitter
			}
			count++
		}
	}

	if count > 0 {
		avgJitter := sumJitter / float64(count)
		fmt.Printf("%s: avg jitter=%.2f px, max jitter=%.2f px\n", label, avgJitter, maxJitter)
	}
}
