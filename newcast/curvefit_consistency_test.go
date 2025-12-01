package newcast

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestCurveFitParameterConsistency tests parameter configurations on multiple datasets
// to verify that the recommended settings work consistently
func TestCurveFitParameterConsistency(t *testing.T) {
	dataDir := "../rainfall_data_uk"
	dateDirs, err := os.ReadDir(dataDir)
	if err != nil {
		t.Skip("UK rainfall data not found")
		return
	}

	// Test configurations to compare
	configs := []struct {
		name   string
		config CurveFitConfig
	}{
		{"Very Strict", CurveFitConfig{
			MinRSquared:     0.95,
			MaxRMSE:         2.0,
			MaxDeviation:    5.0,
			MaxAcceleration: 1.0,
		}},
		{"Strict (Recommended)", CurveFitConfig{
			MinRSquared:     0.90,
			MaxRMSE:         3.0,
			MaxDeviation:    7.0,
			MaxAcceleration: 1.5,
		}},
		{"Default", DefaultCurveFitConfig()},
	}

	fmt.Println("=== Testing Parameter Consistency Across Datasets ===")

	// Test each date directory
	for _, dateDir := range dateDirs {
		if !dateDir.IsDir() {
			continue
		}

		dirPath := filepath.Join(dataDir, dateDir.Name())
		var files []string
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".png" {
				files = append(files, filepath.Join(dirPath, entry.Name()))
			}
		}

		if len(files) < 5 {
			continue
		}

		sort.Strings(files)
		files = files[:min(10, len(files))]

		fmt.Printf("Dataset: %s (%d files)\n", dateDir.Name(), len(files))

		// Get raw tracks
		configRaw := ProcessConfig{
			MaxFeatures:      2000,
			MinTrackLength:   4,
		}

		rawTracks, width, height, err := ProcessFilesToTracks(files, configRaw)
		if err != nil {
			t.Errorf("Failed to process %s: %v", dateDir.Name(), err)
			continue
		}

		fmt.Printf("  Raw tracks: %d\n", len(rawTracks))
		if len(rawTracks) == 0 {
			fmt.Println("  Skipping - no tracks found")
			continue
		}

		// Test each configuration
		fmt.Printf("  %-22s | %8s | %8s | %8s | %8s\n", "Config", "Tracks", "Retain%", "Avg R²", "Smooth")
		fmt.Println("  -----------------------+----------+----------+----------+----------")

		for _, cfg := range configs {
			filtered := FilterTracksByCurveFit(rawTracks, cfg.config)
			retentionRate := 100.0 * float64(len(filtered)) / float64(len(rawTracks))

			// Calculate quality metrics
			var sumRSquared float64
			for _, track := range filtered {
				quality := EvaluateTrackFit(track)
				sumRSquared += quality.RSquared
			}
			avgRSquared := 0.0
			if len(filtered) > 0 {
				avgRSquared = sumRSquared / float64(len(filtered))
			}

			// Calculate smoothness
			gridSize := 64
			smoothness := calculateVectorFieldSmoothness(filtered, width, height, gridSize)

			fmt.Printf("  %-22s | %8d | %7.1f%% | %8.3f | %8.3f\n",
				cfg.name, len(filtered), retentionRate, avgRSquared, smoothness)
		}
		fmt.Println()
	}
}

// TestRecommendedConfigOnAllData runs the recommended configuration on all available data
// and generates detailed statistics
func TestRecommendedConfigOnAllData(t *testing.T) {
	dataDir := "../rainfall_data_uk"
	dateDirs, err := os.ReadDir(dataDir)
	if err != nil {
		t.Skip("UK rainfall data not found")
		return
	}

	// Recommended configuration
	recommendedConfig := CurveFitConfig{
		MinRSquared:     0.90,
		MaxRMSE:         3.0,
		MaxDeviation:    7.0,
		MaxAcceleration: 1.5,
	}

	fmt.Println("=== Testing Recommended Configuration on All Data ===")
	fmt.Printf("Config: MinR²=%.2f, MaxRMSE=%.1f, MaxDev=%.1f, MaxAccel=%.1f\n\n",
		recommendedConfig.MinRSquared, recommendedConfig.MaxRMSE,
		recommendedConfig.MaxDeviation, recommendedConfig.MaxAcceleration)

	type datasetStats struct {
		name           string
		rawTracks      int
		filteredTracks int
		retentionRate  float64
		avgRSquared    float64
		avgRMSE        float64
		avgAccel       float64
		coverage       float64
		smoothness     float64
	}

	var allStats []datasetStats

	// Process each date directory
	for _, dateDir := range dateDirs {
		if !dateDir.IsDir() {
			continue
		}

		dirPath := filepath.Join(dataDir, dateDir.Name())
		var files []string
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".png" {
				files = append(files, filepath.Join(dirPath, entry.Name()))
			}
		}

		if len(files) < 5 {
			continue
		}

		sort.Strings(files)
		files = files[:min(10, len(files))]

		// Get raw tracks
		configRaw := ProcessConfig{
			MaxFeatures:      2000,
			MinTrackLength:   4,
		}

		rawTracks, width, height, err := ProcessFilesToTracks(files, configRaw)
		if err != nil || len(rawTracks) == 0 {
			continue
		}

		// Apply recommended filter
		filtered := FilterTracksByCurveFit(rawTracks, recommendedConfig)

		// Calculate statistics
		stats := datasetStats{
			name:           dateDir.Name(),
			rawTracks:      len(rawTracks),
			filteredTracks: len(filtered),
			retentionRate:  100.0 * float64(len(filtered)) / float64(len(rawTracks)),
		}

		if len(filtered) > 0 {
			var sumRSquared, sumRMSE, sumAccel float64
			for _, track := range filtered {
				quality := EvaluateTrackFit(track)
				sumRSquared += quality.RSquared
				sumRMSE += quality.RMSE
				sumAccel += quality.AvgAccel
			}
			stats.avgRSquared = sumRSquared / float64(len(filtered))
			stats.avgRMSE = sumRMSE / float64(len(filtered))
			stats.avgAccel = sumAccel / float64(len(filtered))

			gridSize := 64
			stats.coverage, _ = calculateCoverage(filtered, width, height, gridSize)
			stats.smoothness = calculateVectorFieldSmoothness(filtered, width, height, gridSize)
		}

		allStats = append(allStats, stats)
	}

	if len(allStats) == 0 {
		t.Skip("No datasets processed")
		return
	}

	// Print results
	fmt.Printf("%-12s | %8s | %8s | %8s | %8s | %8s | %8s | %8s\n",
		"Dataset", "Raw", "Filtered", "Retain%", "Avg R²", "RMSE", "Coverage", "Smooth")
	fmt.Println("-------------+----------+----------+----------+----------+----------+----------+----------")

	for _, stats := range allStats {
		fmt.Printf("%-12s | %8d | %8d | %7.1f%% | %8.3f | %8.2f | %7.1f%% | %8.3f\n",
			stats.name, stats.rawTracks, stats.filteredTracks, stats.retentionRate,
			stats.avgRSquared, stats.avgRMSE, stats.coverage, stats.smoothness)
	}

	// Calculate overall statistics
	fmt.Println("\n=== Overall Statistics ===")
	var totalRaw, totalFiltered int
	var sumRetention, sumRSquared, sumRMSE, sumCoverage, sumSmoothness float64

	for _, stats := range allStats {
		totalRaw += stats.rawTracks
		totalFiltered += stats.filteredTracks
		sumRetention += stats.retentionRate
		sumRSquared += stats.avgRSquared
		sumRMSE += stats.avgRMSE
		sumCoverage += stats.coverage
		sumSmoothness += stats.smoothness
	}

	n := float64(len(allStats))
	fmt.Printf("Total raw tracks: %d\n", totalRaw)
	fmt.Printf("Total filtered tracks: %d\n", totalFiltered)
	fmt.Printf("Average retention rate: %.1f%%\n", sumRetention/n)
	fmt.Printf("Average R²: %.3f\n", sumRSquared/n)
	fmt.Printf("Average RMSE: %.2f pixels\n", sumRMSE/n)
	fmt.Printf("Average coverage: %.1f%%\n", sumCoverage/n)
	fmt.Printf("Average smoothness: %.3f\n", sumSmoothness/n)

	// Consistency check
	fmt.Println("\n=== Consistency Check ===")
	var retentionVariance, rSquaredVariance, smoothnessVariance float64
	avgRetention := sumRetention / n
	avgRSquared := sumRSquared / n
	avgSmoothness := sumSmoothness / n

	for _, stats := range allStats {
		retentionVariance += (stats.retentionRate - avgRetention) * (stats.retentionRate - avgRetention)
		rSquaredVariance += (stats.avgRSquared - avgRSquared) * (stats.avgRSquared - avgRSquared)
		smoothnessVariance += (stats.smoothness - avgSmoothness) * (stats.smoothness - avgSmoothness)
	}

	retentionStdDev := 0.0
	rSquaredStdDev := 0.0
	smoothnessStdDev := 0.0
	if n > 1 {
		retentionStdDev = (retentionVariance / (n - 1))
		rSquaredStdDev = (rSquaredVariance / (n - 1))
		smoothnessStdDev = (smoothnessVariance / (n - 1))
	}

	fmt.Printf("Retention rate std dev: %.2f%%\n", retentionStdDev)
	fmt.Printf("R² std dev: %.4f\n", rSquaredStdDev)
	fmt.Printf("Smoothness std dev: %.4f\n", smoothnessStdDev)

	if retentionStdDev < 5.0 && rSquaredStdDev < 0.01 && smoothnessStdDev < 0.05 {
		fmt.Println("\n✓ Configuration shows GOOD consistency across datasets")
	} else {
		fmt.Println("\n⚠ Configuration shows some variability across datasets")
	}
}
