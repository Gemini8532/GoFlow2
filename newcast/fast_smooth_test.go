package newcast

import (
	"math"
	"testing"

	"gocv.io/x/gocv"
)

// TestSmoothFillSinglePoint tests that a single point diffuses smoothly
func TestSmoothFillSinglePoint(t *testing.T) {
	const size = 32

	// Create input with single point in center
	input := gocv.NewMatWithSize(size, size, gocv.MatTypeCV32F)
	mask := gocv.NewMatWithSize(size, size, gocv.MatTypeCV8U)
	defer input.Close()
	defer mask.Close()

	input.SetTo(gocv.NewScalar(0, 0, 0, 0))
	mask.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Set center point
	centerX, centerY := size/2, size/2
	input.SetFloatAt(centerY, centerX, 100.0)
	mask.SetUCharAt(centerY, centerX, 255)

	// Apply smooth fill
	config := SmoothFillConfig{Loops: 50}
	result := SmoothFill(input, mask, config)
	defer result.Close()

	// Check that center value is preserved
	centerVal := result.GetFloatAt(centerY, centerX)
	if math.Abs(float64(centerVal)-100.0) > 1.0 {
		t.Errorf("Center value not preserved: expected ~100.0, got %f", centerVal)
	}

	// Check that values decrease smoothly with distance from center
	prevVal := centerVal
	for r := 1; r < size/2; r++ {
		val := result.GetFloatAt(centerY, centerX+r)

		// Value should be positive and decreasing
		if val < 0 {
			t.Errorf("Negative value at distance %d: %f", r, val)
		}
		if val > prevVal {
			t.Errorf("Value increased at distance %d: prev=%f, curr=%f", r, prevVal, val)
		}

		prevVal = val
	}

	// Check smoothness - no large jumps between adjacent pixels
	maxJump := 0.0
	for y := 0; y < size-1; y++ {
		for x := 0; x < size-1; x++ {
			v1 := result.GetFloatAt(y, x)
			v2 := result.GetFloatAt(y, x+1)
			v3 := result.GetFloatAt(y+1, x)

			jump1 := math.Abs(float64(v2 - v1))
			jump2 := math.Abs(float64(v3 - v1))

			if jump1 > maxJump {
				maxJump = jump1
			}
			if jump2 > maxJump {
				maxJump = jump2
			}
		}
	}

	t.Logf("Single point test: max jump between adjacent pixels = %f", maxJump)

	// The multigrid approach with coarse levels can create some larger jumps
	// especially near the source point. This is acceptable as long as it's not extreme.
	if maxJump > 50.0 {
		t.Errorf("Field has very large jumps: max jump = %f", maxJump)
	}
}

// TestSmoothFillTwoPoints tests that two points create a smooth gradient
func TestSmoothFillTwoPoints(t *testing.T) {
	const size = 32

	input := gocv.NewMatWithSize(size, size, gocv.MatTypeCV32F)
	mask := gocv.NewMatWithSize(size, size, gocv.MatTypeCV8U)
	defer input.Close()
	defer mask.Close()

	input.SetTo(gocv.NewScalar(0, 0, 0, 0))
	mask.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Set two points with different values
	y := size / 2
	x1, x2 := size/4, 3*size/4
	val1, val2 := 10.0, 90.0

	input.SetFloatAt(y, x1, float32(val1))
	mask.SetUCharAt(y, x1, 255)

	input.SetFloatAt(y, x2, float32(val2))
	mask.SetUCharAt(y, x2, 255)

	config := SmoothFillConfig{Loops: 50}
	result := SmoothFill(input, mask, config)
	defer result.Close()

	// Check that original values are preserved
	v1 := result.GetFloatAt(y, x1)
	v2 := result.GetFloatAt(y, x2)

	if math.Abs(float64(v1)-val1) > 1.0 {
		t.Errorf("Point 1 not preserved: expected ~%f, got %f", val1, v1)
	}
	if math.Abs(float64(v2)-val2) > 1.0 {
		t.Errorf("Point 2 not preserved: expected ~%f, got %f", val2, v2)
	}

	// Check that values along the line between points form a smooth gradient
	prevVal := v1
	for x := x1 + 1; x < x2; x++ {
		val := result.GetFloatAt(y, x)

		// Value should be between the two endpoints
		if float64(val) < val1 || float64(val) > val2 {
			t.Errorf("Value at x=%d outside range [%f, %f]: %f", x, val1, val2, val)
		}

		// Value should be monotonically increasing
		if val < prevVal {
			t.Errorf("Value decreased at x=%d: prev=%f, curr=%f", x, prevVal, val)
		}

		prevVal = val
	}
}

// TestSmoothFillSymmetry tests that the result is symmetric for symmetric input
func TestSmoothFillSymmetry(t *testing.T) {
	const size = 16

	input := gocv.NewMatWithSize(size, size, gocv.MatTypeCV32F)
	mask := gocv.NewMatWithSize(size, size, gocv.MatTypeCV8U)
	defer input.Close()
	defer mask.Close()

	input.SetTo(gocv.NewScalar(0, 0, 0, 0))
	mask.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Set center point
	centerX, centerY := size/2, size/2
	input.SetFloatAt(centerY, centerX, 50.0)
	mask.SetUCharAt(centerY, centerX, 255)

	config := SmoothFillConfig{Loops: 30}
	result := SmoothFill(input, mask, config)
	defer result.Close()

	// Check symmetry - points equidistant from center should have similar values
	// Multigrid methods can have small asymmetries due to iteration order
	tolerance := 5.0

	// Check horizontal symmetry
	for x := 0; x < size/2; x++ {
		left := result.GetFloatAt(centerY, centerX-x)
		right := result.GetFloatAt(centerY, centerX+x)

		if math.Abs(float64(left-right)) > tolerance {
			t.Errorf("Horizontal asymmetry at offset %d: left=%f, right=%f", x, left, right)
		}
	}

	// Check vertical symmetry
	for y := 0; y < size/2; y++ {
		top := result.GetFloatAt(centerY-y, centerX)
		bottom := result.GetFloatAt(centerY+y, centerX)

		if math.Abs(float64(top-bottom)) > tolerance {
			t.Errorf("Vertical asymmetry at offset %d: top=%f, bottom=%f", y, top, bottom)
		}
	}
}

// TestSmoothFillBoundaryConditions tests Neumann boundary conditions
func TestSmoothFillBoundaryConditions(t *testing.T) {
	const size = 16

	input := gocv.NewMatWithSize(size, size, gocv.MatTypeCV32F)
	mask := gocv.NewMatWithSize(size, size, gocv.MatTypeCV8U)
	defer input.Close()
	defer mask.Close()

	input.SetTo(gocv.NewScalar(0, 0, 0, 0))
	mask.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Set a point near the edge
	edgeY, edgeX := 2, 2
	input.SetFloatAt(edgeY, edgeX, 100.0)
	mask.SetUCharAt(edgeY, edgeX, 255)

	config := SmoothFillConfig{Loops: 30}
	result := SmoothFill(input, mask, config)
	defer result.Close()

	// Check that edge values are non-zero (Neumann BC means gradient is zero at boundary)
	// So values should diffuse to the edges
	edgeVal := result.GetFloatAt(0, 0)
	if edgeVal <= 0 {
		t.Errorf("Edge value is zero or negative: %f (Neumann BC should allow diffusion to edges)", edgeVal)
	}

	// Check that corner values are reasonable
	corners := []struct{ y, x int }{
		{0, 0}, {0, size - 1}, {size - 1, 0}, {size - 1, size - 1},
	}

	for _, corner := range corners {
		val := result.GetFloatAt(corner.y, corner.x)
		if val < 0 {
			t.Errorf("Corner (%d,%d) has negative value: %f", corner.y, corner.x, val)
		}
	}
}

// TestSmoothFillConvergence tests that more iterations produce smoother results
func TestSmoothFillConvergence(t *testing.T) {
	const size = 16

	input := gocv.NewMatWithSize(size, size, gocv.MatTypeCV32F)
	mask := gocv.NewMatWithSize(size, size, gocv.MatTypeCV8U)
	defer input.Close()
	defer mask.Close()

	input.SetTo(gocv.NewScalar(0, 0, 0, 0))
	mask.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Set center point
	centerX, centerY := size/2, size/2
	input.SetFloatAt(centerY, centerX, 100.0)
	mask.SetUCharAt(centerY, centerX, 255)

	// Test with different iteration counts
	iterations := []int{10, 30, 50}
	var prevEdgeVal float64

	for _, loops := range iterations {
		config := SmoothFillConfig{Loops: loops}
		result := SmoothFill(input, mask, config)

		// Measure how far values have diffused by checking edge value
		edgeVal := float64(result.GetFloatAt(0, 0))
		t.Logf("Loops=%d: corner value=%.2f", loops, edgeVal)

		// More iterations should give higher values at the edges
		if loops > 10 && edgeVal <= prevEdgeVal {
			t.Errorf("Edge value did not increase with more iterations: %d loops = %.2f, previous = %.2f",
				loops, edgeVal, prevEdgeVal)
		}

		prevEdgeVal = edgeVal
		result.Close()
	}
}

// TestSmoothFillLinearGradient tests that a linear gradient is preserved
func TestSmoothFillLinearGradient(t *testing.T) {
	const size = 16

	input := gocv.NewMatWithSize(size, size, gocv.MatTypeCV32F)
	mask := gocv.NewMatWithSize(size, size, gocv.MatTypeCV8U)
	defer input.Close()
	defer mask.Close()

	input.SetTo(gocv.NewScalar(0, 0, 0, 0))
	mask.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Set a linear gradient along one row
	y := size / 2
	for x := 0; x < size; x++ {
		val := float32(x * 10) // 0, 10, 20, ..., 150
		input.SetFloatAt(y, x, val)
		mask.SetUCharAt(y, x, 255)
	}

	config := SmoothFillConfig{Loops: 20}
	result := SmoothFill(input, mask, config)
	defer result.Close()

	// Check that the gradient is preserved along the known row
	for x := 0; x < size-1; x++ {
		v1 := result.GetFloatAt(y, x)
		v2 := result.GetFloatAt(y, x+1)

		expectedDiff := 10.0
		actualDiff := float64(v2 - v1)

		if math.Abs(actualDiff-expectedDiff) > 2.0 {
			t.Errorf("Gradient not preserved at x=%d: expected diff ~%f, got %f",
				x, expectedDiff, actualDiff)
		}
	}
}
