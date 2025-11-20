package newcast

import (
	"fmt"
	"math"
	"os"
	"testing"
)

// TestSmoothFillArtifacts checks if smooth fill creates banding artifacts
func TestSmoothFillArtifacts(t *testing.T) {
	// Use the rainfall test to generate a grid
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

	avgTracks := CalculateAveragedTracks(tracks)
	grid := GenerateAverageFlowGrid(avgTracks, width, height, 4)

	// Check for banding artifacts by looking at magnitude gradients
	// Banding shows up as sudden changes in magnitude along contours
	bandCount := 0
	for y := 1; y < grid.Height-1; y++ {
		for x := 1; x < grid.Width-1; x++ {
			idx := y*grid.Width + x
			mag := math.Sqrt(grid.Data[idx].Vx*grid.Data[idx].Vx + grid.Data[idx].Vy*grid.Data[idx].Vy)

			// Check 4 neighbors
			magN := math.Sqrt(grid.Data[(y-1)*grid.Width+x].Vx*grid.Data[(y-1)*grid.Width+x].Vx +
				grid.Data[(y-1)*grid.Width+x].Vy*grid.Data[(y-1)*grid.Width+x].Vy)
			magS := math.Sqrt(grid.Data[(y+1)*grid.Width+x].Vx*grid.Data[(y+1)*grid.Width+x].Vx +
				grid.Data[(y+1)*grid.Width+x].Vy*grid.Data[(y+1)*grid.Width+x].Vy)
			magE := math.Sqrt(grid.Data[y*grid.Width+(x+1)].Vx*grid.Data[y*grid.Width+(x+1)].Vx +
				grid.Data[y*grid.Width+(x+1)].Vy*grid.Data[y*grid.Width+(x+1)].Vy)
			magW := math.Sqrt(grid.Data[y*grid.Width+(x-1)].Vx*grid.Data[y*grid.Width+(x-1)].Vx +
				grid.Data[y*grid.Width+(x-1)].Vy*grid.Data[y*grid.Width+(x-1)].Vy)

			// Look for sudden drops in magnitude (characteristic of banding)
			avgNeighbor := (magN + magS + magE + magW) / 4.0
			if avgNeighbor > 5.0 && mag < avgNeighbor*0.3 {
				bandCount++
				if bandCount <= 5 {
					fmt.Printf("Potential band at (%d,%d): mag=%.2f, avg_neighbor=%.2f\n", x, y, mag, avgNeighbor)
				}
			}
		}
	}

	fmt.Printf("Found %d pixels with potential banding artifacts (out of %d total)\n", bandCount, grid.Width*grid.Height)

	if bandCount > grid.Width*grid.Height/100 {
		t.Errorf("Too many banding artifacts: %d pixels (>1%% of grid)", bandCount)
	}
}
