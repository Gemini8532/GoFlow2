package newcast

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestFullPipelineInstrumented tests the complete pipeline with detailed instrumentation
func TestFullPipelineInstrumented(t *testing.T) {
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
		Smoothness:       0.1,
		FilterType:       "smoothness",
		MaxAngle:         0.3,
		GridCellSize:     64,
		MinTracksPerCell: 2,
		MaxTracksPerCell: 5,
		MinTrackLength:   6,
	}

	// STEP 1: Process files to tracks
	fmt.Println("\n=== STEP 1: Processing images to tracks ===")
	filteredTracks, width, height, err := ProcessFilesToTracks(matches, config)
	if err != nil {
		t.Fatalf("Error processing files: %v", err)
	}
	fmt.Printf("Input images: %dx%d\n", width, height)
	fmt.Printf("Filtered tracks: %d\n", len(filteredTracks))

	if len(filteredTracks) == 0 {
		t.Skip("No tracks found")
		return
	}

	// STEP 2: Calculate averaged tracks
	fmt.Println("\n=== STEP 2: Calculating averaged tracks ===")
	averagedTracks := CalculateAveragedTracks(filteredTracks)
	fmt.Printf("Averaged tracks: %d\n", len(averagedTracks))

	// Check for discontinuities in averaged tracks
	checkTrackDiscontinuities(t, averagedTracks, width, height)

	// STEP 3: Generate average flow grid
	fmt.Println("\n=== STEP 3: Generating flow grid at 256x256 ===")
	grid := GenerateAverageFlowGrid(averagedTracks, 256, 256, 5)

	// Check grid for discontinuities
	checkGridDiscontinuities(t, grid, "Generated Grid")

	// Check value ranges
	minVx, maxVx := grid.Data[0].Vx, grid.Data[0].Vx
	minVy, maxVy := grid.Data[0].Vy, grid.Data[0].Vy
	for _, vec := range grid.Data {
		if vec.Vx < minVx {
			minVx = vec.Vx
		}
		if vec.Vx > maxVx {
			maxVx = vec.Vx
		}
		if vec.Vy < minVy {
			minVy = vec.Vy
		}
		if vec.Vy > maxVy {
			maxVy = vec.Vy
		}
	}

	fmt.Printf("Value ranges: Vx [%.2f, %.2f], Vy [%.2f, %.2f]\n", minVx, maxVx, minVy, maxVy)

	// Check if values will be clamped in PNG encoding (int16 range after scaling by 100)
	const scaleFactor = 100.0
	maxEncodableVx := 327.67 // 32767 / 100
	maxEncodableVy := 327.67

	if maxVx > maxEncodableVx || minVx < -maxEncodableVx {
		fmt.Printf("WARNING: Vx values exceed int16 range after scaling! Will be clamped.\n")
	}
	if maxVy > maxEncodableVy || minVy < -maxEncodableVy {
		fmt.Printf("WARNING: Vy values exceed int16 range after scaling! Will be clamped.\n")
	}

	// Save raw grid data for inspection
	saveGridVisualization(t, grid, "debug_grid_raw.png")

	// STEP 4: Test PNG encoding/decoding round trip
	fmt.Println("\n=== STEP 4: Testing PNG encoding round trip ===")
	testPNGRoundTrip(t, grid)

	// STEP 5: Check smoothness of individual components
	fmt.Println("\n=== STEP 5: Checking component smoothness ===")
	checkComponentSmoothness(t, grid)
}

// checkTrackDiscontinuities checks if nearby tracks have very different velocities
func checkTrackDiscontinuities(t *testing.T, tracks []*AveragedTrack, width, height int) {
	scaleX := 256.0 / float64(width)
	scaleY := 256.0 / float64(height)
	const proximityThreshold = 5.0 // pixels at 256x256 scale

	largeJumps := 0
	maxJump := 0.0

	for i := 0; i < len(tracks); i++ {
		for j := i + 1; j < len(tracks); j++ {
			t1, t2 := tracks[i], tracks[j]

			// Scale to 256x256
			x1 := t1.Midpoint.X * scaleX
			y1 := t1.Midpoint.Y * scaleY
			x2 := t2.Midpoint.X * scaleX
			y2 := t2.Midpoint.Y * scaleY

			// Check if tracks are close
			dist := math.Sqrt((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1))
			if dist < proximityThreshold {
				// Check velocity difference
				dvx := t2.Vector.Vx - t1.Vector.Vx
				dvy := t2.Vector.Vy - t1.Vector.Vy
				velDiff := math.Sqrt(dvx*dvx + dvy*dvy)

				if velDiff > 20.0 {
					largeJumps++
					if velDiff > maxJump {
						maxJump = velDiff
					}
				}
			}
		}
	}

	fmt.Printf("Nearby track pairs with large velocity differences: %d (max: %.2f)\n", largeJumps, maxJump)
	if largeJumps > len(tracks)/10 {
		t.Logf("Warning: Many nearby tracks have very different velocities (%d pairs)", largeJumps)
	}
}

// checkGridDiscontinuities checks for discontinuities in the grid
func checkGridDiscontinuities(t *testing.T, grid *AverageFlowGrid, label string) {
	maxJump := 0.0
	discontinuityCount := 0
	threshold := 10.0

	type discontinuity struct {
		x, y int
		jump float64
		dir  string
	}
	var discs []discontinuity

	// Check horizontal
	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width-1; x++ {
			idx := y*grid.Width + x
			idxNext := y*grid.Width + (x + 1)

			dvx := grid.Data[idxNext].Vx - grid.Data[idx].Vx
			dvy := grid.Data[idxNext].Vy - grid.Data[idx].Vy
			jump := math.Sqrt(dvx*dvx + dvy*dvy)

			if jump > threshold {
				discontinuityCount++
				if jump > maxJump {
					maxJump = jump
				}
				if len(discs) < 10 {
					discs = append(discs, discontinuity{x, y, jump, "horizontal"})
				}
			}
		}
	}

	// Check vertical
	for y := 0; y < grid.Height-1; y++ {
		for x := 0; x < grid.Width; x++ {
			idx := y*grid.Width + x
			idxNext := (y+1)*grid.Width + x

			dvx := grid.Data[idxNext].Vx - grid.Data[idx].Vx
			dvy := grid.Data[idxNext].Vy - grid.Data[idx].Vy
			jump := math.Sqrt(dvx*dvx + dvy*dvy)

			if jump > threshold {
				discontinuityCount++
				if jump > maxJump {
					maxJump = jump
				}
				if len(discs) < 10 {
					discs = append(discs, discontinuity{x, y, jump, "vertical"})
				}
			}
		}
	}

	fmt.Printf("%s: Discontinuities (>%.0f): %d, Max jump: %.2f\n", label, threshold, discontinuityCount, maxJump)

	if len(discs) > 0 {
		fmt.Printf("  First few discontinuity locations:\n")
		for _, d := range discs {
			fmt.Printf("    (%d,%d) %s: %.2f\n", d.x, d.y, d.dir, d.jump)
		}
	}

	if discontinuityCount > 100 {
		t.Errorf("%s has too many discontinuities: %d", label, discontinuityCount)
	}
}

// testPNGRoundTrip tests encoding and decoding to ensure no discontinuities are introduced
func testPNGRoundTrip(t *testing.T, originalGrid *AverageFlowGrid) {
	// Create a Frame from the grid
	frame := NewFrame(originalGrid.Width, originalGrid.Height)
	for i := 0; i < len(originalGrid.Data); i++ {
		frame.Data[i] = originalGrid.Data[i]
	}

	// Encode to PNG
	pngData, err := frame.MarshalPNG()
	if err != nil {
		t.Fatalf("Failed to encode PNG: %v", err)
	}
	fmt.Printf("PNG encoded: %d bytes\n", len(pngData))

	// Save for inspection
	os.WriteFile("debug_encoded.png", pngData, 0644)

	// Decode back
	decodedFrame, err := DecodePNG(bytes.NewReader(pngData))
	if err != nil {
		t.Fatalf("Failed to decode PNG: %v", err)
	}

	// Compare original and decoded
	maxDiff := 0.0
	totalDiff := 0.0
	count := 0

	for i := 0; i < len(originalGrid.Data); i++ {
		origVx := originalGrid.Data[i].Vx
		origVy := originalGrid.Data[i].Vy
		decVx := decodedFrame.Data[i].Vx
		decVy := decodedFrame.Data[i].Vy

		diffVx := math.Abs(origVx - decVx)
		diffVy := math.Abs(origVy - decVy)
		diff := math.Sqrt(diffVx*diffVx + diffVy*diffVy)

		totalDiff += diff
		count++

		if diff > maxDiff {
			maxDiff = diff
		}
	}

	avgDiff := totalDiff / float64(count)
	fmt.Printf("PNG round trip - Avg diff: %.6f, Max diff: %.6f\n", avgDiff, maxDiff)

	// Check for discontinuities in decoded grid
	decodedGrid := &AverageFlowGrid{
		Width:  decodedFrame.Width,
		Height: decodedFrame.Height,
		Data:   decodedFrame.Data,
	}
	checkGridDiscontinuities(t, decodedGrid, "Decoded Grid")

	// The encoding should not introduce large errors
	if maxDiff > 1.0 {
		t.Errorf("PNG encoding introduced large errors: max diff = %.6f", maxDiff)
	}
}

// checkComponentSmoothness checks Vx and Vy separately for smoothness
func checkComponentSmoothness(t *testing.T, grid *AverageFlowGrid) {
	// Check Vx component
	vxJumps := 0
	maxVxJump := 0.0

	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width-1; x++ {
			idx := y*grid.Width + x
			idxNext := y*grid.Width + (x + 1)

			jump := math.Abs(grid.Data[idxNext].Vx - grid.Data[idx].Vx)
			if jump > 10.0 {
				vxJumps++
			}
			if jump > maxVxJump {
				maxVxJump = jump
			}
		}
	}

	// Check Vy component
	vyJumps := 0
	maxVyJump := 0.0

	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width-1; x++ {
			idx := y*grid.Width + x
			idxNext := y*grid.Width + (x + 1)

			jump := math.Abs(grid.Data[idxNext].Vy - grid.Data[idx].Vy)
			if jump > 10.0 {
				vyJumps++
			}
			if jump > maxVyJump {
				maxVyJump = jump
			}
		}
	}

	fmt.Printf("Vx component: %d jumps >10, max jump: %.2f\n", vxJumps, maxVxJump)
	fmt.Printf("Vy component: %d jumps >10, max jump: %.2f\n", vyJumps, maxVyJump)

	// Save component visualizations
	saveComponentVisualization(t, grid, "debug_vx.png", "debug_vy.png")
}

// saveGridVisualization saves actual vector values to CSV for inspection
func saveGridVisualization(t *testing.T, grid *AverageFlowGrid, filename string) {
	// Save a few horizontal scan lines to CSV
	csvFile := "debug_scanlines.csv"
	f, err := os.Create(csvFile)
	if err != nil {
		t.Logf("Failed to create CSV: %v", err)
		return
	}
	defer f.Close()

	// Write header
	fmt.Fprintf(f, "y,x,vx,vy,magnitude\n")

	// Save a few scan lines
	scanLines := []int{grid.Height / 4, grid.Height / 2, 3 * grid.Height / 4}
	for _, y := range scanLines {
		for x := 0; x < grid.Width; x++ {
			idx := y*grid.Width + x
			vec := grid.Data[idx]
			mag := math.Sqrt(vec.Vx*vec.Vx + vec.Vy*vec.Vy)
			fmt.Fprintf(f, "%d,%d,%.6f,%.6f,%.6f\n", y, x, vec.Vx, vec.Vy, mag)
		}
	}

	fmt.Printf("Saved scan line data: %s\n", csvFile)
}

// saveComponentVisualization saves component values around discontinuities
func saveComponentVisualization(t *testing.T, grid *AverageFlowGrid, vxFile, vyFile string) {
	// Find a discontinuity location
	discX, discY := -1, -1
	threshold := 10.0

	for y := 0; y < grid.Height && discY < 0; y++ {
		for x := 0; x < grid.Width-1; x++ {
			idx := y*grid.Width + x
			idxNext := y*grid.Width + (x + 1)

			dvx := grid.Data[idxNext].Vx - grid.Data[idx].Vx
			dvy := grid.Data[idxNext].Vy - grid.Data[idx].Vy
			jump := math.Sqrt(dvx*dvx + dvy*dvy)

			if jump > threshold {
				discX, discY = x, y
				break
			}
		}
	}

	if discX < 0 {
		fmt.Println("No discontinuities found for detailed inspection")
		return
	}

	// Save a 11x11 region around the discontinuity
	csvFile := "debug_discontinuity_region.csv"
	f, err := os.Create(csvFile)
	if err != nil {
		t.Logf("Failed to create CSV: %v", err)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "Discontinuity region around (%d,%d)\n", discX, discY)
	fmt.Fprintf(f, "y,x,vx,vy,magnitude\n")

	for dy := -5; dy <= 5; dy++ {
		y := discY + dy
		if y < 0 || y >= grid.Height {
			continue
		}
		for dx := -5; dx <= 5; dx++ {
			x := discX + dx
			if x < 0 || x >= grid.Width {
				continue
			}

			idx := y*grid.Width + x
			vec := grid.Data[idx]
			mag := math.Sqrt(vec.Vx*vec.Vx + vec.Vy*vec.Vy)
			fmt.Fprintf(f, "%d,%d,%.6f,%.6f,%.6f\n", y, x, vec.Vx, vec.Vy, mag)
		}
	}

	fmt.Printf("Saved discontinuity region data: %s (centered at %d,%d)\n", csvFile, discX, discY)

	// Also print the values along the discontinuity line
	fmt.Printf("\nValues along discontinuity at y=%d:\n", discY)
	fmt.Printf("x\tvx\t\tvy\t\tmag\t\tdelta_mag\n")
	prevMag := 0.0
	for x := max(0, discX-5); x <= min(grid.Width-1, discX+5); x++ {
		idx := discY*grid.Width + x
		vec := grid.Data[idx]
		mag := math.Sqrt(vec.Vx*vec.Vx + vec.Vy*vec.Vy)
		deltaMag := mag - prevMag
		marker := ""
		if x == discX || x == discX+1 {
			marker = " <--"
		}
		fmt.Printf("%d\t%.4f\t\t%.4f\t\t%.4f\t\t%.4f%s\n", x, vec.Vx, vec.Vy, mag, deltaMag, marker)
		prevMag = mag
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
