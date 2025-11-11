package newcast

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
	"math"
)

// createTestImage creates a new PNG image file with a polygon drawn on it.
func createTestImage(filePath string, polygon []image.Point, width, height int) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Src)

	// This is a simple way to draw a polygon. For more complex shapes, a library would be better.
	// For this test, we'll just draw lines between the points.
	for i := 0; i < len(polygon)-1; i++ {
		drawLine(img, polygon[i], polygon[i+1], color.White)
	}
	if len(polygon) > 1 {
		drawLine(img, polygon[len(polygon)-1], polygon[0], color.White)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

// drawLine draws a line on an image.
func drawLine(img *image.RGBA, start, end image.Point, col color.Color) {
	dx := math.Abs(float64(end.X - start.X))
	dy := math.Abs(float64(end.Y - start.Y))
	sx := -1
	if start.X < end.X {
		sx = 1
	}
	sy := -1
	if start.Y < end.Y {
		sy = 1
	}
	err := dx - dy

	for {
		img.Set(start.X, start.Y, col)
		if start.X == end.X && start.Y == end.Y {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			start.X += sx
		}
		if e2 < dx {
			err += dx
			start.Y += sy
		}
	}
}

func TestTrackerWithIrregularShape(t *testing.T) {
	width, height := 256, 256
	imgPath1 := "irregular_shape1.png"
	imgPath2 := "irregular_shape2.png"
	defer os.Remove(imgPath1)
	defer os.Remove(imgPath2)

	// Define an irregular polygon
	polygon1 := []image.Point{
		{50, 50}, {150, 70}, {180, 150}, {100, 200}, {30, 150},
	}

	// Create the first image
	if err := createTestImage(imgPath1, polygon1, width, height); err != nil {
		t.Fatalf("Failed to create test image 1: %v", err)
	}

	// Define the shifted polygon
	shiftX, shiftY := 20, 15
	polygon2 := make([]image.Point, len(polygon1))
	for i, p := range polygon1 {
		polygon2[i] = image.Point{p.X + shiftX, p.Y + shiftY}
	}

	// Create the second image
	if err := createTestImage(imgPath2, polygon2, width, height); err != nil {
		t.Fatalf("Failed to create test image 2: %v", err)
	}

	// Load images as grayscale mats
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

	// Run the tracker
	tracker, err := NewTracker(100)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	defer tracker.Close()

	ts1 := time.Now()
	if err := tracker.AddImage(img1, ts1); err != nil {
		t.Fatalf("Failed to add first image: %v", err)
	}

	ts2 := ts1.Add(1 * time.Second)
	if err := tracker.AddImage(img2, ts2); err != nil {
		t.Fatalf("Failed to add second image: %v", err)
	}

	// Assertions
	tracks := tracker.GetTracks()
	if len(tracks) < 10 { // Expect a reasonable number of features to be tracked
		t.Fatalf("Expected at least 10 tracks, but got %d", len(tracks))
	}

	// Check the average displacement
	var totalDx, totalDy float32
	for _, track := range tracks {
		if len(track.Points) == 2 {
			p1 := track.Points[0].Vec
			p2 := track.Points[1].Vec
			totalDx += p2.X - p1.X
			totalDy += p2.Y - p1.Y
		}
	}
	avgDx := totalDx / float32(len(tracks))
	avgDy := totalDy / float32(len(tracks))

	if math.Abs(float64(avgDx-float32(shiftX))) > 2.0 || math.Abs(float64(avgDy-float32(shiftY))) > 2.0 {
		t.Errorf("Expected average displacement close to (%d, %d), but got (%.2f, %.2f)", shiftX, shiftY, avgDx, avgDy)
	}
}

func TestTrackerWithChangingShape(t *testing.T) {
	width, height := 256, 256
	imgPath1 := "changing_shape1.png"
	imgPath2 := "changing_shape2.png"
	defer os.Remove(imgPath1)
	defer os.Remove(imgPath2)

	// A rectangle
	polygon1 := []image.Point{
		{50, 50}, {150, 50}, {150, 150}, {50, 150},
	}
	if err := createTestImage(imgPath1, polygon1, width, height); err != nil {
		t.Fatalf("Failed to create test image 1: %v", err)
	}

	// The rectangle deforms (one vertex moves)
	polygon2 := []image.Point{
		{50, 50}, {160, 60}, {150, 150}, {50, 150},
	}
	if err := createTestImage(imgPath2, polygon2, width, height); err != nil {
		t.Fatalf("Failed to create test image 2: %v", err)
	}

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

	// Run tracker
	tracker, err := NewTracker(100)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	defer tracker.Close()

	ts1 := time.Now()
	if err := tracker.AddImage(img1, ts1); err != nil {
		t.Fatalf("Failed to add first image: %v", err)
	}
	initialTracks := len(tracker.GetTracks())

	ts2 := ts1.Add(1 * time.Second)
	if err := tracker.AddImage(img2, ts2); err != nil {
		t.Fatalf("Failed to add second image: %v", err)
	}

	// Assertions
	survivingTracks := len(tracker.GetTracks())
	survivalRate := float64(survivingTracks) / float64(initialTracks)

	if survivalRate < 0.5 {
		t.Errorf("Expected a survival rate of at least 50%%, but got %.2f%%", survivalRate*100)
	}

	t.Logf("Track survival rate: %.2f%%", survivalRate*100)
}

func TestTrackerWithFeatureLoss(t *testing.T) {
	width, height := 256, 256
	imgPath1 := "feature_loss1.png"
	imgPath2 := "feature_loss2.png"
	defer os.Remove(imgPath1)
	defer os.Remove(imgPath2)

	// A square near the edge
	polygon1 := []image.Point{
		{200, 100}, {240, 100}, {240, 140}, {200, 140},
	}
	if err := createTestImage(imgPath1, polygon1, width, height); err != nil {
		t.Fatalf("Failed to create test image 1: %v", err)
	}

	// The square moves partially out of frame
	// Some features should be lost
	polygon2 := []image.Point{
		{230, 100}, {270, 100}, {270, 140}, {230, 140},
	}
	if err := createTestImage(imgPath2, polygon2, width, height); err != nil {
		t.Fatalf("Failed to create test image 2: %v", err)
	}

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

	// Run tracker
	tracker, err := NewTracker(50) // Use fewer features to make loss more apparent
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	defer tracker.Close()

	ts1 := time.Now()
	if err := tracker.AddImage(img1, ts1); err != nil {
		t.Fatalf("Failed to add first image: %v", err)
	}
	initialTracks := len(tracker.GetTracks())
	if initialTracks == 0 {
		t.Fatal("No initial tracks found for feature loss test.")
	}
	t.Logf("Initial tracks: %d", initialTracks)

	ts2 := ts1.Add(1 * time.Second)
	if err := tracker.AddImage(img2, ts2); err != nil {
		t.Fatalf("Failed to add second image: %v", err)
	}

	// Assertions
	survivingTracks := len(tracker.GetTracks())
	t.Logf("Surviving tracks: %d", survivingTracks)

	if survivingTracks >= initialTracks {
		t.Errorf("Expected some features to be lost, but %d tracks survived out of %d initial tracks.", survivingTracks, initialTracks)
	}

	// Check that at least some tracks were lost
	if survivingTracks == initialTracks {
		t.Errorf("Expected feature loss, but all %d tracks survived.", initialTracks)
	}
	if survivingTracks == 0 {
		t.Errorf("All features were lost, expected some to survive.")
	}
}

func TestTrackerWithSquareToTriangle(t *testing.T) {
	width, height := 256, 256
	imgPath1 := "square_to_triangle1.png"
	imgPath2 := "square_to_triangle2.png"
	defer os.Remove(imgPath1)
	defer os.Remove(imgPath2)

	// A square
	square := []image.Point{
		{50, 150}, {150, 150}, {150, 50}, {50, 50},
	}
	if err := createTestImage(imgPath1, square, width, height); err != nil {
		t.Fatalf("Failed to create test image 1: %v", err)
	}

	// A triangle that shares the base of the square
	triangle := []image.Point{
		{50, 150}, {150, 150}, {100, 50},
	}
	if err := createTestImage(imgPath2, triangle, width, height); err != nil {
		t.Fatalf("Failed to create test image 2: %v", err)
	}

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

	// Run tracker
	tracker, err := NewTracker(100)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	defer tracker.Close()

	ts1 := time.Now()
	if err := tracker.AddImage(img1, ts1); err != nil {
		t.Fatalf("Failed to add first image: %v", err)
	}
	initialTracks := tracker.GetTracks()
	if len(initialTracks) == 0 {
		t.Fatal("No initial tracks found for square to triangle test.")
	}
	t.Logf("Initial tracks: %d", len(initialTracks))

	// Separate initial tracks into top and base features
	var topFeatures, baseFeatures []*Track
	for _, track := range initialTracks {
		if track.Points[0].Vec.Y < 100 {
			topFeatures = append(topFeatures, track)
		} else {
			baseFeatures = append(baseFeatures, track)
		}
	}

	ts2 := ts1.Add(1 * time.Second)
	if err := tracker.AddImage(img2, ts2); err != nil {
		t.Fatalf("Failed to add second image: %v", err)
	}

	// Assertions
	survivingTracks := tracker.GetTracks()
	t.Logf("Surviving tracks: %d", len(survivingTracks))

	if len(survivingTracks) == 0 {
		t.Errorf("All features were lost, but expected some to survive.")
	}

	// Check the displacement of the features
	for _, track := range survivingTracks {
		if len(track.Points) < 2 {
			continue
		}
		p1 := track.Points[0].Vec
		p2 := track.Points[1].Vec
		dy := p2.Y - p1.Y

		// Was this a top or base feature?
		isTopFeature := false
		for _, topTrack := range topFeatures {
			if topTrack.ID == track.ID {
				isTopFeature = true
				break
			}
		}

		if isTopFeature {
			// Top features should have a significant downward displacement
			// as they are tracked towards the new diagonal lines.
			if dy < 10.0 {
				t.Errorf("Top feature (ID %d) was expected to have significant downward displacement, but had dy=%.2f", track.ID, dy)
			}
		} else {
			// Base features should have near-zero Y displacement
			if math.Abs(float64(dy)) > 2.0 {
				t.Errorf("Base feature (ID %d) should have near-zero Y displacement, but had dy=%.2f", track.ID, dy)
			}
		}
	}
}

func TestTrackerWithRainfallData(t *testing.T) {
	// Find the rainfall_data directory
	var rainfallDir string
	possiblePaths := []string{"../rainfall_data", "rainfall_data"}
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			rainfallDir = path
			break
		}
	}

	if rainfallDir == "" {
		t.Skip("Skipping test: rainfall_data directory not found.")
	}

	// Get a sorted list of image files
	files, err := os.ReadDir(rainfallDir)
	if err != nil {
		t.Fatalf("Could not read rainfall_data directory: %v", err)
	}

	var imagePaths []string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".png" {
			imagePaths = append(imagePaths, filepath.Join(rainfallDir, file.Name()))
		}
	}
	sort.Strings(imagePaths)

	if len(imagePaths) < 6 {
		t.Fatalf("Expected at least 6 images in rainfall_data, but found %d", len(imagePaths))
	}

	// Use the first 6 images
	testImagePaths := imagePaths[:6]

	// Run tracker
	tracker, err := NewTracker(200)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	defer tracker.Close()

	var initialTracks int
	for i, imgPath := range testImagePaths {
		img, err := loadImageAsGrayscale(imgPath)
		if err != nil {
			t.Fatalf("Failed to load image %s: %v", imgPath, err)
		}
		defer img.Close()

		ts := time.Now().Add(time.Duration(i) * time.Minute) // Simulate timestamps
		if err := tracker.AddImage(img, ts); err != nil {
			t.Fatalf("Failed to add image %s: %v", imgPath, err)
		}

		if i == 0 {
			initialTracks = len(tracker.GetTracks())
			if initialTracks == 0 {
				t.Fatal("No initial tracks found in rainfall data.")
			}
			t.Logf("Initial tracks: %d", initialTracks)
		}
	}

	// Assertions
	survivingTracks := tracker.GetTracks()
	t.Logf("Surviving tracks after %d frames: %d", len(testImagePaths), len(survivingTracks))

	if len(survivingTracks) == 0 {
		t.Errorf("All features were lost during tracking of rainfall data.")
	}

	if float64(len(survivingTracks))/float64(initialTracks) < 0.1 {
		t.Errorf("Expected at least 10%% of tracks to survive, but only %.2f%% did.", float64(len(survivingTracks))/float64(initialTracks)*100)
	}

	// --- Displacement Diagnostics ---
	displacementsX := make([]float64, 0, len(survivingTracks))
	displacementsY := make([]float64, 0, len(survivingTracks))
	for _, track := range survivingTracks {
		if len(track.Points) == len(testImagePaths) {
			p1 := track.Points[0].Vec
			p_final := track.Points[len(track.Points)-1].Vec
			displacementsX = append(displacementsX, float64(p_final.X-p1.X))
			displacementsY = append(displacementsY, float64(p_final.Y-p1.Y))
		}
	}

	if len(displacementsX) > 0 {
		// Sort for median calculation
		sort.Float64s(displacementsX)
		sort.Float64s(displacementsY)

		// Mean
		var sumX, sumY float64
		for i := range displacementsX {
			sumX += displacementsX[i]
			sumY += displacementsY[i]
		}
		meanX := sumX / float64(len(displacementsX))
		meanY := sumY / float64(len(displacementsY))

		// Median
		medianX := displacementsX[len(displacementsX)/2]
		medianY := displacementsY[len(displacementsY)/2]

		// Standard Deviation
		var sdX, sdY float64
		for i := range displacementsX {
			sdX += math.Pow(displacementsX[i]-meanX, 2)
			sdY += math.Pow(displacementsY[i]-meanY, 2)
		}
		sdX = math.Sqrt(sdX / float64(len(displacementsX)))
		sdY = math.Sqrt(sdY / float64(len(displacementsY)))

		t.Logf("Displacement Stats (X): Mean=%.2f, Median=%.2f, StdDev=%.2f", meanX, medianX, sdX)
		t.Logf("Displacement Stats (Y): Mean=%.2f, Median=%.2f, StdDev=%.2f", meanY, medianY, sdY)
	}
}