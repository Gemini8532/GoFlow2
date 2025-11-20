package newcast

import (
	"fmt"
	"os"
	"testing"
)

// TestCurveFitFiltering compares curve-fit based filtering with current ad-hoc filtering
func TestCurveFitFiltering(t *testing.T) {
	// Load real rainfall data
	matches, err := os.ReadDir("../rainfall_data")
	if err != nil || len(matches) == 0 {
		t.Skip("Rainfall data not found")
		return
	}

	var files []string
	for _, entry := range matches {
		if !entry.IsDir() {
			files = append(files, "../rainfall_data/"+entry.Name())
		}
	}

	if len(files) < 2 {
		t.Skip("Need at least 2 files")
		return
	}

	files = files[:min(10, len(files))]

	// Process with minimal filtering to get raw tracks
	configRaw := ProcessConfig{
		MaxFeatures:      1000,
		Smoothness:       10.0, // Very high = accept everything
		FilterType:       "smoothness",
		MaxAngle:         10.0,
		GridCellSize:     64,
		MinTracksPerCell: 1,
		MaxTracksPerCell: 100,
		MinTrackLength:   4,
	}

	rawTracks, _, _, err := ProcessFilesToTracks(files, configRaw)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Raw tracks (min length 4): %d\n", len(rawTracks))

	// Apply current smoothness filter
	smoothnessTracks := FilterTracksBySmoothness(rawTracks, 0.5)
	fmt.Printf("After smoothness filter (0.5): %d (%.1f%% of raw)\n",
		len(smoothnessTracks), 100.0*float64(len(smoothnessTracks))/float64(len(rawTracks)))

	// Apply curve-fit filter with default config
	curveFitConfig := DefaultCurveFitConfig()
	curveFitTracks := FilterTracksByCurveFit(rawTracks, curveFitConfig)
	fmt.Printf("After curve-fit filter (default): %d (%.1f%% of raw)\n",
		len(curveFitTracks), 100.0*float64(len(curveFitTracks))/float64(len(rawTracks)))

	// Analyze the quality of filtered tracks
	fmt.Println("\n=== Quality Analysis ===")

	// Smoothness-filtered tracks
	fmt.Println("Smoothness-filtered tracks:")
	analyzeTrackQuality(smoothnessTracks)

	// Curve-fit filtered tracks
	fmt.Println("\nCurve-fit filtered tracks:")
	analyzeTrackQuality(curveFitTracks)

	// Try different curve-fit thresholds
	fmt.Println("\n=== Testing Different Curve-Fit Thresholds ===")

	configs := []struct {
		name   string
		config CurveFitConfig
	}{
		{"Strict", CurveFitConfig{MinRSquared: 0.95, MaxRMSE: 2.0, MaxDeviation: 5.0, MaxAcceleration: 1.0}},
		{"Default", DefaultCurveFitConfig()},
		{"Relaxed", CurveFitConfig{MinRSquared: 0.75, MaxRMSE: 5.0, MaxDeviation: 12.0, MaxAcceleration: 3.0}},
	}

	for _, cfg := range configs {
		filtered := FilterTracksByCurveFit(rawTracks, cfg.config)
		fmt.Printf("%s: %d tracks (%.1f%%)\n", cfg.name, len(filtered),
			100.0*float64(len(filtered))/float64(len(rawTracks)))
	}
}

func analyzeTrackQuality(tracks []*Track) {
	if len(tracks) == 0 {
		fmt.Println("  No tracks to analyze")
		return
	}

	var sumRSquared, sumRMSE, sumMaxDev, sumAccel float64
	var minRSquared, maxRMSE, maxMaxDev, maxAccel float64 = 1.0, 0.0, 0.0, 0.0

	for _, track := range tracks {
		quality := EvaluateTrackFit(track)

		sumRSquared += quality.RSquared
		sumRMSE += quality.RMSE
		sumMaxDev += quality.MaxDeviation
		sumAccel += quality.AvgAccel

		if quality.RSquared < minRSquared {
			minRSquared = quality.RSquared
		}
		if quality.RMSE > maxRMSE {
			maxRMSE = quality.RMSE
		}
		if quality.MaxDeviation > maxMaxDev {
			maxMaxDev = quality.MaxDeviation
		}
		if quality.AvgAccel > maxAccel {
			maxAccel = quality.AvgAccel
		}
	}

	n := float64(len(tracks))
	fmt.Printf("  R² (avg/min): %.3f / %.3f\n", sumRSquared/n, minRSquared)
	fmt.Printf("  RMSE (avg/max): %.2f / %.2f pixels\n", sumRMSE/n, maxRMSE)
	fmt.Printf("  Max deviation (avg/max): %.2f / %.2f pixels\n", sumMaxDev/n, maxMaxDev)
	fmt.Printf("  Acceleration (avg/max): %.2f / %.2f px/frame²\n", sumAccel/n, maxAccel)
}
