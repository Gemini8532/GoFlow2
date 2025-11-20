package newcast

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

// SmoothFillConfig holds configuration for the smooth fill algorithm
type SmoothFillConfig struct {
	Loops int // Number of iteration loops (multiplier for base iterations)
}

// DefaultSmoothFillConfig returns a reasonable default configuration
func DefaultSmoothFillConfig() SmoothFillConfig {
	return SmoothFillConfig{
		Loops: 50,
	}
}

// SmoothFill solves Laplace's equation using Gauss-Seidel iteration at full resolution.
// This properly diffuses values from known regions throughout the domain.
// input: CV32F matrix with known values
// mask: CV8U matrix where 255 = known value, 0 = unknown (to be filled)
func SmoothFill(input gocv.Mat, mask gocv.Mat, config SmoothFillConfig) gocv.Mat {
	rows, cols := input.Rows(), input.Cols()

	// Start with the input as our initial guess
	result := input.Clone()

	// Multigrid V-cycle: solve at multiple resolutions
	scales := []struct {
		factor float64
		iters  int
	}{
		{0.125, 20 * config.Loops}, // Coarse - quick propagation
		{0.25, 15 * config.Loops},
		{0.5, 10 * config.Loops},
		{1.0, 5 * config.Loops}, // Full resolution - fine detail
	}

	for _, s := range scales {
		var working, workingMask gocv.Mat
		var workRows, workCols int

		if s.factor < 1.0 {
			// Downsample
			workCols = max(int(float64(cols)*s.factor), 3)
			workRows = max(int(float64(rows)*s.factor), 3)

			working = gocv.NewMat()
			workingMask = gocv.NewMat()

			gocv.Resize(result, &working, image.Point{X: workCols, Y: workRows}, 0, 0, gocv.InterpolationLinear)
			gocv.Resize(mask, &workingMask, image.Point{X: workCols, Y: workRows}, 0, 0, gocv.InterpolationLinear)

			// Threshold mask to ensure it's binary
			gocv.Threshold(workingMask, &workingMask, 127, 255, gocv.ThresholdBinary)
		} else {
			working = result.Clone()
			workingMask = mask.Clone()
			workRows, workCols = rows, cols
		}

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

		// Get direct access to data (much faster than GetFloatAt/SetFloatAt)
		dataPtr, err := working.DataPtrFloat32()
		if err != nil {
			// Fallback to slow method if direct access fails
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
		input.CopyToWithMask(&result, mask)
	}

	return result
}

// SmoothFillVerbose is like SmoothFill but prints progress information
func SmoothFillVerbose(input gocv.Mat, mask gocv.Mat, config SmoothFillConfig) gocv.Mat {
	rows, cols := input.Rows(), input.Cols()

	// Start with the input as our initial guess
	result := input.Clone()

	// Multigrid V-cycle: solve at multiple resolutions
	scales := []struct {
		factor float64
		iters  int
	}{
		{0.125, 20 * config.Loops}, // Coarse - quick propagation
		{0.25, 15 * config.Loops},
		{0.5, 10 * config.Loops},
		{1.0, 5 * config.Loops}, // Full resolution - fine detail
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
			gocv.Resize(mask, &workingMask, image.Point{X: workCols, Y: workRows}, 0, 0, gocv.InterpolationLinear)

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
			gocv.Resize(working, &result, image.Point{X: cols, Y: rows}, 0, 0, gocv.InterpolationCubic)
			working.Close()
			workingMask.Close()
		} else {
			working.CopyTo(&result)
			working.Close()
			workingMask.Close()
		}

		// Always restore exact original values
		input.CopyToWithMask(&result, mask)
	}

	return result
}
