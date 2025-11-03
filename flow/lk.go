package flow

import (
	"fmt"
	"image"
	"image/png"
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

// GenerateAverageFlowMap loads a sequence of images, calculates the sparse optical flow
// by tracking features through the entire sequence, and returns a visualization
// of the total displacement vectors.
func GenerateAverageFlowMap(imagePaths []string, resolutionFactor int) (image.Image, error) {
	if len(imagePaths) < 2 {
		return nil, fmt.Errorf("at least two images are required, but got %d", len(imagePaths))
	}

	initialPoints, currentPoints, err := calculateSparseOpticalFlow(imagePaths)
	if err != nil {
		return nil, err
	}
	defer initialPoints.Close()
	defer currentPoints.Close()

	scaledWidth := originalWidth / resolutionFactor
	scaledHeight := originalHeight / resolutionFactor

	return GenerateDenseFlowMap(initialPoints, currentPoints, scaledWidth, scaledHeight, resolutionFactor)
}

// calculateSparseOpticalFlow computes the sparse optical flow for a sequence of images.
func calculateSparseOpticalFlow(imagePaths []string) (gocv.Mat, gocv.Mat, error) {
	prevMat, err := loadAndPrepImage(imagePaths[0])
	if err != nil {
		return gocv.NewMat(), gocv.NewMat(), fmt.Errorf("failed to load initial image %s: %w", imagePaths[0], err)
	}
	defer prevMat.Close()

	initialPoints, err := findGoodFeatures(prevMat, imagePaths[0])
	if err != nil {
		return gocv.NewMat(), gocv.NewMat(), err
	}

	currentPoints := initialPoints.Clone()

	for i := 1; i < len(imagePaths); i++ {
		nextMat, err := loadAndPrepImage(imagePaths[i])
		if err != nil {
			return gocv.NewMat(), gocv.NewMat(), fmt.Errorf("failed to load image %s: %w", imagePaths[i], err)
		}
		defer nextMat.Close()

		if currentPoints.Rows() == 0 {
			return gocv.NewMat(), gocv.NewMat(), fmt.Errorf("all features lost before reaching frame %s", imagePaths[i])
		}

		newInitialPoints, newCurrentPoints, err := trackFeatures(prevMat, nextMat, initialPoints, currentPoints, imagePaths[i-1], imagePaths[i])
		if err != nil {
			return gocv.NewMat(), gocv.NewMat(), err
		}

		initialPoints.Close()
		currentPoints.Close()

		initialPoints = newInitialPoints
		currentPoints = newCurrentPoints
		prevMat = nextMat.Clone()
	}

	return initialPoints, currentPoints, nil
}

// findGoodFeatures detects good features to track in an image.
func findGoodFeatures(image gocv.Mat, imagePath string) (gocv.Mat, error) {
	points := gocv.NewMat()
	gocv.GoodFeaturesToTrack(image, &points, 100, 0.3, 7)
	if points.Rows() == 0 {
		return gocv.NewMat(), fmt.Errorf("no features found to track in %s", imagePath)
	}
	return points, nil
}

// trackFeatures tracks features between two images using Lucas-Kanade.
func trackFeatures(prevMat, nextMat, initialPoints, currentPoints gocv.Mat, prevImagePath, nextImagePath string) (gocv.Mat, gocv.Mat, error) {
	nextPoints := gocv.NewMat()
	status := gocv.NewMat()
	errMat := gocv.NewMat()
	defer nextPoints.Close()
	defer status.Close()
	defer errMat.Close()

	gocv.CalcOpticalFlowPyrLK(prevMat, nextMat, currentPoints, nextPoints, &status, &errMat)

	newInitialRows := []int{}
	for j := 0; j < status.Rows(); j++ {
		if status.GetUCharAt(j, 0) == 1 {
			newInitialRows = append(newInitialRows, j)
		}
	}

	if len(newInitialRows) == 0 {
		return gocv.NewMat(), gocv.NewMat(), fmt.Errorf("all features lost tracking from %s to %s", prevImagePath, nextImagePath)
	}

	newInitialPoints := gocv.NewMatWithSize(len(newInitialRows), 2, gocv.MatTypeCV32F)
	newCurrentPoints := gocv.NewMatWithSize(len(newInitialRows), 2, gocv.MatTypeCV32F)

	for idx, srcIdx := range newInitialRows {
		x1 := currentPoints.GetFloatAt(srcIdx, 0)
		y1 := currentPoints.GetFloatAt(srcIdx, 1)
		x2 := nextPoints.GetFloatAt(srcIdx, 0)
		y2 := nextPoints.GetFloatAt(srcIdx, 1)

		newInitialPoints.SetFloatAt(idx, 0, x1)
		newInitialPoints.SetFloatAt(idx, 1, y1)
		newCurrentPoints.SetFloatAt(idx, 0, x2)
		newCurrentPoints.SetFloatAt(idx, 1, y2)
	}

	return newInitialPoints, newCurrentPoints, nil
}

// loadAndPrepImage opens an image file, verifies its dimensions, and converts it to grayscale.
func loadAndPrepImage(path string) (gocv.Mat, error) {
	f, err := os.Open(path)
	if err != nil {
		return gocv.NewMat(), err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("failed to decode PNG image: %w", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != originalWidth || bounds.Dy() != originalHeight {
		return gocv.NewMat(), fmt.Errorf("image is not %dx%d (got %dx%d)",
			originalWidth, originalHeight, bounds.Dx(), bounds.Dy())
	}

	grayMat := gocv.NewMatWithSize(bounds.Dy(), bounds.Dx(), gocv.MatTypeCV8UC1)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			gray := uint8(0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8))
			grayMat.SetUCharAt(y, x, gray)
		}
	}
	return grayMat, nil
}
