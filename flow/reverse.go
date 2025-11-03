package flow

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
)

// ReverseTransform applies an optical flow map in reverse to an image.
// It uses the flow vectors to move pixels from a source image to a new destination image.
func ReverseTransform(inputImagePath, flowMapPath string, factor float64) (image.Image, error) {
	// 1. Load the input image
	inputFile, err := os.Open(inputImagePath)
	if err != nil {
		return nil, fmt.Errorf("could not open input image %s: %w", inputImagePath, err)
	}
	defer inputFile.Close()

	inputImg, err := png.Decode(inputFile)
	if err != nil {
		return nil, fmt.Errorf("could not decode input image %s: %w", inputImagePath, err)
	}
	bounds := inputImg.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// 2. Load the flow map
	flowFile, err := os.Open(flowMapPath)
	if err != nil {
		return nil, fmt.Errorf("could not open flow map %s: %w", flowMapPath, err)
	}
	defer flowFile.Close()

	flowImg, err := png.Decode(flowFile)
	if err != nil {
		return nil, fmt.Errorf("could not decode flow map %s: %w", flowMapPath, err)
	}

	// Verify dimensions match
	if flowImg.Bounds().Dx() != width || flowImg.Bounds().Dy() != height {
		return nil, fmt.Errorf("input image and flow map dimensions do not match")
	}

	// 3. Create a new output image
	outputImg := image.NewRGBA(bounds)

	// 4. Iterate through each pixel of the DESTINATION image
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// 5. Decode the flow vector at this position
			r, g, _, _ := flowImg.At(x, y).RGBA()
			// Convert from uint16 (0-65535) to uint8 (0-255) range
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)

			// Reverse the encoding formula to get the displacement vector
			dx := (float64(r8) - FlowMidLevel) / FlowScaleFactor
			dy := (float64(g8) - FlowMidLevel) / FlowScaleFactor

			// 6. Calculate the source coordinates from where to pull the pixel
			// We subtract the scaled displacement vector
			srcX := float64(x) - (dx * factor)
			srcY := float64(y) - (dy * factor)

			// 7. Apply nearest-neighbor interpolation
			// This means rounding to the nearest integer pixel
			finalSrcX := int(math.Round(srcX))
			finalSrcY := int(math.Round(srcY))

			// 8. Boundary check: Clamp the source coordinates to be within the image bounds
			if finalSrcX < 0 {
				finalSrcX = 0
			}
			if finalSrcX >= width {
				finalSrcX = width - 1
			}
			if finalSrcY < 0 {
				finalSrcY = 0
			}
			if finalSrcY >= height {
				finalSrcY = height - 1
			}

			// 9. Get the color from the source image and set it in the output image
			srcColor := inputImg.At(finalSrcX, finalSrcY)
			outputImg.Set(x, y, srcColor)
		}
	}

	return outputImg, nil
}
