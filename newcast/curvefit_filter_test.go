package newcast

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

// TestCurveFitFiltering tests the curve-fit based filtering
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
	sort.Strings(files)

	if len(files) < 2 {
		t.Skip("Need at least 2 files")
		return
	}

	files = files[:min(10, len(files))]

	// Manually generate raw tracks
	tracker, err := NewTracker(1000) // MaxFeatures
	if err != nil {
		t.Fatal(err)
	}
	defer tracker.Close()

	for _, imgPath := range files {
		img, err := loadImageAsGrayscale(imgPath)
		if err != nil {
			t.Fatal(err)
		}
		// defer img.Close() // In a loop, defer will stack up. Better to close manually or rely on GoCV finalizers (if any). 
		// Ideally close manually.
		
		if err := tracker.AddImage(img); err != nil {
			img.Close()
			t.Fatal(err)
		}
		img.Close()
	}

	allTracks := tracker.GetTracks()
	var rawTracks []*Track
	minTrackLength := 4
	for _, track := range allTracks {
		if len(track.Points) >= minTrackLength {
			rawTracks = append(rawTracks, track)
		}
	}
	
	fmt.Printf("Raw tracks (min length %d): %d\n", minTrackLength, len(rawTracks))

	// Apply curve-fit filter with default config
	curveFitConfig := DefaultCurveFitConfig()
	curveFitTracks := FilterTracksByCurveFit(rawTracks, curveFitConfig)
	fmt.Printf("After curve-fit filter (default): %d (%.1f%% of raw)\n",
		len(curveFitTracks), 100.0*float64(len(curveFitTracks))/float64(len(rawTracks)))

	// Analyze the quality of filtered tracks
	fmt.Println("\n=== Quality Analysis ===")

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
