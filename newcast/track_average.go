package newcast

import (
	"fmt"

	"gocv.io/x/gocv"
)

// AveragedTrack holds the average vector and midpoint for a track.
type AveragedTrack struct {
	Midpoint Point
	Vector   Vector
}

// AverageFlowGrid represents a single grid of averaged vectors.
type AverageFlowGrid struct {
	Width  int
	Height int
	Data   []Vector
}

// CalculateAveragedTracks computes the midpoint and average vector for each track.
func CalculateAveragedTracks(tracks []*Track) []*AveragedTrack {
	averagedTracks := make([]*AveragedTrack, 0, len(tracks))
	for _, track := range tracks {
		if len(track.Points) < 2 {
			continue
		}

		var sumVx, sumVy float64
		for i := 0; i < len(track.Points)-1; i++ {
			p1 := track.Points[i]
			p2 := track.Points[i+1]
			sumVx += p2.X - p1.X
			sumVy += p2.Y - p1.Y
		}

		numVectors := float64(len(track.Points) - 1)
		avgVector := Vector{
			Vx: sumVx / numVectors,
			Vy: sumVy / numVectors,
		}

		midpointIndex := len(track.Points) / 2
		midpoint := track.Points[midpointIndex]

		averagedTracks = append(averagedTracks, &AveragedTrack{
			Midpoint: midpoint,
			Vector:   avgVector,
		})
	}
	return averagedTracks
}

// GenerateAverageFlowGrid creates an AverageFlowGrid from averaged tracks using smooth diffusion.
// This uses the SmoothFill algorithm to diffuse vector values from known track points.
// The grid is generated at 256x256 resolution for efficiency, then scaled to the requested size.
// GenerateAverageFlowGrid creates an AverageFlowGrid from averaged tracks using smooth diffusion.
// This uses the SmoothFill algorithm to diffuse vector values from known track points.
// The grid is always generated at 256x256 resolution for efficiency and to avoid interpolation artifacts.
func GenerateAverageFlowGrid(averagedTracks []*AveragedTrack, origWidth, origHeight, _ int) *AverageFlowGrid {
	// Always work at 256x256 to avoid resize artifacts
	const workWidth = 256
	const workHeight = 256

	grid := &AverageFlowGrid{
		Width:  workWidth,
		Height: workHeight,
		Data:   make([]Vector, workWidth*workHeight),
	}

	if len(averagedTracks) == 0 {
		return grid
	}

	scaleX := float64(workWidth) / float64(origWidth)
	scaleY := float64(workHeight) / float64(origHeight)

	// Create OpenCV matrices for Vx, Vy, and mask at working resolution
	vxMat := gocv.NewMatWithSize(workHeight, workWidth, gocv.MatTypeCV32F)
	vyMat := gocv.NewMatWithSize(workHeight, workWidth, gocv.MatTypeCV32F)
	mask := gocv.NewMatWithSize(workHeight, workWidth, gocv.MatTypeCV8U)
	defer vxMat.Close()
	defer vyMat.Close()
	defer mask.Close()

	// Initialize all to zero
	vxMat.SetTo(gocv.NewScalar(0, 0, 0, 0))
	vyMat.SetTo(gocv.NewScalar(0, 0, 0, 0))
	mask.SetTo(gocv.NewScalar(0, 0, 0, 0)) // 0 = unknown

	// Write track vectors to the matrices at scaled coordinates
	// First, collect all tracks that map to each pixel (and nearby pixels)
	type pixelData struct {
		sumVx, sumVy float64
		totalWeight  float64
	}
	pixelMap := make(map[int]*pixelData)

	for _, track := range averagedTracks {
		x := int(track.Midpoint.X*scaleX + 0.5)
		y := int(track.Midpoint.Y*scaleY + 0.5)

		// Write to a 3x3 region to smooth input when tracks are close together
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if nx >= 0 && nx < workWidth && ny >= 0 && ny < workHeight {
					idx := ny*workWidth + nx
					if pixelMap[idx] == nil {
						pixelMap[idx] = &pixelData{}
					}
					// Weight by distance (center pixel gets more weight)
					weight := 1.0
					if dx != 0 || dy != 0 {
						weight = 0.5 // Neighbors get half weight
					}
					pixelMap[idx].sumVx += track.Vector.Vx * weight
					pixelMap[idx].sumVy += track.Vector.Vy * weight
					pixelMap[idx].totalWeight += weight
				}
			}
		}
	}

	// Write averaged values to the matrices
	minWeight := 1000.0
	maxWeight := 0.0
	lowWeightCount := 0

	for idx, data := range pixelMap {
		if data.totalWeight > 0 {
			y := idx / workWidth
			x := idx % workWidth
			avgVx := float32(data.sumVx / data.totalWeight)
			avgVy := float32(data.sumVy / data.totalWeight)
			vxMat.SetFloatAt(y, x, avgVx)
			vyMat.SetFloatAt(y, x, avgVy)
			mask.SetUCharAt(y, x, 255)

			if data.totalWeight < minWeight {
				minWeight = data.totalWeight
			}
			if data.totalWeight > maxWeight {
				maxWeight = data.totalWeight
			}
			if data.totalWeight < 1.0 {
				lowWeightCount++
			}
		}
	}

	fmt.Printf("Track averaging: %d pixels written, weight range [%.2f, %.2f], %d pixels with weight < 1.0\n",
		len(pixelMap), minWeight, maxWeight, lowWeightCount)

	// Debug: check value at (48, 144) before smooth fill
	if pixelMap[144*workWidth+48] != nil {
		data := pixelMap[144*workWidth+48]
		fmt.Printf("DEBUG: Pixel (48,144) BEFORE smooth fill: vx=%.2f, vy=%.2f, weight=%.2f\n",
			data.sumVx/data.totalWeight, data.sumVy/data.totalWeight, data.totalWeight)
	} else {
		fmt.Printf("DEBUG: Pixel (48,144) has NO input data (will be filled by smooth fill)\n")
	}

	// Use moderate iterations for good balance of speed and quality
	config := SmoothFillConfig{Loops: 50}

	// Apply smooth fill to both Vx and Vy grids
	filledVx := SmoothFill(vxMat, mask, config)
	filledVy := SmoothFill(vyMat, mask, config)
	defer filledVx.Close()
	defer filledVy.Close()

	// Copy results directly (no resize)
	for y := 0; y < workHeight; y++ {
		for x := 0; x < workWidth; x++ {
			idx := y*workWidth + x
			vx := float64(filledVx.GetFloatAt(y, x))
			vy := float64(filledVy.GetFloatAt(y, x))
			grid.Data[idx] = Vector{Vx: vx, Vy: vy}

			// Debug: check value at (48, 144) after smooth fill
			if x == 48 && y == 144 {
				fmt.Printf("DEBUG: Pixel (48,144) AFTER smooth fill: vx=%.2f, vy=%.2f\n", vx, vy)
			}
		}
	}

	return grid
}
