package newcast

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestCurveFitParameterSweep runs a comprehensive parameter sweep on UK rainfall data
func TestCurveFitParameterSweep(t *testing.T) {
	// Load UK rainfall data
	dataDir := "../rainfall_data_uk"
	dateDirs, err := os.ReadDir(dataDir)
	if err != nil {
		t.Skip("UK rainfall data not found")
		return
	}

	// Use the first date directory that has files
	var files []string
	for _, dateDir := range dateDirs {
		if !dateDir.IsDir() {
			continue
		}
		dirPath := filepath.Join(dataDir, dateDir.Name())
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".png" {
				files = append(files, filepath.Join(dirPath, entry.Name()))
			}
		}
		if len(files) >= 5 {
			break
		}
	}

	if len(files) < 5 {
		t.Skip("Need at least 5 files for testing")
		return
	}

	sort.Strings(files)
	files = files[:min(10, len(files))] // Use up to 10 files

	fmt.Printf("Testing with %d files from UK rainfall data\n", len(files))
	fmt.Printf("Files: %v\n\n", files)

	// Get raw tracks with minimal filtering
	configRaw := ProcessConfig{
		MaxFeatures:      2000,
		MinTrackLength:   4,
	}

	rawTracks, width, height, err := ProcessFilesToTracks(files, configRaw)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Image dimensions: %dx%d\n", width, height)
	fmt.Printf("Raw tracks (length >= 4): %d\n\n", len(rawTracks))

	if len(rawTracks) == 0 {
		t.Skip("No tracks found in data")
		return
	}

	// Analyze raw track quality distribution
	fmt.Println("=== Raw Track Quality Distribution ===")
	analyzeQualityDistribution(rawTracks)

	// Test different parameter configurations
	fmt.Println("\n=== Testing Parameter Configurations ===")

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
		{"Strict", CurveFitConfig{
			MinRSquared:     0.90,
			MaxRMSE:         3.0,
			MaxDeviation:    7.0,
			MaxAcceleration: 1.5,
		}},
		{"Default", DefaultCurveFitConfig()},
		{"Moderate", CurveFitConfig{
			MinRSquared:     0.80,
			MaxRMSE:         4.0,
			MaxDeviation:    10.0,
			MaxAcceleration: 2.5,
		}},
		{"Relaxed", CurveFitConfig{
			MinRSquared:     0.75,
			MaxRMSE:         5.0,
			MaxDeviation:    12.0,
			MaxAcceleration: 3.0,
		}},
		{"Very Relaxed", CurveFitConfig{
			MinRSquared:     0.70,
			MaxRMSE:         6.0,
			MaxDeviation:    15.0,
			MaxAcceleration: 4.0,
		}},
		{"Minimal", CurveFitConfig{
			MinRSquared:     0.60,
			MaxRMSE:         8.0,
			MaxDeviation:    20.0,
			MaxAcceleration: 5.0,
		}},
	}

	// Create output directory for visualizations
	outDir := "../test_output/curvefit_params"
	os.MkdirAll(outDir, 0755)

	var results []parameterTestResult
	for _, cfg := range configs {
		result := testConfiguration(cfg.name, cfg.config, rawTracks, width, height)
		results = append(results, result)

		fmt.Printf("\n%s:\n", cfg.name)
		fmt.Printf("  Tracks retained: %d / %d (%.1f%%)\n",
			result.numTracks, len(rawTracks), result.retentionRate)
		fmt.Printf("  Quality metrics:\n")
		fmt.Printf("    R² (avg/min): %.3f / %.3f\n", result.avgRSquared, result.minRSquared)
		fmt.Printf("    RMSE (avg/max): %.2f / %.2f pixels\n", result.avgRMSE, result.maxRMSE)
		fmt.Printf("    Max deviation (avg/max): %.2f / %.2f pixels\n",
			result.avgMaxDev, result.maxMaxDev)
		fmt.Printf("    Acceleration (avg/max): %.2f / %.2f px/frame²\n",
			result.avgAccel, result.maxAccel)
		fmt.Printf("  Flow field metrics:\n")
		fmt.Printf("    Coverage: %.1f%% of cells have vectors\n", result.coverage)
		fmt.Printf("    Avg vectors per cell: %.2f\n", result.avgVectorsPerCell)
		fmt.Printf("    Smoothness score: %.3f\n", result.smoothnessScore)

		// Generate visualization
		vizPath := filepath.Join(outDir, fmt.Sprintf("%s.png", sanitizeFilename(cfg.name)))
		if err := visualizeFlowField(result.tracks, width, height, vizPath); err != nil {
			fmt.Printf("  Warning: failed to create visualization: %v\n", err)
		} else {
			fmt.Printf("  Visualization saved to: %s\n", vizPath)
		}
	}

	// Summary comparison
	fmt.Println("\n=== Summary Comparison ===")
	fmt.Printf("%-15s | %8s | %8s | %8s | %8s | %8s\n",
		"Config", "Tracks", "Retain%", "Coverage", "Smooth", "Avg R²")
	fmt.Println("----------------+----------+----------+----------+----------+----------")
	for _, r := range results {
		fmt.Printf("%-15s | %8d | %7.1f%% | %7.1f%% | %8.3f | %8.3f\n",
			r.name, r.numTracks, r.retentionRate, r.coverage, r.smoothnessScore, r.avgRSquared)
	}

	// Recommendations
	fmt.Println("\n=== Recommendations ===")
	recommendBestConfig(results)
}

type parameterTestResult struct {
	name              string
	config            CurveFitConfig
	tracks            []*Track
	numTracks         int
	retentionRate     float64
	avgRSquared       float64
	minRSquared       float64
	avgRMSE           float64
	maxRMSE           float64
	avgMaxDev         float64
	maxMaxDev         float64
	avgAccel          float64
	maxAccel          float64
	coverage          float64 // % of grid cells with vectors
	avgVectorsPerCell float64
	smoothnessScore   float64 // measure of vector field smoothness
}

func testConfiguration(name string, config CurveFitConfig, rawTracks []*Track, width, height int) parameterTestResult {
	filtered := FilterTracksByCurveFit(rawTracks, config)

	result := parameterTestResult{
		name:          name,
		config:        config,
		tracks:        filtered,
		numTracks:     len(filtered),
		retentionRate: 100.0 * float64(len(filtered)) / float64(len(rawTracks)),
	}

	if len(filtered) == 0 {
		return result
	}

	// Calculate quality metrics
	var sumRSquared, sumRMSE, sumMaxDev, sumAccel float64
	result.minRSquared = 1.0

	for _, track := range filtered {
		quality := EvaluateTrackFit(track)
		sumRSquared += quality.RSquared
		sumRMSE += quality.RMSE
		sumMaxDev += quality.MaxDeviation
		sumAccel += quality.AvgAccel

		if quality.RSquared < result.minRSquared {
			result.minRSquared = quality.RSquared
		}
		if quality.RMSE > result.maxRMSE {
			result.maxRMSE = quality.RMSE
		}
		if quality.MaxDeviation > result.maxMaxDev {
			result.maxMaxDev = quality.MaxDeviation
		}
		if quality.AvgAccel > result.maxAccel {
			result.maxAccel = quality.AvgAccel
		}
	}

	n := float64(len(filtered))
	result.avgRSquared = sumRSquared / n
	result.avgRMSE = sumRMSE / n
	result.avgMaxDev = sumMaxDev / n
	result.avgAccel = sumAccel / n

	// Calculate flow field metrics
	gridSize := 64
	result.coverage, result.avgVectorsPerCell = calculateCoverage(filtered, width, height, gridSize)
	result.smoothnessScore = calculateVectorFieldSmoothness(filtered, width, height, gridSize)

	return result
}

func analyzeQualityDistribution(tracks []*Track) {
	if len(tracks) == 0 {
		fmt.Println("No tracks to analyze")
		return
	}

	var rSquareds, rmses, maxDevs, accels []float64
	for _, track := range tracks {
		quality := EvaluateTrackFit(track)
		rSquareds = append(rSquareds, quality.RSquared)
		rmses = append(rmses, quality.RMSE)
		maxDevs = append(maxDevs, quality.MaxDeviation)
		accels = append(accels, quality.AvgAccel)
	}

	fmt.Printf("R² distribution:\n")
	printPercentiles("  ", rSquareds)
	fmt.Printf("RMSE distribution (pixels):\n")
	printPercentiles("  ", rmses)
	fmt.Printf("Max deviation distribution (pixels):\n")
	printPercentiles("  ", maxDevs)
	fmt.Printf("Acceleration distribution (px/frame²):\n")
	printPercentiles("  ", accels)
}

func printPercentiles(prefix string, values []float64) {
	if len(values) == 0 {
		return
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	percentiles := []int{0, 10, 25, 50, 75, 90, 95, 100}
	for _, p := range percentiles {
		idx := (len(sorted) - 1) * p / 100
		fmt.Printf("%sp%d: %.3f\n", prefix, p, sorted[idx])
	}
}

func calculateCoverage(tracks []*Track, width, height, gridSize int) (coverage, avgVectorsPerCell float64) {
	gridW := (width + gridSize - 1) / gridSize
	gridH := (height + gridSize - 1) / gridSize

	cellCounts := make(map[int]int)

	for _, track := range tracks {
		if len(track.Points) < 2 {
			continue
		}
		// Use the first point to determine grid cell
		p := track.Points[0]
		gx := int(p.X) / gridSize
		gy := int(p.Y) / gridSize
		if gx >= 0 && gx < gridW && gy >= 0 && gy < gridH {
			cellIdx := gy*gridW + gx
			cellCounts[cellIdx]++
		}
	}

	totalCells := gridW * gridH
	coverage = 100.0 * float64(len(cellCounts)) / float64(totalCells)

	if len(cellCounts) > 0 {
		totalVectors := 0
		for _, count := range cellCounts {
			totalVectors += count
		}
		avgVectorsPerCell = float64(totalVectors) / float64(len(cellCounts))
	}

	return
}

func calculateVectorFieldSmoothness(tracks []*Track, width, height, gridSize int) float64 {
	// Calculate average vector per grid cell
	gridW := (width + gridSize - 1) / gridSize
	gridH := (height + gridSize - 1) / gridSize

	type Vector struct {
		dx, dy float64
		count  int
	}

	grid := make(map[int]*Vector)

	for _, track := range tracks {
		if len(track.Points) < 2 {
			continue
		}
		p0 := track.Points[0]
		p1 := track.Points[len(track.Points)-1]
		dx := (p1.X - p0.X) / float64(len(track.Points)-1)
		dy := (p1.Y - p0.Y) / float64(len(track.Points)-1)

		gx := int(p0.X) / gridSize
		gy := int(p0.Y) / gridSize
		if gx >= 0 && gx < gridW && gy >= 0 && gy < gridH {
			cellIdx := gy*gridW + gx
			if grid[cellIdx] == nil {
				grid[cellIdx] = &Vector{}
			}
			grid[cellIdx].dx += dx
			grid[cellIdx].dy += dy
			grid[cellIdx].count++
		}
	}

	// Average vectors in each cell
	for _, v := range grid {
		if v.count > 0 {
			v.dx /= float64(v.count)
			v.dy /= float64(v.count)
		}
	}

	// Calculate smoothness as inverse of average difference between neighboring cells
	var totalDiff float64
	var comparisons int

	for gy := 0; gy < gridH; gy++ {
		for gx := 0; gx < gridW; gx++ {
			cellIdx := gy*gridW + gx
			v := grid[cellIdx]
			if v == nil {
				continue
			}

			// Compare with right neighbor
			if gx+1 < gridW {
				rightIdx := gy*gridW + (gx + 1)
				if vr := grid[rightIdx]; vr != nil {
					diff := math.Sqrt((v.dx-vr.dx)*(v.dx-vr.dx) + (v.dy-vr.dy)*(v.dy-vr.dy))
					totalDiff += diff
					comparisons++
				}
			}

			// Compare with bottom neighbor
			if gy+1 < gridH {
				bottomIdx := (gy+1)*gridW + gx
				if vb := grid[bottomIdx]; vb != nil {
					diff := math.Sqrt((v.dx-vb.dx)*(v.dx-vb.dx) + (v.dy-vb.dy)*(v.dy-vb.dy))
					totalDiff += diff
					comparisons++
				}
			}
		}
	}

	if comparisons == 0 {
		return 0
	}

	avgDiff := totalDiff / float64(comparisons)
	// Return inverse (lower difference = higher smoothness)
	return 1.0 / (1.0 + avgDiff)
}

func visualizeFlowField(tracks []*Track, width, height int, outputPath string) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Black background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}

	// Draw tracks
	for _, track := range tracks {
		if len(track.Points) < 2 {
			continue
		}

		// Color based on track quality
		quality := EvaluateTrackFit(track)
		c := colorForQuality(quality.RSquared)

		// Draw track points
		for i := 0; i < len(track.Points)-1; i++ {
			p0 := track.Points[i]
			p1 := track.Points[i+1]
			drawLineRGBA(img, int(p0.X), int(p0.Y), int(p1.X), int(p1.Y), c)
		}

		// Draw arrow at end
		if len(track.Points) >= 2 {
			p0 := track.Points[len(track.Points)-2]
			p1 := track.Points[len(track.Points)-1]
			drawArrow(img, int(p0.X), int(p0.Y), int(p1.X), int(p1.Y), c)
		}
	}

	// Save image
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

func colorForQuality(rSquared float64) color.RGBA {
	// Green for high quality, yellow for medium, red for low
	if rSquared >= 0.9 {
		return color.RGBA{0, 255, 0, 255} // Green
	} else if rSquared >= 0.8 {
		return color.RGBA{128, 255, 0, 255} // Yellow-green
	} else if rSquared >= 0.7 {
		return color.RGBA{255, 255, 0, 255} // Yellow
	} else {
		return color.RGBA{255, 128, 0, 255} // Orange
	}
}

func drawLineRGBA(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy

	for {
		if x0 >= 0 && x0 < img.Bounds().Dx() && y0 >= 0 && y0 < img.Bounds().Dy() {
			img.Set(x0, y0, c)
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func drawArrow(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	// Draw arrowhead
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 1 {
		return
	}

	// Normalize
	dx /= length
	dy /= length

	// Arrow size
	arrowLen := 5.0
	arrowWidth := 3.0

	// Perpendicular vector
	px := -dy
	py := dx

	// Arrow points
	ax1 := int(float64(x1) - dx*arrowLen + px*arrowWidth)
	ay1 := int(float64(y1) - dy*arrowLen + py*arrowWidth)
	ax2 := int(float64(x1) - dx*arrowLen - px*arrowWidth)
	ay2 := int(float64(y1) - dy*arrowLen - py*arrowWidth)

	drawLineRGBA(img, x1, y1, ax1, ay1, c)
	drawLineRGBA(img, x1, y1, ax2, ay2, c)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sanitizeFilename(s string) string {
	// Replace spaces with underscores and convert to lowercase
	result := ""
	for _, r := range s {
		if r == ' ' {
			result += "_"
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result += string(r)
		}
	}
	return result
}

func recommendBestConfig(results []parameterTestResult) {
	if len(results) == 0 {
		fmt.Println("No results to analyze")
		return
	}

	// Find configs with good balance of coverage and smoothness
	type scoredResult struct {
		result *parameterTestResult
		score  float64
	}

	var scored []scoredResult
	for i := range results {
		r := &results[i]
		if r.numTracks == 0 {
			continue
		}

		// Composite score: balance coverage, smoothness, and quality
		// We want:
		// - Good coverage (at least 20%)
		// - High smoothness
		// - High R² (quality)
		// - Reasonable retention (not too strict, not too loose)

		coverageScore := math.Min(r.coverage/50.0, 1.0) // Target 50% coverage
		smoothnessScore := r.smoothnessScore
		qualityScore := r.avgRSquared
		retentionScore := 1.0 - math.Abs(r.retentionRate-30.0)/30.0 // Target ~30% retention
		if retentionScore < 0 {
			retentionScore = 0
		}

		// Weighted combination
		score := coverageScore*0.3 + smoothnessScore*0.3 + qualityScore*0.25 + retentionScore*0.15

		scored = append(scored, scoredResult{result: r, score: score})
	}

	if len(scored) == 0 {
		fmt.Println("No valid configurations found")
		return
	}

	// Sort by score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	fmt.Println("\nTop 3 recommended configurations:")
	for i := 0; i < min(3, len(scored)); i++ {
		r := scored[i].result
		fmt.Printf("\n%d. %s (score: %.3f)\n", i+1, r.name, scored[i].score)
		fmt.Printf("   MinRSquared: %.2f, MaxRMSE: %.1f, MaxDeviation: %.1f, MaxAcceleration: %.1f\n",
			r.config.MinRSquared, r.config.MaxRMSE, r.config.MaxDeviation, r.config.MaxAcceleration)
		fmt.Printf("   Tracks: %d (%.1f%%), Coverage: %.1f%%, Smoothness: %.3f\n",
			r.numTracks, r.retentionRate, r.coverage, r.smoothnessScore)
	}
}
