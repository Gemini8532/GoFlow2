package newcast

import (
	"math"
)

// EvaluateTrackFit calculates quality metrics for a polynomial fit to a track
func EvaluateTrackFit(track *Track) TrackFitQuality {
	if len(track.Points) < 3 {
		return TrackFitQuality{RSquared: 0, RMSE: math.Inf(1), MaxDeviation: math.Inf(1)}
	}

	// Fit polynomial if not already done
	polyX, polyY, err := FitQuadratic(track.Points)
	if err != nil {
		return TrackFitQuality{RSquared: 0, RMSE: math.Inf(1), MaxDeviation: math.Inf(1)}
	}

	// Calculate fit quality metrics
	n := len(track.Points)
	var sumSquaredError, sumSquaredTotal float64
	var maxDeviation float64

	// Calculate mean position
	var meanX, meanY float64
	for _, p := range track.Points {
		meanX += p.X
		meanY += p.Y
	}
	meanX /= float64(n)
	meanY /= float64(n)

	// Calculate errors and deviations
	for i, p := range track.Points {
		t := float64(i)

		// Predicted position from polynomial
		predX := polyX.Eval(t)
		predY := polyY.Eval(t)

		// Error (distance from fitted curve)
		errorX := p.X - predX
		errorY := p.Y - predY
		squaredError := errorX*errorX + errorY*errorY
		sumSquaredError += squaredError

		// Deviation from fitted curve
		deviation := math.Sqrt(squaredError)
		if deviation > maxDeviation {
			maxDeviation = deviation
		}

		// Total variance (distance from mean)
		totalX := p.X - meanX
		totalY := p.Y - meanY
		sumSquaredTotal += totalX*totalX + totalY*totalY
	}

	// Calculate R² (coefficient of determination)
	rSquared := 1.0
	if sumSquaredTotal > 0 {
		rSquared = 1.0 - (sumSquaredError / sumSquaredTotal)
	}

	// Calculate RMSE
	rmse := math.Sqrt(sumSquaredError / float64(n))

	// Calculate acceleration metrics
	accelX := polyX.Acceleration()
	accelY := polyY.Acceleration()
	avgAccel := math.Sqrt(accelX*accelX + accelY*accelY)
	maxAccel := avgAccel // For quadratic, acceleration is constant

	return TrackFitQuality{
		RSquared:     rSquared,
		RMSE:         rmse,
		MaxDeviation: maxDeviation,
		AvgAccel:     avgAccel,
		MaxAccel:     maxAccel,
	}
}

// DefaultCurveFitConfig returns reasonable default values
func DefaultCurveFitConfig() CurveFitConfig {
	return CurveFitConfig{
		MinRSquared:     0.85, // Good fit required
		MaxRMSE:         3.0,  // Low error
		MaxDeviation:    8.0,  // No large outliers
		MaxAcceleration: 2.0,  // Physically plausible motion
	}
}

// FilterTracksByCurveFit filters tracks based on curve fit quality
func FilterTracksByCurveFit(tracks []*Track, config CurveFitConfig) []*Track {
	var filtered []*Track

	for _, track := range tracks {
		quality := EvaluateTrackFit(track)

		// Check all criteria
		if quality.RSquared >= config.MinRSquared &&
			quality.RMSE <= config.MaxRMSE &&
			quality.MaxDeviation <= config.MaxDeviation &&
			quality.AvgAccel <= config.MaxAcceleration {
			filtered = append(filtered, track)
		}
	}

	return filtered
}

// FilterTracksHybrid combines curve fit filtering with length requirements
func FilterTracksHybrid(tracks []*Track, minLength int, curveFitConfig CurveFitConfig) []*Track {
	var filtered []*Track

	for _, track := range tracks {
		// First check length
		if len(track.Points) < minLength {
			continue
		}

		// Then check curve fit quality
		quality := EvaluateTrackFit(track)

		if quality.RSquared >= curveFitConfig.MinRSquared &&
			quality.RMSE <= curveFitConfig.MaxRMSE &&
			quality.MaxDeviation <= curveFitConfig.MaxDeviation &&
			quality.AvgAccel <= curveFitConfig.MaxAcceleration {

			// Store the fitted polynomials in the track
			polyX, polyY, err := FitQuadratic(track.Points)
			if err == nil {
				track.PolyX = polyX
				track.PolyY = polyY
			}

			filtered = append(filtered, track)
		}
	}

	return filtered
}
