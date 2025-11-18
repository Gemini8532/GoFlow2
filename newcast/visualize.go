package newcast

import (
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

// VisualizeTracks draws the paths of the tracks on a black background.
// Each track is drawn in a different color.
func VisualizeTracks(tracks []*Track, width, height int) gocv.Mat {
	img := gocv.NewMatWithSize(height, width, gocv.MatTypeCV8UC3)
	img.SetTo(gocv.NewScalar(0, 0, 0, 0)) // Black background

	for i, track := range tracks {
		if len(track.Points) < 2 {
			continue
		}

		// Assign a color based on the track ID
		c := color.RGBA{
			R: uint8((i * 40) % 255),
			G: uint8((i * 60) % 255),
			B: uint8((i * 80) % 255),
			A: 255,
		}

		// Draw lines between consecutive points in the track
		for j := 0; j < len(track.Points)-1; j++ {
			p1 := image.Point{int(track.Points[j].X), int(track.Points[j].Y)}
			p2 := image.Point{int(track.Points[j+1].X), int(track.Points[j+1].Y)}
			gocv.Line(&img, p1, p2, c, 2)
		}
	}

	return img
}

// VisualizeExtrapolatedTracks draws the actual and extrapolated future paths of tracks.
func VisualizeExtrapolatedTracks(tracks []*Track, width, height, numFuturePoints int) gocv.Mat {
	img := gocv.NewMatWithSize(height, width, gocv.MatTypeCV8UC3)
	img.SetTo(gocv.NewScalar(0, 0, 0, 0)) // Black background

	for i, track := range tracks {
		if len(track.Points) < 2 {
			continue
		}

		// Assign a color based on the track ID
		c := color.RGBA{
			R: uint8((i * 40) % 255),
			G: uint8((i * 60) % 255),
			B: uint8((i * 80) % 255),
			A: 255,
		}

		// --- Draw extrapolated complete path (from beginning of track) ---
		if numFuturePoints > 0 && track.PolyX.A != 0 { // Check if polynomial has been fitted
			// The average time delta is implicitly 1.0
			avgDt := 1.0

			// Generate points for the whole path (original + extrapolated)
			var allPoints []image.Point

			// Add points for original track (evaluated using polynomial)
			for j := 0; j < len(track.Points); j++ {
				currentT := float64(j)
				x := track.PolyX.Eval(currentT)
				y := track.PolyY.Eval(currentT)
				allPoints = append(allPoints, image.Point{int(x), int(y)})
			}

			// Add points for extrapolated track
			lastT := float64(len(track.Points) - 1)
			for j := 1; j <= numFuturePoints; j++ {
				futureT := lastT + float64(j)*avgDt
				futureX := track.PolyX.Eval(futureT)
				futureY := track.PolyY.Eval(futureT)
				allPoints = append(allPoints, image.Point{int(futureX), int(futureY)})
			}

			// Draw the full extrapolated path as solid line
			for j := 0; j < len(allPoints)-1; j++ {
				gocv.Line(&img, allPoints[j], allPoints[j+1], c, 2)
			}

			// Draw original track data points as circles to distinguish them from extrapolated path
			for j := 0; j < len(track.Points); j++ {
				center := image.Point{int(track.Points[j].X), int(track.Points[j].Y)} // Actual original point
				gocv.Circle(&img, center, 4, c, 2)                                    // Draw circle with radius 4
			}
		} else {
			// If no extrapolation is possible, just draw the original track as solid
			for j := 0; j < len(track.Points)-1; j++ {
				p1 := image.Point{int(track.Points[j].X), int(track.Points[j].Y)}
				p2 := image.Point{int(track.Points[j+1].X), int(track.Points[j+1].Y)}
				gocv.Line(&img, p1, p2, c, 2)
			}
		}
	}

	return img
}

// VisualizeVectors draws the final velocity vectors of the tracks.
func VisualizeVectors(tracks []*Track, width, height int, scale float64) gocv.Mat {
	img := gocv.NewMatWithSize(height, width, gocv.MatTypeCV8UC3)
	img.SetTo(gocv.NewScalar(0, 0, 0, 0)) // Black background

	for _, track := range tracks {
		if len(track.Points) < 1 {
			continue
		}

		lastPoint := track.Points[len(track.Points)-1]
		p1 := image.Point{int(lastPoint.X), int(lastPoint.Y)}

		// Calculate the endpoint of the velocity vector
		p2 := image.Point{
			int(lastPoint.X + track.LatestVelocity.X*scale),
			int(lastPoint.Y + track.LatestVelocity.Y*scale),
		}

		gocv.ArrowedLine(&img, p1, p2, color.RGBA{R: 0, G: 255, B: 0, A: 255}, 2)
	}

	return img
}
