package main

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"example/goflow/trace"
)

func main() {
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
		return
	}

	// Use the first rainfall data file
	files, err := os.ReadDir(rainfallDir)
	if err != nil {
		fmt.Printf("Could not read rainfall_data directory: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No rainfall data files found")
		return
	}

	pngFile := filepath.Join(rainfallDir, files[0].Name())
	fmt.Printf("Processing rainfall data file: %s\n", pngFile)

	// Load and process the actual paletted image
	img, err := loadImageAsPaletteValues(pngFile)
	if err != nil {
		fmt.Printf("Error loading image: %v\n", err)
		return
	}

	fmt.Printf("Successfully loaded image with %d unique palette values\n", countUniqueValues(img))
	
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
		// Test trace with different parameters to understand the projection sizing
		origin := trace.Point{X: float64(firstNonZeroX), Y: float64(firstNonZeroY)}
		direction := trace.Point{X: 1, Y: 0} // Horizontal direction from the non-zero pixel
		
		// Test with various distances to see the projection length relationship
		for _, distance := range []float64{10.0, 25.0, 50.0, 100.0} {
			fmt.Printf("\n--- Testing with distance: %.1f ---\n", distance)
			
			projection, triangle, err := trace.ProjectAngularSearch(img, origin, direction, 0.2, distance)
			if err != nil {
				fmt.Printf("Error in ProjectAngularSearch: %v\n", err)
				continue
			}

			fmt.Printf("Triangle: V1=(%.1f,%.1f), V2=(%.1f,%.1f), V3=(%.1f,%.1f)\n",
				triangle.V1.X, triangle.V1.Y, triangle.V2.X, triangle.V2.Y, triangle.V3.X, triangle.V3.Y)
			
			// Calculate the theoretical range of the triangle in the direction
			p1 := dot(triangle.V1, direction)
			p2 := dot(triangle.V2, direction)
			p3 := dot(triangle.V3, direction)
			
			minVal := math.Min(p1, math.Min(p2, p3))
			maxVal := math.Max(p1, math.Max(p2, p3))
			
			fmt.Printf("Theoretical projection range: [%.2f, %.2f], span = %.2f\n", minVal, maxVal, maxVal-minVal)
			fmt.Printf("Actual projection length: %d\n", len(projection))
			
			// Count non-zero values in the projection
			nonZeroCount := 0
			for _, val := range projection {
				if val != 0 {
					nonZeroCount++
				}
			}
			fmt.Printf("Non-zero values in projection: %d\n", nonZeroCount)
		}
	}
}

// dot calculates the dot product of two points (as vectors)
func dot(v1, v2 trace.Point) float64 {
	return v1.X*v2.X + v1.Y*v2.Y
}

// loadImageAsPaletteValues loads a PNG and returns the palette indices as float64 values
func loadImageAsPaletteValues(filename string) ([][]float64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode image
	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}

	// If it's a paletted image, extract the palette indices
	if paletted, ok := img.(*image.Paletted); ok {
		fmt.Printf("Image is paletted with %d palette entries\n", len(paletted.Palette))
		bounds := paletted.Bounds()
		height := bounds.Dy()
		width := bounds.Dx()
		
		result := make([][]float64, height)
		for y := 0; y < height; y++ {
			result[y] = make([]float64, width)
		}
		
		// Count unique palette indices in the image
		uniqueIndices := make(map[uint8]int)
		
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				// Get the palette index at this position
				index := paletted.ColorIndexAt(x, y)
				result[y-bounds.Min.Y][x-bounds.Min.X] = float64(index)
				uniqueIndices[index]++
			}
		}
		
		fmt.Printf("Unique palette indices in image: ")
		for idx := range uniqueIndices {
			fmt.Printf("%d ", idx)
		}
		fmt.Printf("\n")
		
		return result, nil
	} else {
		// For non-paletted images, just extract grayscale values
		fmt.Printf("Image is not paletted, it's type: %T\n", img)
		bounds := img.Bounds()
		height := bounds.Dy()
		width := bounds.Dx()
		
		result := make([][]float64, height)
		for y := 0; y < height; y++ {
			result[y] = make([]float64, width)
		}
		
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				// Convert to grayscale and use as intensity
				r, g, b, _ := img.At(x, y).RGBA()
				// Use luminance formula to convert to grayscale
				gray := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
				result[y-bounds.Min.Y][x-bounds.Min.X] = gray
			}
		}
		
		return result, nil
	}
}

// countUniqueValues counts the number of unique values in the image
func countUniqueValues(img [][]float64) int {
	unique := make(map[float64]bool)
	for _, row := range img {
		for _, val := range row {
			unique[val] = true
		}
	}
	return len(unique)
}