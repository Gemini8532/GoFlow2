package newcast

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"os"
	"testing"
	"time"

	"gocv.io/x/gocv"
)

// loadImageAsGrayscale loads an image from the given path and converts it to a grayscale gocv.Mat.
func loadImageAsGrayscale(path string) (gocv.Mat, error) {
	imgBytes, err := os.ReadFile(path)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("failed to read image file: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("failed to decode image: %w", err)
	}

	mat, err := gocv.ImageToMatRGBA(img)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("failed to convert image to mat: %w", err)
	}

	gray := gocv.NewMat()
	gocv.CvtColor(mat, &gray, gocv.ColorRGBAToGray)
	mat.Close()

	return gray, nil
}

func TestTracker(t *testing.T) {
	// Paths to test images
	imgPath1 := "../test_data/centered.png"
	imgPath2 := "../test_data/shifted.png"

	// Load images
	img1, err := loadImageAsGrayscale(imgPath1)
	if err != nil {
		t.Fatalf("Failed to load image 1: %v", err)
	}
	defer img1.Close()

	img2, err := loadImageAsGrayscale(imgPath2)
	if err != nil {
		t.Fatalf("Failed to load image 2: %v", err)
	}
	defer img2.Close()

	// Create a new tracker
	maxFeatures := 50
	tracker, err := NewTracker(maxFeatures)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	defer tracker.Close()

	// Add the first image
	ts1 := time.Now()
	if err := tracker.AddImage(img1, ts1); err != nil {
		t.Fatalf("Failed to add first image: %v", err)
	}

	// Check initial tracks
	initialTracks := tracker.GetTracks()
	if len(initialTracks) == 0 {
		t.Fatal("No initial tracks were created.")
	}
	if len(initialTracks) > maxFeatures {
		t.Fatalf("Expected at most %d features, but got %d", maxFeatures, len(initialTracks))
	}
	t.Logf("Found %d initial features to track.", len(initialTracks))

	// Add the second image
	ts2 := ts1.Add(1 * time.Second) // 1 second time difference
	if err := tracker.AddImage(img2, ts2); err != nil {
		t.Fatalf("Failed to add second image: %v", err)
	}

	// Check updated tracks
	updatedTracks := tracker.GetTracks()
	if len(updatedTracks) == 0 {
		t.Fatal("All tracks were lost after the second image.")
	}
	t.Logf("%d tracks survived.", len(updatedTracks))

	// Check the motion of a surviving track
	track := updatedTracks[0]
	if len(track.Points) != 2 {
		t.Fatalf("Expected track to have 2 points, but got %d", len(track.Points))
	}

	p1 := track.Points[0]
	p2 := track.Points[1]

	dx := p2.Vec.X - p1.Vec.X
	dy := p2.Vec.Y - p1.Vec.Y

	// The shift in the test data is (20, 10)
	expectedDx := float32(20.0)
	expectedDy := float32(10.0)

	// Allow some tolerance for the feature detection and tracking
	if math.Abs(float64(dx-expectedDx)) > 1.0 || math.Abs(float64(dy-expectedDy)) > 1.0 {
		t.Errorf("Expected displacement close to (%f, %f), but got (%f, %f)", expectedDx, expectedDy, dx, dy)
	}

	// Check velocity
	// dt is 1 second, so velocity should be equal to displacement
	vx := track.LatestVelocity.X
	vy := track.LatestVelocity.Y
	if math.Abs(float64(vx-expectedDx)) > 1.0 || math.Abs(float64(vy-expectedDy)) > 1.0 {
		t.Errorf("Expected velocity close to (%f, %f), but got (%f, %f)", expectedDx, expectedDy, vx, vy)
	}
}
