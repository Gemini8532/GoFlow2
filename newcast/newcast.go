// Package newcast provides tools for tracking features in image sequences and
// extrapolating their future positions. It is designed for tasks like weather
// radar nowcasting, where the short-term movement of objects (like rain cells)
// needs to be predicted.
//
// The core of the package is the Tracker, which uses the Lucas-Kanade optical
// flow algorithm to follow features from one frame to the next. It can handle
// tracks being lost and continuously updates the velocity and acceleration of
// each tracked feature.
//
// For motion estimation, the package first attempts to fit a quadratic polynomial
// to the recent history of a track's positions. This provides a smooth and
// robust estimate of velocity and acceleration. If the polynomial fit fails
// (e.g., due to insufficient data points), it falls back to using finite
// differences for a simpler, but still effective, motion estimation.
//
// The package is designed to be used in real-time applications, where images
// arrive sequentially and predictions need to be updated with each new frame.
package newcast

import (
	"fmt"
	"time"

	"gocv.io/x/gocv"
)

// Point represents a single detection of a feature at a specific time and
// location in image coordinates.
type Point struct {
	Time time.Time    // The timestamp of the image frame.
	Vec  gocv.Point2f // The (x, y) coordinate of the feature.
}

// Track represents the history of a single feature's movement across multiple
// image frames. It stores the feature's path, current motion estimates, and
// tracking status.
type Track struct {
	ID     int     // A unique identifier for the track.
	Points []Point // The sequence of observed points for this feature.

	// LatestVelocity is the estimated velocity (pixels per second) at the most
	// recent point in time.
	LatestVelocity gocv.Point2f

	// LatestAcceleration is the estimated acceleration (pixels per second^2)
	// at the most recent point in time.
	LatestAcceleration gocv.Point2f

	Lost bool // Lost is true if the feature could not be found in the latest frame.

	// PolyX and PolyY are the polynomial curves fitted to the recent history
	// of the track's x and y coordinates, respectively. These are used for
	// smooth motion estimation.
	PolyX Polynomial
	PolyY Polynomial
}

// Tracker is the main object for managing the feature tracking process.
// It detects features in the first frame and then uses optical flow to track
// them through subsequent frames.
type Tracker struct {
	maxFeatures int      // The maximum number of features to track.
	nextTrackID int      // The ID to assign to the next new track.
	tracks      []*Track // The list of all current tracks (including lost ones).
	prevImg     gocv.Mat // The previous image frame.
	prevPoints  gocv.Mat // The feature points from the previous frame.
}

// NewTracker initializes a new feature tracker.
//
// maxFeatures specifies the maximum number of features to detect in the first
// image. The tracker will attempt to find the "best" features up to this number.
func NewTracker(maxFeatures int) (*Tracker, error) {
	if maxFeatures <= 0 {
		return nil, fmt.Errorf("maxFeatures must be positive")
	}
	return &Tracker{
		maxFeatures: maxFeatures,
		nextTrackID: 0,
		tracks:      []*Track{},
		prevImg:     gocv.NewMat(),
		prevPoints:  gocv.NewMat(),
	}, nil
}

// Close releases the gocv.Mat resources held by the tracker. It should be
// called when the tracker is no longer needed to prevent memory leaks.
func (t *Tracker) Close() {
	t.prevImg.Close()
	t.prevPoints.Close()
}

// AddImage is the core function of the tracker. It processes a new image in the
// sequence, updating the tracks of existing features.
//
// If it's the first image, it detects good features to track. For subsequent
// images, it uses CalcOpticalFlowPyrLK to find the new positions of the
// features from the previous frame. It then updates the internal state of each
// track with the new point and re-estimates its motion.
//
// img is the new grayscale image frame.
// timestamp is the time at which the image was captured.
func (t *Tracker) AddImage(img gocv.Mat, timestamp time.Time) error {
	if img.Empty() {
		return fmt.Errorf("input image is empty")
	}

	// If this is the first image, find features to track.
	if t.prevImg.Empty() {
		return t.initializeTracks(img, timestamp)
	}

	// Track features from the previous image to the current one.
	nextPoints := gocv.NewMat()
	defer nextPoints.Close()
	status := gocv.NewMat()
	defer status.Close()
	errMat := gocv.NewMat()
	defer errMat.Close()

	gocv.CalcOpticalFlowPyrLK(t.prevImg, img, t.prevPoints, nextPoints, &status, &errMat)

	// Update tracks with the new points.
	t.updateTracks(nextPoints, status, timestamp)

	// Update the previous image and points for the next iteration.
	t.prevImg.Close()
	t.prevImg = img.Clone()
	t.updatePrevPoints()

	return nil
}

// initializeTracks finds good features in the first image and creates initial tracks.
func (t *Tracker) initializeTracks(img gocv.Mat, timestamp time.Time) error {
	points := gocv.NewMat()
	defer points.Close()

	gocv.GoodFeaturesToTrack(img, &points, t.maxFeatures, 0.01, 10)
	if points.Rows() == 0 {
		return fmt.Errorf("no features found in the first image")
	}

	for i := 0; i < points.Rows(); i++ {
		ptVec := points.GetVecfAt(i, 0)
		track := &Track{
			ID:     t.nextTrackID,
			Points: []Point{{Time: timestamp, Vec: gocv.Point2f{X: ptVec[0], Y: ptVec[1]}}},
			Lost:   false,
		}
		t.tracks = append(t.tracks, track)
		t.nextTrackID++
	}

	t.prevImg = img.Clone()
	t.prevPoints.Close()
	t.prevPoints = points.Clone()

	return nil
}

// updateTracks updates the feature tracks with new points and manages lost tracks.
func (t *Tracker) updateTracks(nextPoints, status gocv.Mat, timestamp time.Time) {
	survivingTracks := []*Track{}
	for i, track := range t.tracks {
		if track.Lost {
			continue
		}
		if status.GetUCharAt(i, 0) == 1 {
			var newPoint Point
			if nextPoints.Channels() == 2 {
				ptVec := nextPoints.GetVecfAt(i, 0)
				newPoint = Point{
					Time: timestamp,
					Vec:  gocv.Point2f{X: ptVec[0], Y: ptVec[1]},
				}
			} else {
				x := nextPoints.GetFloatAt(i, 0)
				y := nextPoints.GetFloatAt(i, 1)
				newPoint = Point{
					Time: timestamp,
					Vec:  gocv.Point2f{X: x, Y: y},
				}
			}
			track.Points = append(track.Points, newPoint)
			t.estimateMotion(track)
			survivingTracks = append(survivingTracks, track)
		} else {
			track.Lost = true
		}
	}
	t.tracks = survivingTracks
}

// updatePrevPoints creates a new set of points to track for the next frame.
func (t *Tracker) updatePrevPoints() {
	t.prevPoints.Close()
	if len(t.tracks) == 0 {
		t.prevPoints = gocv.NewMat()
		return
	}

	newPoints := gocv.NewMatWithSize(len(t.tracks), 2, gocv.MatTypeCV32F)
	for i, track := range t.tracks {
		lastPoint := track.Points[len(track.Points)-1].Vec
		newPoints.SetFloatAt(i, 0, lastPoint.X)
		newPoints.SetFloatAt(i, 1, lastPoint.Y)
	}
	t.prevPoints = newPoints
}

// estimateMotion estimates the velocity and acceleration of a track.
// It first attempts to fit a quadratic curve, falling back to finite differences.
func (t *Tracker) estimateMotion(track *Track) {
	numPoints := len(track.Points)
	if numPoints < 2 {
		return // Not enough data
	}

	// Attempt to fit a quadratic polynomial for better estimation
	if numPoints >= 4 {
		polyX, polyY, err := FitQuadratic(track.Points)
		if err == nil {
			track.PolyX = polyX
			track.PolyY = polyY
			t0 := track.Points[0].Time
			lastT := track.Points[numPoints-1].Time.Sub(t0).Seconds()

			vx := float32(polyX.Velocity(lastT))
			vy := float32(polyY.Velocity(lastT))
			track.LatestVelocity = gocv.Point2f{X: vx, Y: vy}

			ax := float32(polyX.Acceleration())
			ay := float32(polyY.Acceleration())
			track.LatestAcceleration = gocv.Point2f{X: ax, Y: ay}
			return
		}
	}

	// Fallback to simple finite differences if curve fitting fails or not enough points
	p1 := track.Points[numPoints-1]
	p0 := track.Points[numPoints-2]
	dt := p1.Time.Sub(p0.Time).Seconds()
	if dt > 0 {
		vx := (p1.Vec.X - p0.Vec.X) / float32(dt)
		vy := (p1.Vec.Y - p0.Vec.Y) / float32(dt)
		track.LatestVelocity = gocv.Point2f{X: vx, Y: vy}
	}

	if numPoints < 3 {
		return
	}
	p_minus_1 := track.Points[numPoints-3]
	dt_prev := p0.Time.Sub(p_minus_1.Time).Seconds()
	if dt_prev > 0 {
		vx_prev := (p0.Vec.X - p_minus_1.Vec.X) / float32(dt_prev)
		vy_prev := (p0.Vec.Y - p_minus_1.Vec.Y) / float32(dt_prev)

		avg_dt := (dt + dt_prev) / 2.0
		if avg_dt > 0 {
			ax := (track.LatestVelocity.X - vx_prev) / float32(avg_dt)
			ay := (track.LatestVelocity.Y - vy_prev) / float32(avg_dt)
			track.LatestAcceleration = gocv.Point2f{X: ax, Y: ay}
		}
	}
}

// GetTracks returns a slice of the currently active tracks.
// It filters out any tracks that have been marked as "lost".
func (t *Tracker) GetTracks() []*Track {
	activeTracks := []*Track{}
	for _, track := range t.tracks {
		if !track.Lost {
			activeTracks = append(activeTracks, track)
		}
	}
	return activeTracks
}
