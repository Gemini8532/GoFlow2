package trace

import (
	"math"
	"testing"
)

func TestProjectAngularSearch(t *testing.T) {
	// Create a test image with a diagonal line of high values
	imageSize := 10
	image := make([][]float64, imageSize)
	for i := range image {
		image[i] = make([]float64, imageSize)
	}

	// Add a diagonal line of 100s (5 elements long)
	diagonalLength := 5
	for i := 0; i < diagonalLength; i++ {
		image[i][i] = 100.0
	}

	// Common parameters for all tests
	origin := Point{X: 0, Y: 0}
	fieldOfViewAngleRadians := math.Pi / 12 // 15 degrees (narrow field of view)

	// Test case 1: Search along the diagonal direction (should find all diagonal values if distance is sufficient)
	t.Run("DiagonalSearch", func(t *testing.T) {
		direction := Point{X: 1, Y: 1} // Diagonal direction
		distance := 7.0 // Sufficient to cover the full diagonal (sqrt(2)*5 ≈ 7.07)

		projection, _, err := ProjectAngularSearch(image, origin, direction, fieldOfViewAngleRadians, distance)
		if err != nil {
			t.Fatalf("ProjectAngularSearch returned error: %v", err)
		}

		if len(projection) == 0 {
			t.Error("Expected non-empty projection")
		}

		// Count high values (should be at least diagonalLength number of 100.0 values)
		highValueCount := 0
		for _, val := range projection {
			if val >= 100.0 {
				highValueCount++
			}
		}
		
		// We expect at least as many high values as there are diagonal elements
		if highValueCount < diagonalLength {
			t.Errorf("Expected at least %d high values in diagonal search, got %d", diagonalLength, highValueCount)
		}
	})

	// Test case 2: Search perpendicular to the diagonal (should find at most one peak if aligned properly)
	t.Run("PerpendicularSearch", func(t *testing.T) {
		// Perpendicular to diagonal (1,1) is (-1,1) or (1,-1)
		direction := Point{X: -1, Y: 1} // Perpendicular to diagonal
		distance := 10.0

		projection, _, err := ProjectAngularSearch(image, Point{X: 2, Y: 0}, direction, fieldOfViewAngleRadians, distance) // Shift origin to intersect diagonal
		if err != nil {
			t.Fatalf("ProjectAngularSearch returned error: %v", err)
		}

		// Count high values - should find the diagonal values that intersect with perpendicular search
		highValueCount := 0
		for _, val := range projection {
			if val >= 100.0 {
				highValueCount++
			}
		}
		
		// Perpendicular search through diagonal should find at least one high value
		if highValueCount == 0 {
			t.Error("Expected to find at least one high value in perpendicular search intersecting diagonal")
		}
	})

	// Test case 3: Search offset from the diagonal (should find no values or very few)
	t.Run("OffsetSearch", func(t *testing.T) {
		// Search in a direction away from the diagonal region
		// Start from a position that clearly avoids the diagonal
		originOffset := Point{X: 7, Y: 2} // Start from a position far from the diagonal
		direction := Point{X: 0, Y: 1} // Straight down
		
		distance := 5.0

		projection, _, err := ProjectAngularSearch(image, originOffset, direction, fieldOfViewAngleRadians, distance)
		if err != nil {
			t.Fatalf("ProjectAngularSearch returned error: %v", err)
		}

		// Count high values - should find none since we're searching far from the diagonal
		highValueCount := 0
		for _, val := range projection {
			if val >= 100.0 {
				highValueCount++
			}
		}
		
		// Downward search from offset position should not intersect diagonal values
		if highValueCount > 0 {
			t.Errorf("Expected no high values in offset direction, but found %d", highValueCount)
		}
	})

	// Test case 4: Search with insufficient distance (should find fewer diagonal values)
	t.Run("ShortDistanceSearch", func(t *testing.T) {
		direction := Point{X: 1, Y: 1} // Diagonal direction
		shortDistance := 2.0 // Shorter distance, should hit fewer diagonal points

		projection, _, err := ProjectAngularSearch(image, origin, direction, fieldOfViewAngleRadians, shortDistance)
		if err != nil {
			t.Fatalf("ProjectAngularSearch returned error: %v", err)
		}

		highValueCount := 0
		for _, val := range projection {
			if val >= 100.0 {
				highValueCount++
			}
		}
		
		// With shorter distance, we might find fewer values
		if highValueCount > diagonalLength {
			t.Errorf("Expected fewer high values with short distance, got %d", highValueCount)
		}
	})

	// Test case 5: Invalid inputs
	t.Run("InvalidInputs", func(t *testing.T) {
		_, _, err := ProjectAngularSearch(image, origin, Point{X: 0, Y: 0}, fieldOfViewAngleRadians, 5.0)
		if err == nil {
			t.Error("Expected error for zero direction vector")
		}

		_, _, err = ProjectAngularSearch(image, origin, Point{X: 1, Y: 1}, -1.0, 5.0)
		if err == nil {
			t.Error("Expected error for negative field of view angle")
		}

		_, _, err = ProjectAngularSearch(image, origin, Point{X: 1, Y: 1}, math.Pi*2, 5.0)
		if err == nil {
			t.Error("Expected error for field of view angle >= Pi")
		}

		_, _, err = ProjectAngularSearch(image, origin, Point{X: 1, Y: 1}, fieldOfViewAngleRadians, -1.0)
		if err == nil {
			t.Error("Expected error for negative distance")
		}
	})
}

func TestProjectTriangleMax(t *testing.T) {
	// Create a simple test image (5x5)
	image := make([][]float64, 5)
	for i := range image {
		image[i] = make([]float64, 5)
	}

	// Add a single high value point
	image[2][2] = 200.0

	// Create a triangle that encompasses the high value point
	tri := Triangle{
		V1: Point{X: 1, Y: 1},
		V2: Point{X: 3, Y: 1},
		V3: Point{X: 2, Y: 3},
	}
	
	// Use a direction vector pointing right (X direction)
	dirUnitVec := Point{X: 1, Y: 0}

	projection := ProjectTriangleMax(image, tri, dirUnitVec)

	// Check that we got a valid projection
	if len(projection) == 0 {
		t.Error("Expected non-empty projection")
	}

	// The projection should contain the high value somewhere
	foundHighValue := false
	for _, val := range projection {
		if val >= 200.0 {
			foundHighValue = true
			break
		}
	}
	if !foundHighValue {
		t.Error("Expected to find high values (200.0) in the projection")
	}
}

func TestNormalize(t *testing.T) {
	// Test normalizing a simple vector
	vec := Point{X: 3, Y: 4}
	unitVec, mag := normalize(vec)

	expectedMag := 5.0 // sqrt(3^2 + 4^2)
	if math.Abs(mag-expectedMag) > 1e-10 {
		t.Errorf("Expected magnitude %f, got %f", expectedMag, mag)
	}

	expectedUnitVec := Point{X: 0.6, Y: 0.8} // 3/5, 4/5
	if math.Abs(unitVec.X-expectedUnitVec.X) > 1e-10 || math.Abs(unitVec.Y-expectedUnitVec.Y) > 1e-10 {
		t.Errorf("Expected unit vector %v, got %v", expectedUnitVec, unitVec)
	}

	// Test zero vector
	zeroVec := Point{X: 0, Y: 0}
	unitVec, mag = normalize(zeroVec)
	if mag != 0 || unitVec.X != 0 || unitVec.Y != 0 {
		t.Error("Expected zero vector to return zero magnitude and zero unit vector")
	}
}

func TestDotProduct(t *testing.T) {
	vec1 := Point{X: 1, Y: 2}
	vec2 := Point{X: 3, Y: 4}
	result := dot(vec1, vec2)

	expected := float64(1*3 + 2*4) // 3 + 8 = 11
	if result != expected {
		t.Errorf("Expected dot product %f, got %f", expected, result)
	}
}

// TestPaletteImageSupport tests the trace module with paletted image data
func TestPaletteImageSupport(t *testing.T) {
	// Create a mock paletted image represented as indices
	// This simulates a rainfall data image with integer index values representing rainfall amounts
	imageSize := 20
	paletteImage := make([][]float64, imageSize)
	for i := range paletteImage {
		paletteImage[i] = make([]float64, imageSize)
	}

	// Create some "rainfall" patterns using palette indices (0-8)
	// Index 0 = no rain, 1-8 = increasing rainfall intensity
	for i := 5; i < 10; i++ {
		paletteImage[5][i] = 5.0  // Level 5 rainfall along row 5
		paletteImage[10][i] = 3.0 // Level 3 rainfall along row 10
	}

	// Add a diagonal line of intensity 7
	for i := 0; i < 15; i++ {
		if i < imageSize {
			paletteImage[i][i] = 7.0
		}
	}

	// Test 1: Diagonal search that should follow the main diagonal
	t.Run("DiagonalSearchOnPaletteImage", func(t *testing.T) {
		origin := Point{X: 0, Y: 0}
		direction := Point{X: 1, Y: 1} // Diagonal direction
		fieldOfViewAngleRadians := math.Pi / 18 // 10 degrees
		distance := 20.0

		projection, _, err := ProjectAngularSearch(paletteImage, origin, direction, fieldOfViewAngleRadians, distance)
		if err != nil {
			t.Fatalf("ProjectAngularSearch returned error: %v", err)
		}

		// The projection should contain the maximum values along the diagonal
		// Since we set diagonal values to 7.0, we should see these in the projection
		foundHighValue := false
		for _, val := range projection {
			if val == 7.0 {
				foundHighValue = true
				break
			}
		}
		
		if !foundHighValue {
			t.Error("Expected to find high values (7.0) from diagonal in projection")
		}
	})

	// Test 2: Horizontal search for the horizontal rainfall bands
	t.Run("HorizontalSearchOnPaletteImage", func(t *testing.T) {
		origin := Point{X: 5, Y: 5}  // Starting at row 5 where we have level 5 rainfall
		direction := Point{X: 1, Y: 0} // Horizontal direction
		fieldOfViewAngleRadians := math.Pi / 36 // 5 degrees
		distance := 10.0

		projection, _, err := ProjectAngularSearch(paletteImage, origin, direction, fieldOfViewAngleRadians, distance)
		if err != nil {
			t.Fatalf("ProjectAngularSearch returned error: %v", err)
		}

		// Should find level 5 values along the horizontal band
		foundLevel5 := false
		for _, val := range projection {
			if val == 5.0 {
				foundLevel5 = true
				break
			}
		}
		
		if !foundLevel5 {
			t.Error("Expected to find level 5 values in horizontal search")
		}
	})

	// Test 3: Verify that original index values are preserved in the projection
	t.Run("OriginalValuesPreserved", func(t *testing.T) {
		origin := Point{X: 0, Y: 10}  // Near row 10 where we have level 3 rainfall
		direction := Point{X: 1, Y: 0} // Horizontal direction
		fieldOfViewAngleRadians := math.Pi / 12 // 15 degrees
		distance := 15.0

		projection, _, err := ProjectAngularSearch(paletteImage, origin, direction, fieldOfViewAngleRadians, distance)
		if err != nil {
			t.Fatalf("ProjectAngularSearch returned error: %v", err)
		}

		// Check that we find the original palette index values (like 3.0) in the projection
		foundLevel3 := false
		for _, val := range projection {
			if val == 3.0 {
				foundLevel3 = true
				break
			}
		}
		
		if !foundLevel3 {
			t.Error("Expected to preserve original palette index values in projection")
		}
	})
}

// TestRainfallDataIntegration tests the trace module with actual rainfall data handling
func TestRainfallDataIntegration(t *testing.T) {
	// Create a simplified rainfall pattern to test the conceptual approach
	// In a real scenario, we would load actual rainfall PNG files using our palette loader
	imageSize := 10
	rainfallImage := make([][]float64, imageSize)
	for i := range rainfallImage {
		rainfallImage[i] = make([]float64, imageSize)
	}

	// Create a rainfall storm pattern: higher values in the center
	for y := 3; y < 7; y++ {
		for x := 3; x < 7; x++ {
			rainfallImage[y][x] = 6.0 // High rainfall in center
		}
	}

	// Add some border rain
	for x := 0; x < imageSize; x++ {
		rainfallImage[0][x] = 2.0
		rainfallImage[imageSize-1][x] = 2.0
	}
	for y := 0; y < imageSize; y++ {
		rainfallImage[y][0] = 2.0
		rainfallImage[y][imageSize-1] = 2.0
	}

	// Test searching for the high-rainfall area
	t.Run("RainfallStormSearch", func(t *testing.T) {
		origin := Point{X: 0, Y: 5}  // From left side
		direction := Point{X: 1, Y: 0} // Moving right
		fieldOfViewAngleRadians := math.Pi / 12 // 15 degrees
		distance := 15.0

		projection, _, err := ProjectAngularSearch(rainfallImage, origin, direction, fieldOfViewAngleRadians, distance)
		if err != nil {
			t.Fatalf("ProjectAngularSearch returned error: %v", err)
		}

		// Should find the high-rainfall values (6.0) in the center
		foundHighRainfall := false
		for _, val := range projection {
			if val == 6.0 {
				foundHighRainfall = true
				break
			}
		}
		
		if !foundHighRainfall {
			t.Error("Expected to find high rainfall values (6.0) in storm search")
		}
	})
}

// TestPalettedImageCoordinateSystem tests trace with non-zero regions in the image
func TestPalettedImageCoordinateSystem(t *testing.T) {
	// Create an image with non-zero values at specific locations to test coordinate system
	imageSize := 20
	testImage := make([][]float64, imageSize)
	for i := range testImage {
		testImage[i] = make([]float64, imageSize)
	}

	// Add values in different regions of the image
	testImage[5][10] = 3.0  // Value at row 5, col 10
	testImage[5][11] = 3.0
	testImage[5][12] = 3.0
	testImage[10][5] = 7.0  // Value at row 10, col 5
	testImage[11][5] = 7.0

	// Test searching in the region with value 3
	t.Run("SearchRegionWithValues", func(t *testing.T) {
		origin := Point{X: 9, Y: 5}   // Start near the horizontal line of value 3
		direction := Point{X: 1, Y: 0} // Move horizontally
		fieldOfViewAngleRadians := math.Pi / 18 // Narrow angle
		distance := 5.0

		projection, _, err := ProjectAngularSearch(testImage, origin, direction, fieldOfViewAngleRadians, distance)
		if err != nil {
			t.Fatalf("ProjectAngularSearch returned error: %v", err)
		}

		// Should find the value 3 in the projection
		foundValue3 := false
		for _, val := range projection {
			if val == 3.0 {
				foundValue3 = true
				break
			}
		}
		
		if !foundValue3 {
			t.Error("Expected to find value 3.0 in projection")
		}
	})

	// Test searching in the region with value 7
	t.Run("SearchVerticalRegion", func(t *testing.T) {
		origin := Point{X: 5, Y: 9}   // Start near the vertical line of value 7
		direction := Point{X: 0, Y: 1} // Move vertically
		fieldOfViewAngleRadians := math.Pi / 18 // Narrow angle
		distance := 5.0

		projection, _, err := ProjectAngularSearch(testImage, origin, direction, fieldOfViewAngleRadians, distance)
		if err != nil {
			t.Fatalf("ProjectAngularSearch returned error: %v", err)
		}

		// Should find the value 7 in the projection
		foundValue7 := false
		for _, val := range projection {
			if val == 7.0 {
				foundValue7 = true
				break
			}
		}
		
		if !foundValue7 {
			t.Error("Expected to find value 7.0 in projection")
		}
	})
}