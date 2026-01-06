package newcast

import (
	"image"
	"testing"

	"github.com/stretchr/testify/require"
	"gocv.io/x/gocv"
)

func TestTrackerRecoverFromLostTracks(t *testing.T) {
	// Paths to test images
	imgPath1 := "../test_data/centered.png"
	// Blank image will cause tracking to fail for all points
	imgPathBlank := "../test_data/blank.png"

	img1, err := loadImageAsGrayscale(imgPath1)
	require.NoError(t, err)
	defer img1.Close()

	imgBlank, err := loadImageAsGrayscale(imgPathBlank)
	require.NoError(t, err)
	defer imgBlank.Close()

	// Ensure images are compatible sizes
	if imgBlank.Rows() != img1.Rows() || imgBlank.Cols() != img1.Cols() {
		gocv.Resize(imgBlank, &imgBlank, image.Point{X: img1.Cols(), Y: img1.Rows()}, 0, 0, gocv.InterpolationLinear)
	}

	tracker, err := NewTracker(50)
	require.NoError(t, err)
	defer tracker.Close()

	// 1. Add valid image -> tracks created
	err = tracker.AddImage(img1)
	require.NoError(t, err)
	tracks := tracker.GetTracks()
	require.NotEmpty(t, tracks, "Should have initial tracks")

	// 2. Add blank image -> all tracks should be lost
	// Optical flow on blank image will likely yield no matches
	err = tracker.AddImage(imgBlank)
	require.NoError(t, err)
	tracks = tracker.GetTracks()
	require.Empty(t, tracks, "Should have lost all tracks on blank image")

	// 3. Add valid image again -> should re-initialize and NOT crash
	err = tracker.AddImage(img1)
	require.NoError(t, err, "Should not crash when re-initializing after lost tracks")
	
	tracks = tracker.GetTracks()
	require.NotEmpty(t, tracks, "Should have found new tracks after re-initialization")
}