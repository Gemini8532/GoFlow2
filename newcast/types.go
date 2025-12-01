package newcast

// Vector represents a 2D velocity vector.
type Vector struct {
	Vx, Vy float64
}

// Point represents a point in space.
type Point struct {
	X, Y float64
}

// Polynomial represents the coefficients of a degree 2 polynomial: a*t^2 + b*t + c
type Polynomial struct {
	A, B, C float64
}

// Track represents the path of a single feature over time.
type Track struct {
	ID                 int
	Points             []Point
	LatestVelocity     Point
	LatestAcceleration Point
	Lost               bool
	PolyX              Polynomial // Polynomial for X coordinate
	PolyY              Polynomial // Polynomial for Y coordinate
}

// TrackFitQuality contains metrics for evaluating how well a polynomial fits a track
type TrackFitQuality struct {
	RSquared     float64 // Coefficient of determination (1.0 = perfect fit)
	RMSE         float64 // Root mean square error
	MaxDeviation float64 // Maximum distance from fitted curve
	AvgAccel     float64 // Average acceleration magnitude
	MaxAccel     float64 // Maximum acceleration magnitude
}

// CurveFitConfig filters tracks based on polynomial fit quality
// This is more principled than ad-hoc angle/smoothness checks
type CurveFitConfig struct {
	MinRSquared     float64 // Minimum R² value (e.g., 0.8)
	MaxRMSE         float64 // Maximum RMSE in pixels (e.g., 5.0)
	MaxDeviation    float64 // Maximum deviation from curve in pixels (e.g., 10.0)
	MaxAcceleration float64 // Maximum acceleration in pixels/frame² (e.g., 2.0)
}

// ProcessConfig holds all the parameters for processing images into tracks.
type ProcessConfig struct {
	MaxFeatures    int
	MinTrackLength int
	BlurSigma      float64 // Gaussian blur sigma for flow grid smoothing (0 = no blur)

	// Curve-fit filtering parameters
	MinRSquared     float64 // Minimum R² for polynomial fit (e.g., 0.85)
	MaxRMSE         float64 // Maximum RMSE in pixels (e.g., 3.0)
	MaxDeviation    float64 // Maximum deviation from curve in pixels (e.g., 8.0)
	MaxAcceleration float64 // Maximum acceleration in pixels/frame² (e.g., 2.0)
}

// AverageFlowGrid represents a single grid of averaged vectors.
type AverageFlowGrid struct {
	Width  int
	Height int
	Data   []Vector
}
