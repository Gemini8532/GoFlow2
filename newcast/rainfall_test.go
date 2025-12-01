package newcast

import (
	"fmt"
	"path/filepath"
	"testing"
)

// TestRainfallDataAveraging tests the averaging on real rainfall data
func TestRainfallDataAveraging(t *testing.T) {
	// Find rainfall data files
	pattern := "../rainfall_data/*.png"
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		t.Skip("Rainfall data not found, skipping test")
		return
	}

	// Use first 10 files
	if len(matches) > 10 {
		matches = matches[:10]
	}

	config := ProcessConfig{
		MaxFeatures:      1000,
		MinTrackLength:   6,
	}

	// Process files to tracks
	filteredTracks, width, height, err := ProcessFilesToTracks(matches, config)
	if err != nil {
		t.Fatalf("Error processing files: %v", err)
	}

	fmt.Printf("Processed %d files, got %d filtered tracks\n", len(matches), len(filteredTracks))
	fmt.Printf("Image dimensions: %dx%d\n", width, height)

	if len(filteredTracks) == 0 {
		t.Skip("No tracks found, skipping averaging test")
		return
	}

	// Generate average flow grid at 256x256 resolution (GenerateFlowGridFromTracks handles scaling internally)
	grid := GenerateFlowGridFromTracks(filteredTracks, width, height, 0.0)

	// Check that the grid has reasonable values
	nonZeroCount := 0
	var sumMagnitude float64
	for _, vec := range grid.Data {
		mag := vec.Vx*vec.Vx + vec.Vy*vec.Vy
		if mag > 0.001 {
			nonZeroCount++
			sumMagnitude += mag
		}
	}

	percentNonZero := float64(nonZeroCount) / float64(len(grid.Data)) * 100
	avgMagnitude := sumMagnitude / float64(nonZeroCount)

	fmt.Printf("Non-zero vectors: %d / %d (%.1f%%)\n", nonZeroCount, len(grid.Data), percentNonZero)
	fmt.Printf("Average magnitude of non-zero vectors: %.4f\n", avgMagnitude)

	// We expect good coverage at 256x256 with smooth fill
	if percentNonZero < 70.0 {
		t.Errorf("Expected at least 70%% non-zero vectors at 256x256, but got %.1f%%", percentNonZero)
	}

	// Check smoothness by looking at gradients
	// Calculate average gradient magnitude (should be low for smooth fields)
	var totalGradient float64
	gradientCount := 0
	for y := 1; y < 255; y++ {
		for x := 1; x < 255; x++ {
			idx := y*256 + x
			idxRight := y*256 + (x + 1)
			idxDown := (y+1)*256 + x

			// Gradient in x direction
			dvx_dx := grid.Data[idxRight].Vx - grid.Data[idx].Vx
			dvy_dx := grid.Data[idxRight].Vy - grid.Data[idx].Vy

			// Gradient in y direction
			dvx_dy := grid.Data[idxDown].Vx - grid.Data[idx].Vx
			dvy_dy := grid.Data[idxDown].Vy - grid.Data[idx].Vy

			gradMag := dvx_dx*dvx_dx + dvy_dx*dvy_dx + dvx_dy*dvx_dy + dvy_dy*dvy_dy
			totalGradient += gradMag
			gradientCount++
		}
	}

	avgGradient := totalGradient / float64(gradientCount)
	fmt.Printf("Average gradient magnitude: %.6f\n", avgGradient)

	// Check for discontinuities along scan lines
	maxDiscontinuity := 0.0
	discontinuityCount := 0
	threshold := 10.0 // Flag jumps larger than this

	// Check horizontal scan lines
	for y := 0; y < 256; y++ {
		for x := 0; x < 255; x++ {
			idx := y*256 + x
			idxNext := y*256 + (x + 1)

			dvx := grid.Data[idxNext].Vx - grid.Data[idx].Vx
			dvy := grid.Data[idxNext].Vy - grid.Data[idx].Vy
			jump := dvx*dvx + dvy*dvy

			if jump > threshold*threshold {
				discontinuityCount++
				if jump > maxDiscontinuity {
					maxDiscontinuity = jump
				}
			}
		}
	}

	fmt.Printf("Discontinuities found: %d (max jump: %.2f)\n", discontinuityCount, maxDiscontinuity)


}
