package nowcast

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"testing"
)

// createImage creates a single PNG image with a rectangle at a specific position.
func createImage(filePath string, width, height, rectX, rectY, rectSize int) error {
	img := image.NewGray(image.Rect(0, 0, width, height))
	// Fill background with random noise
	for i := range img.Pix {
		img.Pix[i] = uint8(rand.Intn(64)) // Low-level noise
	}

	// Draw a rectangle with random noise for texture
	for y := rectY; y < rectY+rectSize; y++ {
		for x := rectX; x < rectX+rectSize; x++ {
			if x >= 0 && x < width && y >= 0 && y < height {
				// Add noise to provide texture for the optical flow algorithm
				img.SetGray(x, y, color.Gray{Y: uint8(128 + rand.Intn(128))})
			}
		}
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create image file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("failed to encode image as PNG: %w", err)
	}
	return nil
}

// createTestSequence generates a series of images showing a rectangle moving
// at a constant velocity. It returns the file paths of the created images.
func createTestSequence(t *testing.T, numFrames, width, height, rectSize, startX, startY, vx, vy int) []string {
	t.Helper()
	var paths []string
	tempDir := t.TempDir() // Use a temporary directory for test images

	for i := 0; i < numFrames; i++ {
		filePath := fmt.Sprintf("%s/frame_%02d.png", tempDir, i)
		rectX := startX + i*vx
		rectY := startY + i*vy
		err := createImage(filePath, width, height, rectX, rectY, rectSize)
		if err != nil {
			t.Fatalf("Failed to create test image %s: %v", filePath, err)
		}
		paths = append(paths, filePath)
	}
	return paths
}

func TestProcessImagesWithMovingRectangle(t *testing.T) {
	// --- Test Parameters ---
	numFrames := 4
	width, height := 256, 256
	rectSize := 50
	startX, startY := 50, 100
	vx, vy := 10, 0 // Moving 10 pixels right per frame
	gridRes := 4    // 4x4 grid, so each cell is 64x64 pixels
	timeStep := 1.0 // Simple time step

	// --- Generate Test Data ---
	imagePaths := createTestSequence(t, numFrames, width, height, rectSize, startX, startY, vx, vy)
	// No need to defer cleanup, t.TempDir() handles it automatically

	// --- Run the Function Under Test ---
	extrapolationData, err := ProcessImages(imagePaths, gridRes, timeStep)
	if err != nil {
		t.Fatalf("ProcessImages failed: %v", err)
	}

	// --- Assertions ---
	if extrapolationData.GridRes != gridRes {
		t.Errorf("Expected GridRes to be %d, but got %d", gridRes, extrapolationData.GridRes)
	}

	// The rectangle starts at (50, 100) and moves right.
	// It primarily occupies grid cells along the y=1 row (64-127).
	// Let's check the grid cell (1, 1), which covers pixels from x=64 to x=127 and y=64 to y=127.
	// This cell should capture the core of the motion.
	targetGridPoint := image.Point{X: 1, Y: 1} // Corresponds to x=1, y=1 in a 4x4 grid

	gv, ok := extrapolationData.Data[targetGridPoint]
	if !ok {
		t.Fatalf("No extrapolation data found for the target grid point %v", targetGridPoint)
	}

	// --- Velocity Assertions ---
	// --- Velocity Assertions ---
	// The calculated flow might not be exactly vx, vy due to algorithmic artifacts,
	// so we check if it's within a reasonable tolerance.
	tolerance := 1.5
	expectedVx := float64(vx)
	if diff := abs(gv.Vx - expectedVx); diff > tolerance {
		t.Errorf("Expected Vx for grid point %v to be around %.2f, but got %.2f (diff=%.2f)",
			targetGridPoint, expectedVx, gv.Vx, diff)
	}

	expectedVy := float64(vy)
	if diff := abs(gv.Vy - expectedVy); diff > tolerance {
		t.Errorf("Expected Vy for grid point %v to be around %.2f, but got %.2f (diff=%.2f)",
			targetGridPoint, expectedVy, gv.Vy, diff)
	}

	// --- Acceleration Assertions ---
	// Since velocity is constant, acceleration should be close to zero.
	accelTolerance := 0.5
	if abs(gv.Ax) > accelTolerance {
		t.Errorf("Expected Ax to be close to 0.0, but got %.2f", gv.Ax)
	}
	if abs(gv.Ay) > accelTolerance {
		t.Errorf("Expected Ay to be close to 0.0, but got %.2f", gv.Ay)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
