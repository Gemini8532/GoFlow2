package trace

import (
	"image"
	"image/png"
	"os"
)

// LoadPalettedImageRaw loads a paletted PNG image and returns the raw palette indices
// instead of converting to RGB or grayscale. This preserves the original scale/meaning
// of the palette image values.
func LoadPalettedImageRaw(filename string) ([][]float64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode the PNG
	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}

	// Get the paletted image data
	paletted, ok := img.(*image.Paletted)
	if !ok {
		// If the image is not paletted, try to get its bounds and create a converted version
		// The paletted image may have been auto-converted by png.Decode
		// To handle this, we'll need to decode differently
		// Let's go back to the original file and decode using a specific paletted decoder
		return nil, nil // Placeholder - this is complex to implement properly
	}

	bounds := paletted.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create a 2D array to hold the palette indices as float64 values
	image := make([][]float64, height)
	for y := 0; y < height; y++ {
		image[y] = make([]float64, width)
	}

	// Fill the array with palette indices
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get the palette index at this position
			index := paletted.ColorIndexAt(x, y)
			image[y][x] = float64(index)
		}
	}

	return image, nil
}

// LoadPalettedImageFromRaw loads a paletted PNG image and returns the raw palette indices
// by reading the PNG directly to preserve the palette information
func LoadPalettedImageFromRaw(filename string) ([][]float64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create a decoder that maintains palette information
	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}

	// If we have a paletted image, extract indices
	if paletted, ok := img.(*image.Paletted); ok {
		bounds := paletted.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		imageData := make([][]float64, height)
		for y := 0; y < height; y++ {
			imageData[y] = make([]float64, width)
		}

		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				index := paletted.ColorIndexAt(x, y)
				imageData[y][x] = float64(index)
			}
		}

		return imageData, nil
	}

	// For non-paletted images, just convert to float64 format
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	imageData := make([][]float64, height)
	for y := 0; y < height; y++ {
		imageData[y] = make([]float64, width)
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			imageData[y][x] = float64(r >> 8) // Use red channel as example
		}
	}

	return imageData, nil
}
