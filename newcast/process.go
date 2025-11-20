package newcast

import (
	"fmt"
	"sort"

	"gocv.io/x/gocv"
)

// ProcessConfig holds all the parameters for processing images into tracks.
type ProcessConfig struct {
	MaxFeatures      int
	Smoothness       float64
	FilterType       string // "smoothness", "density", "max_angle", or "curvefit"
	MaxAngle         float64
	GridCellSize     int
	MinTracksPerCell int
	MaxTracksPerCell int
	MinTrackLength   int
	BlurSigma        float64 // Gaussian blur sigma for flow grid smoothing (0 = no blur)
	UseFittedPoints  bool    // Use polynomial-fitted points instead of raw tracked points

	// Curve-fit filtering parameters (used when FilterType = "curvefit")
	MinRSquared     float64 // Minimum R² for polynomial fit (e.g., 0.85)
	MaxRMSE         float64 // Maximum RMSE in pixels (e.g., 3.0)
	MaxDeviation    float64 // Maximum deviation from curve in pixels (e.g., 8.0)
	MaxAcceleration float64 // Maximum acceleration in pixels/frame² (e.g., 2.0)
}

// ProcessFilesToTracks takes a list of file paths and processing configuration,
// and returns the filtered tracks, along with the width and height of the images.
func ProcessFilesToTracks(filePaths []string, config ProcessConfig) ([]*Track, int, int, error) {
	// Sort the input files to ensure consistent processing order
	sort.Strings(filePaths)

	tracker, err := NewTracker(config.MaxFeatures)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error creating tracker: %w", err)
	}
	defer tracker.Close()

	var width, height int
	for i, imgPath := range filePaths {
		img, err := loadImageAsGrayscale(imgPath)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("error loading image %s: %w", imgPath, err)
		}
		defer img.Close()

		if i == 0 {
			width = img.Cols()
			height = img.Rows()
		}

		if err := tracker.AddImage(img); err != nil {
			return nil, 0, 0, fmt.Errorf("error adding image %s: %w", imgPath, err)
		}
	}

	allTracks := tracker.GetTracks()

	var longTracks []*Track
	for _, track := range allTracks {
		if len(track.Points) >= config.MinTrackLength {
			longTracks = append(longTracks, track)
		}
	}

	var filteredTracks []*Track
	switch config.FilterType {
	case "density":
		smoothTracks := FilterTracksBySmoothness(longTracks, config.Smoothness)
		filteredTracks = FilterTracksByDensityAndSmoothness(smoothTracks, config.GridCellSize, config.MinTracksPerCell, config.MaxTracksPerCell)
	case "max_angle":
		filteredTracks = FilterTracksByMaxAngleChange(longTracks, config.MaxAngle)
	case "curvefit":
		// Use curve-fit filtering with config parameters
		curveFitConfig := CurveFitConfig{
			MinRSquared:     config.MinRSquared,
			MaxRMSE:         config.MaxRMSE,
			MaxDeviation:    config.MaxDeviation,
			MaxAcceleration: config.MaxAcceleration,
		}
		// Use defaults if not specified
		if curveFitConfig.MinRSquared == 0 {
			defaults := DefaultCurveFitConfig()
			curveFitConfig = defaults
		}
		filteredTracks = FilterTracksByCurveFit(longTracks, curveFitConfig)
	default: // "smoothness"
		filteredTracks = FilterTracksBySmoothness(longTracks, config.Smoothness)
	}

	return filteredTracks, width, height, nil
}

// loadImageAsGrayscale loads an image from the given path and converts it to a grayscale gocv.Mat.
func loadImageAsGrayscale(path string) (gocv.Mat, error) {
	imgMat := gocv.IMRead(path, gocv.IMReadGrayScale)
	if imgMat.Empty() {
		return gocv.NewMat(), fmt.Errorf("failed to read image %s", path)
	}
	return imgMat, nil
}
