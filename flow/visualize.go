package flow

import (
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

func CreateVisualization(initialPoints gocv.Mat, currentPoints gocv.Mat, scaledHeight int, scaledWidth int) gocv.Mat {
	flowMap := gocv.NewMatWithSize(scaledHeight, scaledWidth, gocv.MatTypeCV8UC3)
	flowMap.SetTo(gocv.NewScalar(0, 0, 0, 0)) // Black background

	// Define colors for drawing
	lineColor := color.RGBA{R: 0, G: 255, B: 0, A: 255} // Green lines for vectors
	dotColor := color.RGBA{R: 255, G: 0, B: 0, A: 25}   // Blue dots for endpoints

	for i := 0; i < initialPoints.Rows(); i++ {
		// Get original point
		p0x := initialPoints.GetFloatAt(i, 0)
		p0y := initialPoints.GetFloatAt(i, 1)
		pt0 := image.Pt(int(p0x), int(p0y))

		// Get new (tracked) point
		p1x := currentPoints.GetFloatAt(i, 0)
		p1y := currentPoints.GetFloatAt(i, 1)

		pt1 := image.Pt(int(p1x), int(p1y))

		// Draw the flow vector as a line
		gocv.Line(&flowMap, pt0, pt1, lineColor, 1)
		// Draw the endpoint as a small circle
		gocv.Circle(&flowMap, pt1, 2, dotColor, -1)
	}
	return flowMap
}
