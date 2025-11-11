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
			p1 := image.Point{int(track.Points[j].Vec.X), int(track.Points[j].Vec.Y)}
			p2 := image.Point{int(track.Points[j+1].Vec.X), int(track.Points[j+1].Vec.Y)}
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

		// --- Draw existing track ---
		c := color.RGBA{
			R: uint8((i * 40) % 255),
			G: uint8((i * 60) % 255),
			B: uint8((i * 80) % 255),
			A: 255,
		}
		for j := 0; j < len(track.Points)-1; j++ {
			p1 := image.Point{int(track.Points[j].Vec.X), int(track.Points[j].Vec.Y)}
			p2 := image.Point{int(track.Points[j+1].Vec.X), int(track.Points[j+1].Vec.Y)}
			gocv.Line(&img, p1, p2, c, 2)
		}

		// --- Draw extrapolated future path ---
		if numFuturePoints > 0 && track.PolyX.A != 0 { // Check if polynomial has been fitted
			t0 := track.Points[0].Time
			lastPoint := track.Points[len(track.Points)-1]
			lastT := lastPoint.Time.Sub(t0).Seconds()
			
			// Calculate the average time interval between tracked points
			var avgDt float64
			if len(track.Points) > 1 {
				totalDuration := track.Points[len(track.Points)-1].Time.Sub(track.Points[0].Time).Seconds()
				avgDt = totalDuration / float64(len(track.Points)-1)
			} else {
				avgDt = 1.0 // Default if only one point
			}

			p1 := image.Point{int(lastPoint.Vec.X), int(lastPoint.Vec.Y)}

			for j := 1; j <= numFuturePoints; j++ {
				futureT := lastT + float64(j)*avgDt
				futureX := track.PolyX.Eval(futureT)
				futureY := track.PolyY.Eval(futureT)
				p2 := image.Point{int(futureX), int(futureY)}

				gocv.Line(&img, p1, p2, color.RGBA{R: 255, G: 0, B: 0, A: 255}, 1)
				p1 = p2
			}
		}
	}

	return img
}

// VisualizeVectors draws the final velocity vectors of the tracks.
func VisualizeVectors(tracks []*Track, width, height int, scale float32) gocv.Mat {
	img := gocv.NewMatWithSize(height, width, gocv.MatTypeCV8UC3)
	img.SetTo(gocv.NewScalar(0, 0, 0, 0)) // Black background

	for _, track := range tracks {
		if len(track.Points) < 1 {
			continue
		}

		lastPoint := track.Points[len(track.Points)-1]
		p1 := image.Point{int(lastPoint.Vec.X), int(lastPoint.Vec.Y)}

		// Calculate the endpoint of the velocity vector
		p2 := image.Point{
			int(lastPoint.Vec.X + track.LatestVelocity.X*scale),
			int(lastPoint.Vec.Y + track.LatestVelocity.Y*scale),
		}

		gocv.ArrowedLine(&img, p1, p2, color.RGBA{R: 0, G: 255, B: 0, A: 255}, 2)
	}

	return img
}
