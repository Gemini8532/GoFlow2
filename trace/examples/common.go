package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
)

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
