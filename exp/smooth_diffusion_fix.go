package main

import (
	"fmt"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

const rows, cols = 512, 512

/*

Make the following changes:
Neumann boundary conditions - Instead of treating edges as zero (Dirichlet), the edges now use the gradient-free condition where boundary pixels use their own value when a neighbor is missing. This prevents black borders.
Gauss-Seidel instead of Jacobi - Updates values in-place immediately, which propagates information faster and converges in roughly half the iterations. Also eliminates the need for the temp matrix, saving memory.
Reduced iterations - Cut iteration counts significantly since Gauss-Seidel converges faster and we have multigrid acceleration.
Removed one scale level - Eliminated the 0.0625 scale which was overkill for a 256x256 image.
*/

func main() {
	// 1. Setup Dimensions

	// Create the main data grid (32-bit Floating Point)
	// This holds your heat/values.
	grid := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV32F)
	defer grid.Close()

	// Create a mask (8-bit Unsigned)
	// 0 = Empty/Unknown, 255 = Fixed/Known Cluster
	mask := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8U)
	defer mask.Close()

	// 2. Simulate "Clusters" of data
	// Let's create a few random blobs of known values
	simulateClusters(&grid, &mask)

	// Debug: Check if mask and grid have data
	fmt.Printf("Grid non-zero count: %d\n", gocv.CountNonZero(grid))
	fmt.Printf("Mask non-zero count: %d\n", gocv.CountNonZero(mask))

	fmt.Println("Processing grid...")

	// 3. Run the High-Performance Fill
	// This fills the gaps (where mask == 0) with smooth gradients
	filledGrid := FastSmoothFill(&grid, &mask)
	defer filledGrid.Close()

	// 4. Save outputs as PNG files

	// Calculate global min/max from input to ensure consistent visualization
	// This prevents the "darkest" known value (e.g. 10.0) from becoming black (0)
	// if the diffusion fills in all the true zeros.
	//minVal, maxVal, _, _ := gocv.MinMaxLoc(grid)
	//if maxVal == minVal {
	//	maxVal = minVal + 1
	//}
	//scale := 255.0 / float64(maxVal-minVal)
	//shift := -float64(minVal) * scale
	// not sure why everything was rescaled
	scale := 1.0
	shift := 0.0

	// Save original data
	displayGrid := gocv.NewMat()
	defer displayGrid.Close()
	gocv.ConvertScaleAbs(grid, &displayGrid, scale, shift)
	gocv.IMWrite("input_data.png", displayGrid)
	fmt.Println("Saved: input_data.png")

	// Save mask
	gocv.IMWrite("mask.png", mask)
	fmt.Println("Saved: mask.png")

	// Save result (using same scale/shift as input)
	displayResult := gocv.NewMat()
	defer displayResult.Close()
	gocv.ConvertScaleAbs(filledGrid, &displayResult, scale, shift)
	gocv.IMWrite("result.png", displayResult)
	fmt.Println("Saved: result.png")

	fmt.Println("Processing complete!")
}

// FastSmoothFill solves Laplace's equation using Gauss-Seidel iteration at multiple scales
// This properly diffuses values from known regions throughout the domain
func FastSmoothFill(input *gocv.Mat, mask *gocv.Mat) gocv.Mat {
	rows, cols := input.Rows(), input.Cols()

	// Start with the input as our initial guess
	result := input.Clone()
	const loops = 30

	// Multigrid V-cycle: solve at multiple resolutions
	scales := []struct {
		factor float64
		iters  int
	}{
		{0.125, 20 * loops}, // Coarse - quick propagation
		{0.25, 15 * loops},
		{0.5, 10 * loops},
		{1.0, 5 * loops}, // Full resolution - fine detail
	}

	for scaleIdx, s := range scales {
		var working, workingMask gocv.Mat
		var workRows, workCols int

		if s.factor < 1.0 {
			// Downsample
			workCols = max(int(float64(cols)*s.factor), 3)
			workRows = max(int(float64(rows)*s.factor), 3)

			working = gocv.NewMat()
			workingMask = gocv.NewMat()

			gocv.Resize(result, &working, image.Point{X: workCols, Y: workRows}, 0, 0, gocv.InterpolationLinear)
			gocv.Resize(*mask, &workingMask, image.Point{X: workCols, Y: workRows}, 0, 0, gocv.InterpolationLinear)

			// Threshold mask to ensure it's binary
			gocv.Threshold(workingMask, &workingMask, 127, 255, gocv.ThresholdBinary)
		} else {
			working = result.Clone()
			workingMask = mask.Clone()
			workRows, workCols = rows, cols
		}

		fmt.Printf("Scale %d/%d (%.1f%% resolution): Solving %dx%d grid...\n",
			scaleIdx+1, len(scales), s.factor*100, workRows, workCols)

		// Pre-compute list of unknown cells (mask == 0)
		type cell struct{ y, x int }
		unknownCells := make([]cell, 0, workRows*workCols/2)
		for y := 0; y < workRows; y++ {
			for x := 0; x < workCols; x++ {
				if workingMask.GetUCharAt(y, x) == 0 {
					unknownCells = append(unknownCells, cell{y, x})
				}
			}
		}

		fmt.Printf("  Processing %d unknown cells\n", len(unknownCells))

		// Get direct access to data (much faster than GetFloatAt/SetFloatAt)
		dataPtr, err := working.DataPtrFloat32()
		if err != nil {
			// Fallback to slow method if direct access fails
			fmt.Printf("  Warning: Using slow access method\n")
			for iter := 0; iter < s.iters; iter++ {
				for _, c := range unknownCells {
					y, x := c.y, c.x
					currentVal := working.GetFloatAt(y, x)

					var north, south, west, east float32
					if y > 0 {
						north = working.GetFloatAt(y-1, x)
					} else {
						north = currentVal
					}
					if y < workRows-1 {
						south = working.GetFloatAt(y+1, x)
					} else {
						south = currentVal
					}
					if x > 0 {
						west = working.GetFloatAt(y, x-1)
					} else {
						west = currentVal
					}
					if x < workCols-1 {
						east = working.GetFloatAt(y, x+1)
					} else {
						east = currentVal
					}

					avg := (north + south + east + west) / 4.0
					working.SetFloatAt(y, x, avg)
				}

				if (iter+1)%50 == 0 {
					fmt.Printf("  Iteration %d/%d\n", iter+1, s.iters)
				}
			}
		} else {
			// Fast path: direct memory access
			stride := workCols
			for iter := 0; iter < s.iters; iter++ {
				for _, c := range unknownCells {
					y, x := c.y, c.x
					idx := y*stride + x
					currentVal := dataPtr[idx]

					var north, south, west, east float32
					if y > 0 {
						north = dataPtr[(y-1)*stride+x]
					} else {
						north = currentVal
					}
					if y < workRows-1 {
						south = dataPtr[(y+1)*stride+x]
					} else {
						south = currentVal
					}
					if x > 0 {
						west = dataPtr[y*stride+(x-1)]
					} else {
						west = currentVal
					}
					if x < workCols-1 {
						east = dataPtr[y*stride+(x+1)]
					} else {
						east = currentVal
					}

					dataPtr[idx] = (north + south + east + west) / 4.0
				}

				if (iter+1)%50 == 0 {
					fmt.Printf("  Iteration %d/%d\n", iter+1, s.iters)
				}
			}
		}

		// Upsample back to full resolution (or copy if already full res)
		if s.factor < 1.0 {
			gocv.Resize(working, &result, image.Point{X: cols, Y: rows}, 0, 0, gocv.InterpolationLinear)
			working.Close()
			workingMask.Close()
		} else {
			working.CopyTo(&result)
			working.Close()
			workingMask.Close()
		}

		// Always restore exact original values
		input.CopyToWithMask(&result, *mask)
	}

	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// simulateClusters populates the grid with random "known" data
func simulateClusters(grid *gocv.Mat, mask *gocv.Mat) {
	// Fill with zeros initially
	grid.SetTo(gocv.NewScalar(0, 0, 0, 0))
	mask.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Create 3 clusters
	clusters := []struct {
		row, col, radius int
		val              float32
	}{
		{rows / 4, cols / 4, rows / 11, 20.0},         // Top Left (Cool)
		{rows * 3 / 4, cols * 3 / 4, rows / 9, 150.0}, // Bottom Right (Hot)
		{rows / 2, cols / 4, rows / 8, 80.0},          // Middle Left (Warm)
	}

	for _, c := range clusters {
		// First draw the mask circle (255 = known region)
		gocv.Circle(mask, image.Point{c.col, c.row}, c.radius, color.RGBA{255, 255, 255, 255}, -1)

		// Now we need to set the float values in the grid
		// We'll iterate over the bounding box and set values where mask is 255
		minX := max(0, c.col-c.radius)
		maxX := min(grid.Cols()-1, c.col+c.radius)
		minY := max(0, c.row-c.radius)
		maxY := min(grid.Rows()-1, c.row+c.radius)

		// Manual pixel iteration to set float values
		for y := minY; y <= maxY; y++ {
			for x := minX; x <= maxX; x++ {
				// Check if this pixel is in the mask
				maskVal := mask.GetUCharAt(y, x)
				if maskVal == 255 {
					grid.SetFloatAt(y, x, c.val)
				}
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
