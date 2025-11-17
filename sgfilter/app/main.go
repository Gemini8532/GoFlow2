package main

import (
	"example/goflow/sgfilter"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"
)

func main() {
	// Define command line flags
	numPoints := flag.Int("n", 20, "number of points to generate for the noisy circle")
	filterType := flag.String("filter", "sgwrapper", "filter type to use (options: sgwrapper, catmull, kalman, bilateral, gaussian, sg)")
	curvature := flag.Float64("curvature", 1.0, "curvature factor (0.0 = straight line, 0.5 = arc, 1.0 = full circle)")
	noiseLevel := flag.Float64("noise", 0.5, "noise level (standard deviation for Gaussian noise)")
	impulseFraction := flag.Float64("impulse", 0.0, "fraction of points affected by impulse noise (0.0 = none, 1.0 = all points)")
	impulseMagnitude := flag.Float64("impulse-mag", 3.0, "magnitude of impulse noise")
	seed := flag.Int64("seed", time.Now().UnixNano(), "random seed for reproducible results (default is current time)")
	help := flag.Bool("h", false, "show help")

	// Parse command line flags
	flag.Parse()

	// Set random seed for reproducible results
	rand.Seed(*seed)

	// Show help and exit if requested
	if *help {
		fmt.Println("Usage: app [OPTIONS]")
		fmt.Println("Options:")
		fmt.Println("  -n int")
		fmt.Println("        number of points to generate for the noisy circle (default 20)")
		fmt.Println("  -filter string")
		fmt.Println("        filter type to use (options: sgwrapper, catmull, kalman, bilateral, gaussian, sg) (default \"sgwrapper\")")
		fmt.Println("  -curvature float")
		fmt.Println("        curvature factor (0.0 = straight line, 0.5 = arc, 1.0 = full circle) (default 1)")
		fmt.Println("  -noise float")
		fmt.Println("        noise level (standard deviation for Gaussian noise) (default 0.5)")
		fmt.Println("  -impulse float")
		fmt.Println("        fraction of points affected by impulse noise (0.0 = none, 1.0 = all points) (default 0.0)")
		fmt.Println("  -impulse-mag float")
		fmt.Println("        magnitude of impulse noise (default 3.0)")
		fmt.Println("  -seed int64")
		fmt.Println("        random seed for reproducible results (default is current time)")
		fmt.Println("  -h")
		fmt.Println("        show help")
		fmt.Println("\nFilter types:")
		fmt.Println("  sgwrapper   - Savitzky-Golay filter (wrapper)")
		fmt.Println("  sg          - Savitzky-Golay filter (direct)")
		fmt.Println("  catmull     - Catmull-Rom Spline filter")
		fmt.Println("  kalman      - Kalman Filter")
		fmt.Println("  bilateral   - Bilateral Filter")
		fmt.Println("  gaussian    - Gaussian Smoothing + velocity computation")
		fmt.Println("  robust      - Robust filter with outlier detection and correction")
		fmt.Println("  linear      - Linear filter with minimal edge effects (no smoothing)")
		fmt.Println("\nCurvature values:")
		fmt.Println("  0.0         - straight line")
		fmt.Println("  0.5         - arc (partial circle)")
		fmt.Println("  1.0         - full circle")
		return
	}

	fmt.Printf("Generating %d points for the noisy circle...\n", *numPoints)
	fmt.Printf("Using filter type: %s\n", *filterType)
	fmt.Printf("Using curvature factor: %.2f\n", *curvature)
	fmt.Printf("Using noise level: %.2f\n", *noiseLevel)
	if *impulseFraction > 0 {
		fmt.Printf("Using impulse noise: %.2f fraction with magnitude %.2f\n", *impulseFraction, *impulseMagnitude)
	}

	// Generate both clean and noisy circle points
	cleanPoints, noisyPoints := generateCleanAndNoisyCircle(*numPoints, 5.0, *noiseLevel, *curvature, *impulseFraction, *impulseMagnitude) // numPoints, radius 5.0, noiseLevel, curvature, impulseFraction, impulseMagnitude

	dt := 0.1 // 100ms between frames

	var smoothed []sgfilter.Point2D
	var velocities []sgfilter.Point2D

	// Apply selected filter
	var filterDisplayName string
	switch *filterType {
	case "sg", "savgolay", "savitzkygolay":
		// Note: SavitzkyGolay needs windowSize parameter, so we can't use the wrapper
		smoothed, velocities = sgfilter.SavitzkyGolay(noisyPoints, 7, dt)
		filterDisplayName = "Savitzky-Golay"
		fmt.Println("Savitzky-Golay Results:")
	case "sgwrapper":
		// Use the wrapper function that doesn't require window size
		smoothed, velocities = sgfilter.SavitzkyGolayFilter(noisyPoints, dt)
		filterDisplayName = "Savitzky-Golay (wrapper)"
		fmt.Println("Savitzky-Golay (wrapper) Results:")
	case "catmull", "catmullrom":
		smoothed, velocities = sgfilter.CatmullRomFilter(noisyPoints, dt)
		filterDisplayName = "Catmull-Rom Spline"
		fmt.Println("Catmull-Rom Spline Results:")
	case "kalman":
		smoothed, velocities = sgfilter.KalmanFilter(noisyPoints, dt)
		filterDisplayName = "Kalman Filter"
		fmt.Println("Kalman Filter Results:")
	case "bilateral":
		smoothed, velocities = sgfilter.BilateralFilter(noisyPoints, dt)
		filterDisplayName = "Bilateral Filter"
		fmt.Println("Bilateral Filter Results:")
	case "gaussian":
		// Gaussian smoothing + velocity computation
		smoothed, velocities = sgfilter.GaussianSmoothFilter(noisyPoints, dt)
		fmt.Println("Gaussian Smoothing Results:")
	case "robust":
		// Robust smoothing with outlier detection and correction
		smoothed, velocities = sgfilter.RobustFilter(noisyPoints, dt)
		filterDisplayName = "Robust Filter"
		fmt.Println("Robust Filter Results:")
	case "linear":
		// Linear filter with minimal edge effects (essentially no smoothing, just velocity computation)
		smoothed, velocities = sgfilter.LinearFilter(noisyPoints, dt)
		filterDisplayName = "Linear Filter (minimal edge effects)"
		fmt.Println("Linear Filter Results:")
	default:
		// Default to Savitzky-Golay wrapper
		smoothed, velocities = sgfilter.SavitzkyGolayFilter(noisyPoints, dt)
		filterDisplayName = "Savitzky-Golay (wrapper)"
		fmt.Printf("Unknown filter type '%s', defaulting to Savitzky-Golay (wrapper)\n", *filterType)
		fmt.Println("Savitzky-Golay (wrapper) Results:")
	}

	// Print results with original points for comparison
	fmt.Println("\nComparison of Points:")
	fmt.Println("Index\tOriginal X\tOriginal Y\tSmoothed X\tSmoothed Y\tVelocity X\tVelocity Y")
	fmt.Println("-----\t----------\t----------\t----------\t----------\t----------\t----------")
	for i := range smoothed {
		fmt.Printf("%d\t%.2f\t\t%.2f\t\t%.2f\t\t%.2f\t\t%.2f\t\t%.2f\n",
			i, 
			noisyPoints[i].X, 
			noisyPoints[i].Y, 
			smoothed[i].X, 
			smoothed[i].Y, 
			velocities[i].X, 
			velocities[i].Y)
	}

	err := CreatePlots(cleanPoints, noisyPoints, smoothed, filterDisplayName)
	if err != nil {
		log.Printf("Could not create plots: %v", err)
	}
}

// generateCleanAndNoisyCircle creates both clean circle points and noisy versions
func generateCleanAndNoisyCircle(numPoints int, radius float64, noiseLevel float64, curvature float64, impulseFraction float64, impulseMagnitude float64) (clean []sgfilter.Point2D, noisy []sgfilter.Point2D) {
	clean = make([]sgfilter.Point2D, numPoints)
	noisy = make([]sgfilter.Point2D, numPoints)

	for i := 0; i < numPoints; i++ {
		// Calculate angle with gradually increasing velocity (acceleration)
		// Using a quadratic function to make angular velocity increase over time
		t := float64(i) / float64(numPoints-1) // normalized time from 0 to 1

		// Calculate coordinates based on curvature value
		// Curvature: 0.0 = straight line, 0.5 = half circle (arc), 1.0 = full circle
		var x, y float64
		
		// For a smooth transition, we'll use different approaches for different ranges
		if curvature <= 0.0 {
			// Straight line (horizontal from -radius to +radius)
			x = radius * (2*t - 1) // from -radius to +radius
			y = 0.0
		} else if curvature <= 1.0 {
			// Progressively move from straight line to full circle
			// When curvature = 0.5, we get a half circle (180 degrees)
			// When curvature = 1.0, we get a full circle (360 degrees)
			maxAngle := 2 * math.Pi * curvature  // angle range 0 to 2π
			angle := maxAngle * t                // angle increases linearly
			x = radius * math.Cos(angle)
			y = radius * math.Sin(angle)
		} else {
			// For values > 1.0, let's continue the circle (multiple rotations)
			maxAngle := 2 * math.Pi * curvature
			angle := maxAngle * t
			x = radius * math.Cos(angle)
			y = radius * math.Sin(angle)
		}

		clean[i] = sgfilter.Point2D{X: x, Y: y}

		// Add Gaussian noise to create noisy version
		noisyX := x + rand.NormFloat64() * noiseLevel
		noisyY := y + rand.NormFloat64() * noiseLevel

		// Add impulse noise to a fraction of points
		if impulseFraction > 0 {
			if rand.Float64() < impulseFraction {
				// Add impulse noise (much larger than Gaussian noise)
				angle := rand.Float64() * 2 * math.Pi  // Random direction for impulse
				impulseX := impulseMagnitude * math.Cos(angle)
				impulseY := impulseMagnitude * math.Sin(angle)
				noisyX += impulseX
				noisyY += impulseY
			}
		}

		noisy[i] = sgfilter.Point2D{X: noisyX, Y: noisyY}
	}

	return clean, noisy
}
