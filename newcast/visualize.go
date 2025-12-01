package newcast

import (
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

// MatToPNG converts a gocv.Mat to a PNG byte buffer.
func MatToPNG(img gocv.Mat) ([]byte, error) {
	buf, err := gocv.IMEncode(".png", img)
	if err != nil {
		return nil, err
	}
	defer buf.Close()
	return buf.GetBytes(), nil
}

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