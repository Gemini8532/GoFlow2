package flow

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"testing"

	"gocv.io/x/gocv"
)

// calculateAverageFlow decodes a flow map image and computes the average (dx, dy) vector.
// It only considers pixels that represent non-zero flow to avoid the background skewing the result.
func calculateAverageFlow(t *testing.T, img image.Image) (float64, float64) {
	t.Helper()
	bounds := img.Bounds()
	var totalDx, totalDy float64
	numPixels := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, _, _ := img.At(x, y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)

			// Decode the flow vector from the pixel color
			dx := (float64(r8) - FlowMidLevel) / FlowScaleFactor
			dy := (float64(g8) - FlowMidLevel) / FlowScaleFactor

			// Only include pixels with significant flow in the average
			if math.Abs(dx) > 0.1 || math.Abs(dy) > 0.1 {
				totalDx += dx
				totalDy += dy
				numPixels++
			}
		}
	}

	if numPixels == 0 {
		return 0, 0
	}
	return totalDx / float64(numPixels), totalDy / float64(numPixels)
}

// compareImages checks if two images are similar within a certain tolerance.
func compareImages(t *testing.T, img1, img2 image.Image, tolerance int) {
	t.Helper()
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()
	if !bounds1.Eq(bounds2) {
		t.Fatalf("Image bounds are not equal: %v vs %v", bounds1, bounds2)
	}

	var diffPixels int
	for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
		for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
			c1 := img1.At(x, y)
			c2 := img2.At(x, y)
			r1, g1, b1, _ := c1.RGBA()
			r2, g2, b2, _ := c2.RGBA()

			if r1 != r2 || g1 != g2 || b1 != b2 {
				diffPixels++
			}
		}
	}

	if diffPixels > tolerance {
		t.Errorf("Images differ by %d pixels, which is more than the tolerance of %d", diffPixels, tolerance)
	}
}

// TestZeroFlow checks that running optical flow on two identical images produces a zero flow map.
func TestZeroFlow(t *testing.T) {
	// Use an image with features, but run it against itself.
	imagePaths := []string{"../test_data/centered.png", "../test_data/centered.png"}
	resolutionFactor := 4

	flowMap, err := GenerateAverageFlowMap(imagePaths, resolutionFactor)
	if err != nil {
		t.Fatalf("GenerateAverageFlowMap failed: %v", err)
	}

	avgDx, avgDy := calculateAverageFlow(t, flowMap)
	if math.Abs(avgDx) > 0.1 || math.Abs(avgDy) > 0.1 {
		t.Errorf("Expected zero flow, but got average flow of (%f, %f)", avgDx, avgDy)
	}
}

// TestShiftedFlow checks that flow on a shifted image produces the correct average flow vector.
func TestShiftedFlow(t *testing.T) {
	imagePaths := []string{"../test_data/centered.png", "../test_data/shifted.png"}
	resolutionFactor := 4
	// The actual shift is (20, 10), but it's downscaled by the resolution factor.
	expectedDx := 20.0 / float64(resolutionFactor)
	expectedDy := 10.0 / float64(resolutionFactor)

	flowMap, err := GenerateAverageFlowMap(imagePaths, resolutionFactor)
	if err != nil {
		t.Fatalf("GenerateAverageFlowMap failed: %v", err)
	}

	avgDx, avgDy := calculateAverageFlow(t, flowMap)
	if math.Abs(avgDx-expectedDx) > 1.0 || math.Abs(avgDy-expectedDy) > 1.0 {
		t.Errorf("Expected average flow close to (%f, %f), but got (%f, %f)",
			expectedDx, expectedDy, avgDx, avgDy)
	}
}

// TestForwardFlow checks that applying a calculated flow map in forward can reconstruct the original image.
func TestForwardFlow(t *testing.T) {
	imageAPath := "../test_data/centered.png"
	imageBPath := "../test_data/shifted.png"
	resolutionFactor := 4
	scaledWidth := 1024 / resolutionFactor
	scaledHeight := 1024 / resolutionFactor

	// 1. Generate the flow map
	flowMap, err := GenerateAverageFlowMap([]string{imageAPath, imageBPath}, resolutionFactor)
	if err != nil {
		t.Fatalf("GenerateAverageFlowMap failed: %v", err)
	}

	// 2. Save the flow map to a temporary file
	flowMapFile, err := os.CreateTemp("", "flowmap-*.png")
	if err != nil {
		t.Fatalf("Failed to create temp file for flow map: %v", err)
	}
	defer os.Remove(flowMapFile.Name())
	if err := png.Encode(flowMapFile, flowMap); err != nil {
		t.Fatalf("Failed to encode flow map: %v", err)
	}
	flowMapFile.Close() // Close the file so ForwardTransform can open it

	// 3. Resize image B to match the flow map dimensions and save to a temp file
	imgB, err := resizeImage(imageBPath, scaledWidth, scaledHeight)
	if err != nil {
		t.Fatalf("Failed to resize image B: %v", err)
	}
	imgBFile, err := os.CreateTemp("", "imageB-*.png")
	if err != nil {
		t.Fatalf("Failed to create temp file for image B: %v", err)
	}
	defer os.Remove(imgBFile.Name())
	if err := png.Encode(imgBFile, imgB); err != nil {
		t.Fatalf("Failed to encode resized image B: %v", err)
	}
	imgBFile.Close()

	// 4. Run the forward transform
	forwardedImage, err := ForwardTransform(imgBFile.Name(), flowMapFile.Name(), 1.0)
	if err != nil {
		t.Fatalf("ForwardTransform failed: %v", err)
	}

	// 5. Load and resize image A to compare against the result
	expectedImage, err := resizeImage(imageAPath, scaledWidth, scaledHeight)
	if err != nil {
		t.Fatalf("Failed to resize image A: %v", err)
	}

	// 6. Compare the images with a tolerance for minor artifacts
	compareImages(t, expectedImage, forwardedImage, 1500) // Allow up to 1500 pixels to be different
}

// TestSymmetricFlow checks that the flow from A->B is the negative of the flow from B->A.
func TestSymmetricFlow(t *testing.T) {
	imageAPath := "../test_data/centered.png"
	imageBPath := "../test_data/shifted.png"
	resolutionFactor := 4

	// Flow from A to B
	flowAB, err := GenerateAverageFlowMap([]string{imageAPath, imageBPath}, resolutionFactor)
	if err != nil {
		t.Fatalf("GenerateAverageFlowMap(a, b) failed: %v", err)
	}
	avgDxAB, avgDyAB := calculateAverageFlow(t, flowAB)

	// Flow from B to A
	flowBA, err := GenerateAverageFlowMap([]string{imageBPath, imageAPath}, resolutionFactor)
	if err != nil {
		t.Fatalf("GenerateAverageFlowMap(b, a) failed: %v", err)
	}
	avgDxBA, avgDyBA := calculateAverageFlow(t, flowBA)

	// Check if the flow vectors are approximately opposite
	if math.Abs(avgDxAB+avgDxBA) > 1.0 || math.Abs(avgDyAB+avgDyBA) > 1.0 {
		t.Errorf("Expected symmetric flow, but got A->B=(%f, %f) and B->A=(%f, %f)",
			avgDxAB, avgDyAB, avgDxBA, avgDyBA)
	}
}

// --- Test Helpers ---

// resizeImage loads an image and resizes it using gocv.
func resizeImage(imgPath string, width, height int) (image.Image, error) {
	mat := gocv.IMRead(imgPath, gocv.IMReadColor)
	if mat.Empty() {
		return nil, fmt.Errorf("failed to read image %s with gocv", imgPath)
	}
	defer mat.Close()

	resizedMat := gocv.NewMat()
	defer resizedMat.Close()

	gocv.Resize(mat, &resizedMat, image.Pt(width, height), 0, 0, gocv.InterpolationArea)

	return resizedMat.ToImage()
}
