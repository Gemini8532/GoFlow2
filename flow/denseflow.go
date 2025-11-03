package flow

import (
	"image"
	"image/color"
	"math"

	"gocv.io/x/gocv"
)

// GenerateDenseFlowMap creates a dense flow visualization from sparse feature points.
func GenerateDenseFlowMap(initialPoints, currentPoints gocv.Mat, width, height, resolutionFactor int) (image.Image, error) {
	// Create a Go image for the dense flow visualization
	resultImg := image.NewRGBA(image.Rect(0, 0, width, height))
	resFactor := float32(resolutionFactor)

	// Calculate displacement vectors from initialPoints to currentPoints
	// Store them in a map for sparse to dense conversion
	displacementMap := make(map[image.Point]image.Point)
	for i := 0; i < initialPoints.Rows(); i++ {
		// Get original point
		p0x := initialPoints.GetFloatAt(i, 0)
		p0y := initialPoints.GetFloatAt(i, 1)

		// Get new (tracked) point
		p1x := currentPoints.GetFloatAt(i, 0)
		p1y := currentPoints.GetFloatAt(i, 1)

		// Calculate displacement vector and scale it
		dx := (p1x - p0x) / resFactor
		dy := (p1y - p0y) / resFactor

		// Store displacement vector at the original position, scaled down
		pt := image.Pt(int(p0x/resFactor), int(p0y/resFactor))
		displacementMap[pt] = image.Pt(int(dx), int(dy))
	}

	// Since OpenCV doesn't have a direct sparse interpolation function in gocv,
	// we'll use our efficient algorithm but make it more optimized
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pt := image.Pt(x, y)

			// Check if we have a direct displacement vector for this point
			if disp, exists := displacementMap[pt]; exists {
				// Use the direct displacement
				dx := float64(disp.X)
				dy := float64(disp.Y)

				// Map x,y vectors to r,g scaled by the flow scale factor and centered around the mid level
				// This allows negative values to be represented properly
				r := uint8(math.Min(255, math.Max(0, FlowMidLevel+dx*FlowScaleFactor)))
				g := uint8(math.Min(255, math.Max(0, FlowMidLevel+dy*FlowScaleFactor)))
				b := uint8(0) // Blue channel set to 0

				resultImg.Set(x, y, color.RGBA{r, g, b, 255})
			} else {
				// Interpolate from nearby sparse points using inverse distance weighting
				var totalX, totalY, totalWeight float64

				// Process all sparse points and calculate weighted contributions
				for sparsePt, disp := range displacementMap {
					dx := float64(x - sparsePt.X)
					dy := float64(y - sparsePt.Y)
					distanceSquared := dx*dx + dy*dy

					// Skip if the distance is too large (optional optimization)
					if distanceSquared > 2500 { // 50^2, limiting influence to 50 pixels
						continue
					}

					if distanceSquared < 1.0 {
						distanceSquared = 1.0 // Avoid division by zero
					}

					weight := 1.0 / distanceSquared // Inverse distance squared weighting

					totalX += float64(disp.X) * weight
					totalY += float64(disp.Y) * weight
					totalWeight += weight
				}

				if totalWeight > 0 {
					// Average the weighted contributions
					avgX := totalX / totalWeight
					avgY := totalY / totalWeight

					// Map x,y vectors to r,g scaled by the flow scale factor and centered around the mid level
					r := uint8(math.Min(255, math.Max(0, FlowMidLevel+avgX*FlowScaleFactor)))
					g := uint8(math.Min(255, math.Max(0, FlowMidLevel+avgY*FlowScaleFactor)))
					b := uint8(0) // Blue channel set to 0

					resultImg.Set(x, y, color.RGBA{r, g, b, 255})
				} else {
					// No nearby sparse points, set to middle value (no flow)
					resultImg.Set(x, y, color.RGBA{FlowMidLevel, FlowMidLevel, 0, 255})
				}
			}
		}
	}

	return resultImg, nil
}
