package flow

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"gocv.io/x/gocv"
)

const (
	// Original image dimensions
	originalWidth  = 1024
	originalHeight = 1024

	// Flow visualization constants
	FlowScaleFactor = 10.0 // Scaling factor for flow vectors
	FlowMidLevel    = 128  // Mid-level value for centering flow visualization
)

func CreateVisualization(initialPoints gocv.Mat, currentPoints gocv.Mat, scaledHeight int, scaledWidth int) gocv.Mat {
	flowMap := gocv.NewMatWithSize(scaledHeight, scaledWidth, gocv.MatTypeCV8UC3)
	flowMap.SetTo(gocv.NewScalar(0, 0, 0, 0)) // Black background

	// Define colors for drawing
	lineColor := color.RGBA{R: 0, G: 255, B: 0, A: 255} // Green lines for vectors
	dotColor := color.RGBA{R: 255, G: 0, B: 0, A: 25}   // Blue dots for endpoints

	for i := 0; i < initialPoints.Rows(); i++ {
		// Get original point
		p0x := initialPoints.GetFloatAt(i, 0)
		p0y := initialPoints.GetFloatAt(i, 1)
		pt0 := image.Pt(int(p0x), int(p0y))

		// Get new (tracked) point
		p1x := currentPoints.GetFloatAt(i, 0)
		p1y := currentPoints.GetFloatAt(i, 1)

		pt1 := image.Pt(int(p1x), int(p1y))

		// Draw the flow vector as a line
		gocv.Line(&flowMap, pt0, pt1, lineColor, 1)
		// Draw the endpoint as a small circle
		gocv.Circle(&flowMap, pt1, 2, dotColor, -1)
	}
	return flowMap
}

// GenerateAverageFlowMap loads a sequence of images, calculates the sparse optical flow
// by tracking features through the entire sequence, and returns a visualization
// of the total displacement vectors as an image where x and y vectors are mapped to r,g scaled by 10.
// resolutionFactor determines the downscaling (e.g., 4 = 1/4th width & height).
func GenerateAverageFlowMap(imagePaths []string, resolutionFactor int) (image.Image, error) {
	if len(imagePaths) < 2 {
		return nil, fmt.Errorf("at least two images are required, but got %d", len(imagePaths))
	}

	// --- 1. Load and Prepare First Image ---
	scaledWidth := originalWidth / resolutionFactor
	scaledHeight := originalHeight / resolutionFactor

	prevMat, err := loadAndPrepImage(imagePaths[0], scaledWidth, scaledHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to load initial image %s: %w", imagePaths[0], err)
	}
	defer prevMat.Close()

	// --- 2. Find Good Features to Track in First Image ---
	initialPoints := gocv.NewMat()
	defer initialPoints.Close()

	// Find "good" corners in the first image.
	gocv.GoodFeaturesToTrack(prevMat, &initialPoints, 100, 0.3, 7)
	if initialPoints.Rows() == 0 {
		return nil, fmt.Errorf("no features found to track in %s", imagePaths[0])
	}

	// currentPoints will be updated in each iteration of the loop
	currentPoints := initialPoints.Clone()
	defer currentPoints.Close()

	// --- 3. Iteratively Calculate Optical Flow ---
	for i := 1; i < len(imagePaths); i++ {
		nextMat, err := loadAndPrepImage(imagePaths[i], scaledWidth, scaledHeight)
		if err != nil {
			return nil, fmt.Errorf("failed to load image %s: %w", imagePaths[i], err)
		}
		// nextMat will be closed at the end of the loop

		if currentPoints.Rows() == 0 {
			nextMat.Close()
			return nil, fmt.Errorf("all features lost before reaching frame %s", imagePaths[i])
		}

		nextPoints := gocv.NewMat()
		status := gocv.NewMat()
		errMat := gocv.NewMat()

		// Run the pyramidal Lucas-Kanade optical flow algorithm
		gocv.CalcOpticalFlowPyrLK(prevMat, nextMat, currentPoints, nextPoints, &status, &errMat)

		// --- Filter out lost points ---
		// Create new matrices for valid points only
		newInitialRows := []int{}
		for j := 0; j < status.Rows(); j++ {
			if status.GetUCharAt(j, 0) == 1 {
				newInitialRows = append(newInitialRows, j)
			}
		}

		if len(newInitialRows) == 0 {
			nextMat.Close()
			nextPoints.Close()
			status.Close()
			errMat.Close()
			prevMat.Close()
			return nil, fmt.Errorf("all features lost tracking from %s to %s", imagePaths[i-1], imagePaths[i])
		}

		// Create new matrices with only the valid points
		newInitialPoints := gocv.NewMatWithSize(len(newInitialRows), 2, gocv.MatTypeCV32F)
		newCurrentPoints := gocv.NewMatWithSize(len(newInitialRows), 2, gocv.MatTypeCV32F)

		for idx, srcIdx := range newInitialRows {
			// Copy from initialPoints and nextPoints to the new matrices
			x1 := currentPoints.GetFloatAt(srcIdx, 0)
			y1 := currentPoints.GetFloatAt(srcIdx, 1)
			x2 := nextPoints.GetFloatAt(srcIdx, 0)
			y2 := nextPoints.GetFloatAt(srcIdx, 1)

			newInitialPoints.SetFloatAt(idx, 0, x1)
			newInitialPoints.SetFloatAt(idx, 1, y1)
			newCurrentPoints.SetFloatAt(idx, 0, x2)
			newCurrentPoints.SetFloatAt(idx, 1, y2)
		}

		nextPoints.Close()
		status.Close()
		errMat.Close()
		prevMat.Close() // Close the old "prevMat"

		// Close the old point matrices
		initialPoints.Close()
		currentPoints.Close()

		// Re-assign the filtered points for the next iteration
		initialPoints = newInitialPoints
		currentPoints = newCurrentPoints

		// The "next" image becomes the "previous" image for the next loop
		prevMat = nextMat.Clone() // Must clone as nextMat will be closed
		nextMat.Close()
	}

	// --- 4. Create Dense Flow Visualization ---
	// After the loop, initialPoints holds the starting positions (p0)
	// and currentPoints holds the final positions (pN) for all
	// features that survived the entire sequence.

	// Create a Go image for the dense flow visualization
	resultImg := image.NewRGBA(image.Rect(0, 0, scaledWidth, scaledHeight))

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

		// Calculate displacement vector
		dx := p1x - p0x
		dy := p1y - p0y

		// Store displacement vector at the original position
		pt := image.Pt(int(p0x), int(p0y))
		displacementMap[pt] = image.Pt(int(dx), int(dy))
	}

	// Since OpenCV doesn't have a direct sparse interpolation function in gocv,
	// we'll use our efficient algorithm but make it more optimized
	for y := 0; y < scaledHeight; y++ {
		for x := 0; x < scaledWidth; x++ {
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

// loadAndPrepImage opens an image file, verifies its dimensions,
// converts it to grayscale, and resizes it.
func loadAndPrepImage(path string, width, height int) (gocv.Mat, error) {
	// Open file
	f, err := os.Open(path)
	if err != nil {
		return gocv.NewMat(), err
	}
	defer f.Close()

	// Decode PNG image
	img, err := png.Decode(f)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("failed to decode PNG image: %w", err)
	}

	// Verify dimensions
	bounds := img.Bounds()
	if bounds.Dx() != originalWidth || bounds.Dy() != originalHeight {
		return gocv.NewMat(), fmt.Errorf("image is not %dx%d (got %dx%d)",
			originalWidth, originalHeight, bounds.Dx(), bounds.Dy())
	}

	// Convert Go's 'image.Image' to a gocv.Mat in grayscale
	grayMat := gocv.NewMatWithSize(bounds.Dy(), bounds.Dx(), gocv.MatTypeCV8UC1)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert to grayscale: 0.299*R + 0.587*G + 0.114*B
			gray := uint8(0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8))
			grayMat.SetUCharAt(y, x, gray)
		}
	}

	// Resize to the target reduced resolution
	resizedMat := gocv.NewMat()
	gocv.Resize(grayMat, &resizedMat, image.Pt(width, height), 0, 0, gocv.InterpolationArea)
	grayMat.Close()

	return resizedMat, nil
}
