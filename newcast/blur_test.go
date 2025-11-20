package newcast

import (
	"fmt"
	"math"
	"os"
	"testing"
)

// TestGaussianBlurEffect compares flow grids with and without Gaussian blur
// to determine if the blur makes a meaningful difference
func TestGaussianBlurEffect(t *testing.T) {
	// Use real rainfall data
	matches, err := os.ReadDir("../rainfall_data")
	if err != nil || len(matches) == 0 {
		t.Skip("Rainfall data not found")
		return
	}

	// Get file paths
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

	config := ProcessConfig{
		MaxFeatures:      1000,
		Smoothness:       0.1,
		FilterType:       "smoothness",
		MaxAngle:         0.3,
		GridCellSize:     64,
		MinTracksPerCell: 2,
		MaxTracksPerCell: 5,
		MinTrackLength:   6,
	}

	tracks, width, height, err := ProcessFilesToTracks(files, config)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Processing %d tracks from %dx%d images\n", len(tracks), width, height)

	// Generate grid without blur
	gridNoBlur := GenerateFlowGridFromTracks(tracks, width, height, 0.0)

	// Generate grids with different blur levels
	gridBlur1 := GenerateFlowGridFromTracks(tracks, width, height, 1.0)
	gridBlur2 := GenerateFlowGridFromTracks(tracks, width, height, 2.0)

	// Compare the grids
	fmt.Println("\n=== Comparing Grids ===")

	// Calculate statistics
	var sumDiff1, sumDiff2, maxDiff1, maxDiff2 float64
	var countNonZero int

	for i := 0; i < len(gridNoBlur.Data); i++ {
		v0 := gridNoBlur.Data[i]
		v1 := gridBlur1.Data[i]
		v2 := gridBlur2.Data[i]

		mag0 := math.Sqrt(v0.Vx*v0.Vx + v0.Vy*v0.Vy)

		if mag0 > 0.1 { // Only compare non-zero vectors
			countNonZero++

			// Difference with blur sigma=1.0
			diff1 := math.Sqrt((v0.Vx-v1.Vx)*(v0.Vx-v1.Vx) + (v0.Vy-v1.Vy)*(v0.Vy-v1.Vy))
			sumDiff1 += diff1
			if diff1 > maxDiff1 {
				maxDiff1 = diff1
			}

			// Difference with blur sigma=2.0
			diff2 := math.Sqrt((v0.Vx-v2.Vx)*(v0.Vx-v2.Vx) + (v0.Vy-v2.Vy)*(v0.Vy-v2.Vy))
			sumDiff2 += diff2
			if diff2 > maxDiff2 {
				maxDiff2 = diff2
			}
		}
	}

	avgDiff1 := sumDiff1 / float64(countNonZero)
	avgDiff2 := sumDiff2 / float64(countNonZero)

	fmt.Printf("Non-zero vectors: %d (%.1f%% of grid)\n", countNonZero,
		100.0*float64(countNonZero)/float64(len(gridNoBlur.Data)))
	fmt.Printf("\nBlur sigma=1.0:\n")
	fmt.Printf("  Average difference: %.4f\n", avgDiff1)
	fmt.Printf("  Max difference: %.4f\n", maxDiff1)
	fmt.Printf("\nBlur sigma=2.0:\n")
	fmt.Printf("  Average difference: %.4f\n", avgDiff2)
	fmt.Printf("  Max difference: %.4f\n", maxDiff2)

	// Calculate what percentage change this represents
	var sumMag float64
	for i := 0; i < len(gridNoBlur.Data); i++ {
		v := gridNoBlur.Data[i]
		mag := math.Sqrt(v.Vx*v.Vx + v.Vy*v.Vy)
		if mag > 0.1 {
			sumMag += mag
		}
	}
	avgMag := sumMag / float64(countNonZero)

	fmt.Printf("\nAverage vector magnitude: %.4f\n", avgMag)
	fmt.Printf("Blur sigma=1.0 changes vectors by %.2f%% on average\n", 100.0*avgDiff1/avgMag)
	fmt.Printf("Blur sigma=2.0 changes vectors by %.2f%% on average\n", 100.0*avgDiff2/avgMag)

	// Analyze smoothness (gradient magnitude)
	fmt.Println("\n=== Smoothness Analysis ===")
	smoothness0 := calculateSmoothness(gridNoBlur)
	smoothness1 := calculateSmoothness(gridBlur1)
	smoothness2 := calculateSmoothness(gridBlur2)

	fmt.Printf("Average gradient magnitude (lower = smoother):\n")
	fmt.Printf("  No blur: %.4f\n", smoothness0)
	fmt.Printf("  Blur sigma=1.0: %.4f (%.2f%% smoother)\n", smoothness1, 100.0*(smoothness0-smoothness1)/smoothness0)
	fmt.Printf("  Blur sigma=2.0: %.4f (%.2f%% smoother)\n", smoothness2, 100.0*(smoothness0-smoothness2)/smoothness0)

	// Conclusion
	fmt.Println("\n=== Conclusion ===")
	if avgDiff1/avgMag < 0.01 {
		fmt.Println("Gaussian blur has MINIMAL effect (<1% change)")
		fmt.Println("The 3x3 weighted averaging and smooth fill already provide sufficient smoothing.")
	} else if avgDiff1/avgMag < 0.05 {
		fmt.Println("Gaussian blur has SMALL effect (1-5% change)")
		fmt.Println("May provide marginal improvement in smoothness.")
	} else {
		fmt.Println("Gaussian blur has SIGNIFICANT effect (>5% change)")
		fmt.Println("Provides meaningful additional smoothing beyond 3x3 averaging and diffusion.")
	}
}

// calculateSmoothness computes the average gradient magnitude across the grid
func calculateSmoothness(grid *AverageFlowGrid) float64 {
	var sumGradient float64
	count := 0

	for y := 1; y < grid.Height-1; y++ {
		for x := 1; x < grid.Width-1; x++ {
			idx := y*grid.Width + x
			v := grid.Data[idx]

			// Get neighbors
			vN := grid.Data[(y-1)*grid.Width+x]
			vS := grid.Data[(y+1)*grid.Width+x]
			vE := grid.Data[y*grid.Width+(x+1)]
			vW := grid.Data[y*grid.Width+(x-1)]

			// Calculate gradient in x direction
			gradX := math.Sqrt((vE.Vx-vW.Vx)*(vE.Vx-vW.Vx)+(vE.Vy-vW.Vy)*(vE.Vy-vW.Vy)) / 2.0

			// Calculate gradient in y direction
			gradY := math.Sqrt((vS.Vx-vN.Vx)*(vS.Vx-vN.Vx)+(vS.Vy-vN.Vy)*(vS.Vy-vN.Vy)) / 2.0

			// Total gradient magnitude
			gradMag := math.Sqrt(gradX*gradX + gradY*gradY)

			// Only count non-zero areas
			mag := math.Sqrt(v.Vx*v.Vx + v.Vy*v.Vy)
			if mag > 0.1 {
				sumGradient += gradMag
				count++
			}
		}
	}

	if count == 0 {
		return 0
	}
	return sumGradient / float64(count)
}
