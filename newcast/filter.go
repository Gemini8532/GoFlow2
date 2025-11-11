package newcast

import (
	"image"
	"math"
	"sort"

	"gocv.io/x/gocv"
)

// calculateSmoothnessMetric calculates the smoothness of a track.
// A lower value means a smoother track. Returns a large value if smoothness
// cannot be calculated.
func calculateSmoothnessMetric(track *Track) float64 {
	if len(track.Points) < 3 {
		return 0.0 // Consider tracks with < 3 points as perfectly smooth
	}

	var totalAngleChange float64
	numSegments := 0

	for i := 0; i < len(track.Points)-2; i++ {
		p1 := track.Points[i].Vec
		p2 := track.Points[i+1].Vec
		p3 := track.Points[i+2].Vec

		v1 := gocv.Point2f{X: p2.X - p1.X, Y: p2.Y - p1.Y}
		v2 := gocv.Point2f{X: p3.X - p2.X, Y: p3.Y - p2.Y}

		// Ignore zero-length vectors
		if (v1.X == 0 && v1.Y == 0) || (v2.X == 0 && v2.Y == 0) {
			continue
		}

		angle := angleBetween(v1, v2)
		totalAngleChange += math.Abs(angle)
		numSegments++
	}

	if numSegments > 0 {
		return totalAngleChange / float64(numSegments)
	}

	return math.MaxFloat64 // Cannot be calculated, treat as infinitely noisy
}

// FilterTracksBySmoothness filters tracks based on a simple smoothness threshold.
func FilterTracksBySmoothness(tracks []*Track, maxAverageAngleChange float64) []*Track {
	var smoothTracks []*Track
	for _, track := range tracks {
		if calculateSmoothnessMetric(track) <= maxAverageAngleChange {
			smoothTracks = append(smoothTracks, track)
		}
	}
	return smoothTracks
}

// FilterTracksByDensityAndSmoothness filters tracks based on spatial density,
// keeping only the smoothest tracks in dense areas.
func FilterTracksByDensityAndSmoothness(tracks []*Track, gridCellSize, minTracksPerCell, maxTracksPerCell int) []*Track {
	if gridCellSize <= 0 {
		gridCellSize = 32 // Default value
	}

	// 1. Bin tracks into a grid
	grid := make(map[image.Point][]*Track)
	for _, track := range tracks {
		if len(track.Points) == 0 {
			continue
		}
		startPoint := track.Points[0].Vec
		cellX := int(startPoint.X) / gridCellSize
		cellY := int(startPoint.Y) / gridCellSize
		cell := image.Point{X: cellX, Y: cellY}
		grid[cell] = append(grid[cell], track)
	}

	var finalTracks []*Track

	// 2. Filter within each grid cell
	for _, tracksInCell := range grid {
		if len(tracksInCell) < minTracksPerCell {
			continue // Skip sparse cells
		}

		// Sort tracks in the cell by length (descending), then by smoothness (ascending)
		sort.Slice(tracksInCell, func(i, j int) bool {
			lenI := len(tracksInCell[i].Points)
			lenJ := len(tracksInCell[j].Points)
			if lenI != lenJ {
				return lenI > lenJ
			}
			return calculateSmoothnessMetric(tracksInCell[i]) < calculateSmoothnessMetric(tracksInCell[j])
		})

		// Keep the top N smoothest tracks
		numToKeep := maxTracksPerCell
		if len(tracksInCell) < numToKeep {
			numToKeep = len(tracksInCell)
		}
		finalTracks = append(finalTracks, tracksInCell[:numToKeep]...)
	}

	return finalTracks
}

// FilterTracksByMaxAngleChange filters tracks by ensuring no single turn is too sharp.
func FilterTracksByMaxAngleChange(tracks []*Track, maxAngleChange float64) []*Track {
	var smoothTracks []*Track

	for _, track := range tracks {
		if len(track.Points) < 3 {
			smoothTracks = append(smoothTracks, track)
			continue
		}

		isSmooth := true
		for i := 0; i < len(track.Points)-2; i++ {
			p1 := track.Points[i].Vec
			p2 := track.Points[i+1].Vec
			p3 := track.Points[i+2].Vec

			v1 := gocv.Point2f{X: p2.X - p1.X, Y: p2.Y - p1.Y}
			v2 := gocv.Point2f{X: p3.X - p2.X, Y: p3.Y - p2.Y}

			if (v1.X == 0 && v1.Y == 0) || (v2.X == 0 && v2.Y == 0) {
				continue
			}

			angle := angleBetween(v1, v2)
			if math.Abs(angle) > maxAngleChange {
				isSmooth = false
				break
			}
		}

		if isSmooth {
			smoothTracks = append(smoothTracks, track)
		}
	}

	return smoothTracks
}

// angleBetween calculates the angle between two 2D vectors.
func angleBetween(v1, v2 gocv.Point2f) float64 {
	dot := float64(v1.X*v2.X + v1.Y*v2.Y)
	det := float64(v1.X*v2.Y - v1.Y*v2.X) // Cross product in 2D
	angle := math.Atan2(det, dot)
	return angle
}
