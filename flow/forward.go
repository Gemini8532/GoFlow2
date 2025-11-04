package flow

import (
	"image/color"
	"fmt"
	"image"
	"math"

	"gocv.io/x/gocv"
)

// ForwardTransform applies an optical flow map in forward to an image.
// It uses the flow vectors to move pixels from a source image to a new destination image.
func ForwardTransform(inputImagePath, flowMapPath string, factor float64) (image.Image, error) {
	// 1. Load the input image using OpenCV for proper format handling
	inputMat := gocv.IMRead(inputImagePath, gocv.IMReadColor)
	if inputMat.Empty() {
		return nil, fmt.Errorf("failed to read input image %s with gocv", inputImagePath)
	}
	defer inputMat.Close()

	// Get input image dimensions
	width, height := inputMat.Cols(), inputMat.Rows()

	// 2. Load the flow map using OpenCV
	flowMat := gocv.IMRead(flowMapPath, gocv.IMReadColor)
	if flowMat.Empty() {
		return nil, fmt.Errorf("failed to read flow map %s with gocv", flowMapPath)
	}
	defer flowMat.Close()

	flowWidth := flowMat.Cols()
	flowHeight := flowMat.Rows()

	// 3. If flow map resolution differs from input image, resize the flow map to match the input image resolution
	var processedFlowMat gocv.Mat
	if flowWidth != width || flowHeight != height {
		processedFlowMat = gocv.NewMat()
		defer processedFlowMat.Close()
		gocv.Resize(flowMat, &processedFlowMat, image.Pt(width, height), 0, 0, gocv.InterpolationCubic)
	} else {
		processedFlowMat = flowMat
	}

	// 4. Create output image
	outputImg := image.NewRGBA(image.Rect(0, 0, width, height))

	// 5. Iterate through each pixel and apply the flow
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Get the flow vector from the processed flow map
			bgr := processedFlowMat.GetVecbAt(y, x)
			// The flow is encoded in R and G channels (B is unused)
			r8 := uint8(bgr[2])  // Red channel
			g8 := uint8(bgr[1])  // Green channel
			
			// Reverse the encoding formula to get the displacement vector
			dx := (float64(r8) - FlowMidLevel) / FlowScaleFactor
			dy := (float64(g8) - FlowMidLevel) / FlowScaleFactor

			// Calculate the source coordinates from where to pull the pixel
			// We subtract the scaled displacement vector
			srcX := float64(x) - (dx * factor)
			srcY := float64(y) - (dy * factor)

			// Apply nearest-neighbor interpolation (rounding to nearest integer pixel)
			finalSrcX := int(math.Round(srcX))
			finalSrcY := int(math.Round(srcY))

			// Boundary check: Clamp the source coordinates to be within the image bounds
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

			// Get the color from the input Mat
			srcBGR := inputMat.GetVecbAt(finalSrcY, finalSrcX)
			srcR, srcG, srcB := srcBGR[2], srcBGR[1], srcBGR[0]  // BGR to RGB conversion

			// Check if the source pixel has zero value (black), which should be transparent
			// If all RGB values are 0, this represents a transparent pixel
			if srcR == 0 && srcG == 0 && srcB == 0 {
				outputImg.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 0})
			} else {
				outputImg.Set(x, y, color.RGBA{R: srcR, G: srcG, B: srcB, A: 255})
			}
		}
	}

	return outputImg, nil
}
