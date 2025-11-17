package sgfilter

import (
	"math"
)

// Point2D represents a 2D point
type Point2D struct {
	X, Y float64
}

func SavitzkyGolayFilter(points []Point2D, dt float64) (smoothed []Point2D, velocities []Point2D) {
	windowSize := 7
	return SavitzkyGolay(points, windowSize, dt)
}

// SavitzkyGolayFilter applies Savitzky-Golay smoothing and returns velocities
// windowSize should be odd (e.g., 5, 7, 9)
// dt is the time interval between points
func SavitzkyGolay(points []Point2D, windowSize int, dt float64) (smoothed []Point2D, velocities []Point2D) {
	n := len(points)
	if n < windowSize {
		return points, make([]Point2D, n)
	}

	// Ensure odd window size
	if windowSize%2 == 0 {
		windowSize++
	}
	halfWindow := windowSize / 2

	// Quadratic Savitzky-Golay coefficients for smoothing
	smoothCoeffs := getSGSmoothCoeffs(windowSize)
	// Coefficients for first derivative
	derivCoeffs := getSGDerivCoeffs(windowSize)

	smoothed = make([]Point2D, n)
	velocities = make([]Point2D, n)

	// Separate X and Y for processing
	xVals := make([]float64, n)
	yVals := make([]float64, n)
	for i := range points {
		xVals[i] = points[i].X
		yVals[i] = points[i].Y
	}

	// Apply filter to each axis
	xSmooth := applySGFilter(xVals, smoothCoeffs, halfWindow)
	ySmooth := applySGFilter(yVals, smoothCoeffs, halfWindow)
	xDeriv := applySGFilter(xVals, derivCoeffs, halfWindow)
	yDeriv := applySGFilter(yVals, derivCoeffs, halfWindow)

	for i := range smoothed {
		smoothed[i] = Point2D{X: xSmooth[i], Y: ySmooth[i]}
		velocities[i] = Point2D{X: xDeriv[i] / dt, Y: yDeriv[i] / dt}
	}

	return smoothed, velocities
}

func applySGFilter(data []float64, coeffs []float64, halfWindow int) []float64 {
	n := len(data)
	result := make([]float64, n)

	for i := 0; i < n; i++ {
		sum := 0.0

		// Handle edges by clamping indices (original approach)
		for j := -halfWindow; j <= halfWindow; j++ {
			idx := i + j
			if idx < 0 {
				idx = 0
			} else if idx >= n {
				idx = n - 1
			}
			sum += coeffs[j+halfWindow] * data[idx]
		}
		result[i] = sum
	}

	return result
}

// Precomputed Savitzky-Golay coefficients for window size 5, quadratic
func getSGSmoothCoeffs(windowSize int) []float64 {
	// For window size 5 (most common)
	if windowSize == 5 {
		return []float64{-3.0 / 35.0, 12.0 / 35.0, 17.0 / 35.0, 12.0 / 35.0, -3.0 / 35.0}
	}
	// For window size 7
	if windowSize == 7 {
		return []float64{-2.0 / 21.0, 3.0 / 21.0, 6.0 / 21.0, 7.0 / 21.0, 6.0 / 21.0, 3.0 / 21.0, -2.0 / 21.0}
	}
	// Default: simple moving average
	coeffs := make([]float64, windowSize)
	for i := range coeffs {
		coeffs[i] = 1.0 / float64(windowSize)
	}
	return coeffs
}

func getSGDerivCoeffs(windowSize int) []float64 {
	// For window size 5 (first derivative, quadratic)
	if windowSize == 5 {
		return []float64{-2.0 / 10.0, -1.0 / 10.0, 0.0, 1.0 / 10.0, 2.0 / 10.0}
	}
	// For window size 7
	if windowSize == 7 {
		return []float64{-3.0 / 28.0, -2.0 / 28.0, -1.0 / 28.0, 0.0, 1.0 / 28.0, 2.0 / 28.0, 3.0 / 28.0}
	}
	// Default: simple finite difference
	coeffs := make([]float64, windowSize)
	halfWindow := windowSize / 2
	for i := range coeffs {
		coeffs[i] = float64(i-halfWindow) / float64(halfWindow*halfWindow)
	}
	return coeffs
}

func GaussianSmoothFilter(points []Point2D, dt float64) (smoothed []Point2D, velocities []Point2D) {
	sigma := 1.0
	windowSize := 5
	smoothed = GaussianSmooth(points, sigma, windowSize)
	velocities = ComputeVelocities(smoothed, dt)
	return smoothed, velocities
}

// Simple Gaussian smoothing alternative
func GaussianSmooth(points []Point2D, sigma float64, windowSize int) []Point2D {
	if windowSize%2 == 0 {
		windowSize++
	}
	halfWindow := windowSize / 2

	// Generate Gaussian kernel
	kernel := make([]float64, windowSize)
	sum := 0.0
	for i := 0; i < windowSize; i++ {
		x := float64(i - halfWindow)
		kernel[i] = math.Exp(-x * x / (2 * sigma * sigma))
		sum += kernel[i]
	}
	// Normalize
	for i := range kernel {
		kernel[i] /= sum
	}

	n := len(points)
	smoothed := make([]Point2D, n)

	for i := 0; i < n; i++ {
		var sumX, sumY float64
		for j := -halfWindow; j <= halfWindow; j++ {
			idx := i + j
			if idx < 0 {
				idx = 0
			} else if idx >= n {
				idx = n - 1
			}
			sumX += kernel[j+halfWindow] * points[idx].X
			sumY += kernel[j+halfWindow] * points[idx].Y
		}
		smoothed[i] = Point2D{X: sumX, Y: sumY}
	}

	return smoothed
}

// Compute velocities from smoothed points
func ComputeVelocities(points []Point2D, dt float64) []Point2D {
	n := len(points)
	velocities := make([]Point2D, n)

	for i := 0; i < n; i++ {
		if i == 0 {
			// Forward difference
			velocities[i] = Point2D{
				X: (points[1].X - points[0].X) / dt,
				Y: (points[1].Y - points[0].Y) / dt,
			}
		} else if i == n-1 {
			// Backward difference
			velocities[i] = Point2D{
				X: (points[n-1].X - points[n-2].X) / dt,
				Y: (points[n-1].Y - points[n-2].Y) / dt,
			}
		} else {
			// Central difference
			velocities[i] = Point2D{
				X: (points[i+1].X - points[i-1].X) / (2 * dt),
				Y: (points[i+1].Y - points[i-1].Y) / (2 * dt),
			}
		}
	}

	return velocities
}

// LinearFilter applies linear interpolation which preserves endpoints and has minimal edge effects
func LinearFilter(points []Point2D, dt float64) (smoothed []Point2D, velocities []Point2D) {
	n := len(points)
	if n < 2 {
		return points, make([]Point2D, n)
	}

	smoothed = make([]Point2D, n)
	velocities = make([]Point2D, n)

	// For linear filter, we just return the original points but compute velocities properly
	copy(smoothed, points)

	// Compute velocities with proper handling of edge cases
	for i := 0; i < n; i++ {
		if i == 0 {
			// Forward difference for first point
			if n > 1 {
				velocities[i] = Point2D{
					X: (points[1].X - points[0].X) / dt,
					Y: (points[1].Y - points[0].Y) / dt,
				}
			} else {
				velocities[i] = Point2D{X: 0, Y: 0}
			}
		} else if i == n-1 {
			// Backward difference for last point
			velocities[i] = Point2D{
				X: (points[n-1].X - points[n-2].X) / dt,
				Y: (points[n-1].Y - points[n-2].Y) / dt,
			}
		} else {
			// Central difference for interior points
			velocities[i] = Point2D{
				X: (points[i+1].X - points[i-1].X) / (2 * dt),
				Y: (points[i+1].Y - points[i-1].Y) / (2 * dt),
			}
		}
	}

	return smoothed, velocities
}

// RobustFilter applies iterative smoothing with outlier detection and correction
// This filter first applies smoothing, identifies outliers based on distance from the smooth curve,
// excludes outliers, and re-smooths to determine proper values for outlier points
func RobustFilter(points []Point2D, dt float64) (smoothed []Point2D, velocities []Point2D) {
	n := len(points)
	if n < 7 {
		return points, make([]Point2D, n)
	}

	// Make a working copy of the data
	currentPoints := make([]Point2D, n)
	copy(currentPoints, points)

	// First pass: apply initial smoothing to establish trend
	initialSmoothX, _ := apply1DSmoothing(extractX(points), 7)
	initialSmoothY, _ := apply1DSmoothing(extractY(points), 7)

	// Calculate distances from original points to initial smooth curve
	distances := make([]float64, n)
	for i := 0; i < n; i++ {
		dx := points[i].X - initialSmoothX[i]
		dy := points[i].Y - initialSmoothY[i]
		distances[i] = math.Sqrt(dx*dx + dy*dy)
	}

	// Calculate threshold (e.g., 2 standard deviations from mean distance)
	meanDist, stdDist := calculateMeanStd(distances)
	threshold := meanDist + 2.0*stdDist

	// Identify outliers (points significantly away from smooth trend)
	outlierMask := make([]bool, n)
	for i := 0; i < n; i++ {
		if distances[i] > threshold {
			outlierMask[i] = true
		}
	}

	// Replace outliers with smoothed values temporarily for re-smoothing
	workingPoints := make([]Point2D, n)
	for i := 0; i < n; i++ {
		if outlierMask[i] {
			// Use the initial smooth value for outliers
			workingPoints[i] = Point2D{X: initialSmoothX[i], Y: initialSmoothY[i]}
		} else {
			// Keep non-outliers as-is
			workingPoints[i] = points[i]
		}
	}

	// Apply smoothing to the cleaned data
	finalSmoothX, dX := apply1DSmoothing(extractX(workingPoints), 7)
	finalSmoothY, dY := apply1DSmoothing(extractY(workingPoints), 7)

	// Create final result
	smoothed = make([]Point2D, n)
	velocities = make([]Point2D, n)

	for i := range points {
		smoothed[i] = Point2D{X: finalSmoothX[i], Y: finalSmoothY[i]}
		velocities[i] = Point2D{X: dX[i] / dt, Y: dY[i] / dt}
	}

	return smoothed, velocities
}

// Helper function to extract X coordinates
func extractX(points []Point2D) []float64 {
	result := make([]float64, len(points))
	for i, p := range points {
		result[i] = p.X
	}
	return result
}

// Helper function to extract Y coordinates
func extractY(points []Point2D) []float64 {
	result := make([]float64, len(points))
	for i, p := range points {
		result[i] = p.Y
	}
	return result
}

// apply1DSmoothing applies smoothing and returns both smoothed values and derivatives
func apply1DSmoothing(data []float64, windowSize int) ([]float64, []float64) {
	if len(data) < windowSize {
		result := make([]float64, len(data))
		copy(result, data)
		return result, make([]float64, len(data))
	}

	if windowSize%2 == 0 {
		windowSize++
	}
	halfWindow := windowSize / 2

	smoothCoeffs := getSGSmoothCoeffs(windowSize)
	derivCoeffs := getSGDerivCoeffs(windowSize)

	smoothed := applySGFilter(data, smoothCoeffs, halfWindow)
	derivatives := applySGFilter(data, derivCoeffs, halfWindow)

	return smoothed, derivatives
}

// calculateMeanStd calculates mean and standard deviation
func calculateMeanStd(data []float64) (mean, std float64) {
	// Calculate mean
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	mean = sum / float64(len(data))

	// Calculate standard deviation
	sumSquares := 0.0
	for _, v := range data {
		diff := v - mean
		sumSquares += diff * diff
	}
	std = math.Sqrt(sumSquares / float64(len(data)))

	return mean, std
}
