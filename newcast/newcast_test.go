package newcast

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTracker(t *testing.T) {
	// Paths to test images, relative to project root
	imgPath1 := "../test_data/centered.png"
	imgPath2 := "../test_data/shifted.png"

	// Load images
	img1, err := loadImageAsGrayscale(imgPath1)
	require.NoError(t, err, "Failed to load image 1")
	defer img1.Close()

	img2, err := loadImageAsGrayscale(imgPath2)
	require.NoError(t, err, "Failed to load image 2")
	defer img2.Close()

	// Create a new tracker
	maxFeatures := 50
	tracker, err := NewTracker(maxFeatures)
	require.NoError(t, err, "Failed to create tracker")
	defer tracker.Close()

	// Add the first image
	err = tracker.AddImage(img1)
	require.NoError(t, err, "Failed to add first image")

	// Check initial tracks
	initialTracks := tracker.GetTracks()
	require.NotEmpty(t, initialTracks, "No initial tracks were created.")
	require.LessOrEqual(t, len(initialTracks), maxFeatures, "Expected at most %d features, but got %d", maxFeatures, len(initialTracks))
	t.Logf("Found %d initial features to track.", len(initialTracks))

	// Add the second image
	err = tracker.AddImage(img2)
	require.NoError(t, err, "Failed to add second image")

	// Check updated tracks
	updatedTracks := tracker.GetTracks()
	require.NotEmpty(t, updatedTracks, "All tracks were lost after the second image.")
	t.Logf("%d tracks survived.", len(updatedTracks))

	// Check the motion of a surviving track
	track := updatedTracks[0]
	require.Len(t, track.Points, 2, "Expected track to have 2 points")

	p1 := track.Points[0]
	p2 := track.Points[1]

	dx := p2.X - p1.X
	dy := p2.Y - p1.Y

	// The shift in the test data is (20, 10)
	expectedDx := 20.0
	expectedDy := 10.0

	// Allow some tolerance for the feature detection and tracking
	require.InDelta(t, expectedDx, dx, 1.0, "Expected displacement close to (%f, %f), but got (%f, %f)", expectedDx, expectedDy, dx, dy)
	require.InDelta(t, expectedDy, dy, 1.0, "Expected displacement close to (%f, %f), but got (%f, %f)", expectedDx, expectedDy, dx, dy)

	// Check velocity
	// dt is implicitly 1.0, so velocity should be equal to displacement
	vx := track.LatestVelocity.X
	vy := track.LatestVelocity.Y
	require.InDelta(t, expectedDx, float64(vx), 1.0, "Expected velocity close to (%f, %f), but got (%f, %f)", expectedDx, expectedDy, vx, vy)
	require.InDelta(t, expectedDy, float64(vy), 1.0, "Expected velocity close to (%f, %f), but got (%f, %f)", expectedDx, expectedDy, vx, vy)
}

