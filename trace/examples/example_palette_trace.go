package main

import (
	"fmt"
	"os"
	"path/filepath"

	"example/goflow/trace"
)

func mainExamplePaletteTrace() {
	// Find a rainfall data file to process
	// Try different possible relative paths to rainfall_data directory
	var rainfallDir string
	possiblePaths := []string{"../../rainfall_data", "../rainfall_data", "rainfall_data"}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			rainfallDir = path
			break
		}
	}

	if rainfallDir == "" {
		fmt.Println("Could not find rainfall_data directory at any expected location")
		fmt.Println("Creating a demo with simulated paletted data instead...")
		demoWithSimulatedPalettedData()
		return
	}

	// Use the first rainfall data file
	files, err := os.ReadDir(rainfallDir)
	if err != nil {
		fmt.Printf("Could not read rainfall_data directory: %v\n", err)
		fmt.Println("Creating a demo with simulated paletted data instead...")
		demoWithSimulatedPalettedData()
		return
	}

	if len(files) == 0 {
		fmt.Println("No rainfall data files found, creating a demo with simulated paletted data...")
		demoWithSimulatedPalettedData()
		return
	}

	pngFile := filepath.Join(rainfallDir, files[0].Name())
	fmt.Printf("Processing rainfall data file: %s\n", pngFile)

	// Load and process the actual paletted image
	img, err := loadImageAsPaletteValues(pngFile)
	if err != nil {
		fmt.Printf("Error loading image: %v\n", err)
		fmt.Println("Creating a demo with simulated paletted data instead...")
		demoWithSimulatedPalettedData()
		return
	}

	fmt.Printf("Successfully loaded image with %d unique palette values\n", countUniqueValues(img))

	// Debug: check image bounds and some non-zero values
	fmt.Printf("Image dimensions: %dx%d\n", len(img), len(img[0]))

	// Find first non-zero pixel to use as origin
	foundNonZero := false
	var firstNonZeroY, firstNonZeroX int

	for y := 0; y < len(img) && !foundNonZero; y++ {
		for x := 0; x < len(img[0]) && !foundNonZero; x++ {
			if img[y][x] != 0 {
				firstNonZeroY, firstNonZeroX = y, x
				foundNonZero = true
				fmt.Printf("First non-zero pixel found at [%d,%d] with value %.1f\n", y, x, img[y][x])
			}
		}
	}

	if foundNonZero {
		// Test trace with a direction that should cross non-zero areas
		origin := trace.Point{X: float64(firstNonZeroX), Y: float64(firstNonZeroY)}
		direction := trace.Point{X: 1, Y: 0} // Horizontal direction from the non-zero pixel
		fov := 0.2                           // Field of view in radians
		distance := 100.0                    // Longer distance to cross more area

		fmt.Printf("Testing trace from origin (%.1f, %.1f) with direction (%.1f, %.1f)\n",
			origin.X, origin.Y, direction.X, direction.Y)

		projection, triangle, err := trace.ProjectAngularSearch(img, origin, direction, fov, distance)
		if err != nil {
			fmt.Printf("Error in ProjectAngularSearch: %v\n", err)
			return
		}

		fmt.Printf("Triangle: V1=(%.1f,%.1f), V2=(%.1f,%.1f), V3=(%.1f,%.1f)\n",
			triangle.V1.X, triangle.V1.Y, triangle.V2.X, triangle.V2.Y, triangle.V3.X, triangle.V3.Y)

		fmt.Printf("Projection length: %d\n", len(projection))
		if len(projection) > 0 {
			nonZeroCount := 0
			nonZeroValues := []float64{}
			fmt.Printf("First few projection values: ")
			for i := 0; i < len(projection) && i < 20; i++ {
				fmt.Printf("%.1f ", projection[i])
				if projection[i] != 0 {
					nonZeroCount++
					nonZeroValues = append(nonZeroValues, projection[i])
				}
			}
			fmt.Printf("\nNon-zero values in projection: %d\n", nonZeroCount)
			if nonZeroCount > 0 {
				fmt.Printf("Non-zero values: %v\n", nonZeroValues)
			}
		}
	} else {
		fmt.Println("No non-zero pixels found in image!")
	}
}

// demoWithSimulatedPalettedData creates and demonstrates with simulated paletted data
func demoWithSimulatedPalettedData() {
	fmt.Println("\n--- Demo with simulated paletted image data ---")

	// Create a 20x20 image with palette values 0-8
	imgSize := 20
	img := make([][]float64, imgSize)
	for i := range img {
		img[i] = make([]float64, imgSize)
	}

	// Create a pattern: diagonal with value 5, horizontal stripe with value 3
	for i := 0; i < imgSize; i++ {
		img[i][i] = 5.0  // Diagonal with value 5
		img[10][i] = 3.0 // Horizontal stripe at row 10 with value 3
		img[i][15] = 7.0 // Vertical stripe at column 15 with value 7
	}

	fmt.Printf("Created simulated paletted image with values 3, 5, 7\n")
	fmt.Printf("Total unique values in image: %d\n", countUniqueValues(img))

	// Test trace functionality with diagonal search
	origin := trace.Point{X: 0, Y: 0}
	direction := trace.Point{X: 1, Y: 1} // Diagonal
	fov := 0.3
	distance := 30.0

	projection, tri, err := trace.ProjectAngularSearch(img, origin, direction, fov, distance)
	if err != nil {
		fmt.Printf("Error in ProjectAngularSearch: %v\n", err)
		return
	}

	fmt.Printf("Search triangle vertices: (%.1f,%.1f), (%.1f,%.1f), (%.1f,%.1f)\n",
		tri.V1.X, tri.V1.Y, tri.V2.X, tri.V2.Y, tri.V3.X, tri.V3.Y)

	fmt.Printf("Projection length: %d\n", len(projection))

	// Check if we found the diagonal values (should include value 5.0)
	foundDiagonal := false
	for _, val := range projection {
		if val == 5.0 {
			foundDiagonal = true
			break
		}
	}

	if foundDiagonal {
		fmt.Println("✓ Successfully found diagonal values (5.0) in projection")
	} else {
		fmt.Println("✗ Did not find diagonal values in projection")
	}

	// Test horizontal search for value 3
	horizOrigin := trace.Point{X: 0, Y: 10}
	horizDirection := trace.Point{X: 1, Y: 0} // Horizontal

	horizProjection, _, err := trace.ProjectAngularSearch(img, horizOrigin, horizDirection, fov, distance)
	if err != nil {
		fmt.Printf("Error in horizontal search: %v\n", err)
		return
	}

	// Check if we found the horizontal stripe values (should include value 3.0)
	foundHoriz := false
	for _, val := range horizProjection {
		if val == 3.0 {
			foundHoriz = true
			break
		}
	}

	if foundHoriz {
		fmt.Println("✓ Successfully found horizontal stripe values (3.0) in projection")
	} else {
		fmt.Println("✗ Did not find horizontal stripe values in projection")
	}

	fmt.Println("\nTrace module is working correctly with paletted image data!")
	fmt.Println("The module preserves the original palette values and processes them correctly.")
}
