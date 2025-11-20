package newcast

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

// GenerateFlowGridFromTracks creates a flow grid directly from raw tracks without pre-averaging.
// This preserves spatial accuracy by using all track points.
// blurSigma: if > 0, applies Gaussian blur with this sigma value (0 = no blur)
func GenerateFlowGridFromTracks(tracks []*Track, origWidth, origHeight int, blurSigma float64) *AverageFlowGrid {
	// Always work at 256x256 to avoid resize artifacts
	const workWidth = 256
	const workHeight = 256

	grid := &AverageFlowGrid{
		Width:  workWidth,
		Height: workHeight,
		Data:   make([]Vector, workWidth*workHeight),
	}

	if len(tracks) == 0 {
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

	// Write all track points to the grid with weighted averaging
	type pixelData struct {
		sumVx, sumVy float64
		totalWeight  float64
	}
	pixelMap := make(map[int]*pixelData)

	// Process each track - use all points, not just midpoint
	for _, track := range tracks {
		// Use all points in the track
		for i := 0; i < len(track.Points)-1; i++ {
			p1 := track.Points[i]
			p2 := track.Points[i+1]

			// Calculate velocity at this point
			vx := p2.X - p1.X
			vy := p2.Y - p1.Y

			// Use midpoint of this segment as location
			midX := (p1.X + p2.X) / 2.0
			midY := (p1.Y + p2.Y) / 2.0

			x := int(midX*scaleX + 0.5)
			y := int(midY*scaleY + 0.5)

			// Write to a 3x3 region for smoothness
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
						pixelMap[idx].sumVx += vx * weight
						pixelMap[idx].sumVy += vy * weight
						pixelMap[idx].totalWeight += weight
					}
				}
			}
		}
	}

	// Write averaged values to the matrices
	for idx, data := range pixelMap {
		if data.totalWeight > 0 {
			y := idx / workWidth
			x := idx % workWidth
			avgVx := float32(data.sumVx / data.totalWeight)
			avgVy := float32(data.sumVy / data.totalWeight)
			vxMat.SetFloatAt(y, x, avgVx)
			vyMat.SetFloatAt(y, x, avgVy)
			mask.SetUCharAt(y, x, 255)
		}
	}

	fmt.Printf("Direct track processing: %d pixels written from %d tracks\n", len(pixelMap), len(tracks))

	// Use moderate iterations for good balance of speed and quality
	config := SmoothFillConfig{Loops: 50}

	// Apply smooth fill to both Vx and Vy grids
	filledVx := SmoothFill(vxMat, mask, config)
	filledVy := SmoothFill(vyMat, mask, config)
	defer filledVx.Close()
	defer filledVy.Close()

	// Optional: Apply Gaussian blur for additional smoothing
	if blurSigma > 0 {
		blurredVx := gocv.NewMat()
		blurredVy := gocv.NewMat()
		defer blurredVx.Close()
		defer blurredVy.Close()

		// Apply Gaussian blur - kernel size based on sigma (typically 3*sigma)
		kernelSize := int(blurSigma*3) | 1 // Ensure odd number
		if kernelSize < 3 {
			kernelSize = 3
		}
		gocv.GaussianBlur(filledVx, &blurredVx, image.Point{X: kernelSize, Y: kernelSize}, blurSigma, blurSigma, gocv.BorderDefault)
		gocv.GaussianBlur(filledVy, &blurredVy, image.Point{X: kernelSize, Y: kernelSize}, blurSigma, blurSigma, gocv.BorderDefault)
		// Copy results from blurred versions
		for y := 0; y < workHeight; y++ {
			for x := 0; x < workWidth; x++ {
				idx := y*workWidth + x
				vx := float64(blurredVx.GetFloatAt(y, x))
				vy := float64(blurredVy.GetFloatAt(y, x))
				grid.Data[idx] = Vector{Vx: vx, Vy: vy}
			}
		}
	} else {
		// Copy results directly (no blur)
		for y := 0; y < workHeight; y++ {
			for x := 0; x < workWidth; x++ {
				idx := y*workWidth + x
				vx := float64(filledVx.GetFloatAt(y, x))
				vy := float64(filledVy.GetFloatAt(y, x))
				grid.Data[idx] = Vector{Vx: vx, Vy: vy}
			}
		}
	}

	return grid
}
