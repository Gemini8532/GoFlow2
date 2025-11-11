package newcast

import (
	"fmt"
	"time"

	"gocv.io/x/gocv"
)

// Point represents a point in time and space.
type Point struct {
	Time time.Time
	Vec  gocv.Point2f
}

// Track represents the path of a single feature over time.
type Track struct {
	ID                 int
	Points             []Point
	LatestVelocity     gocv.Point2f
	LatestAcceleration gocv.Point2f
	Lost               bool
	PolyX              Polynomial // Polynomial for X coordinate
	PolyY              Polynomial // Polynomial for Y coordinate
}

// Tracker manages the tracking of features across multiple images.
type Tracker struct {
	maxFeatures int
	nextTrackID int
	tracks      []*Track
	prevImg     gocv.Mat
	prevPoints  gocv.Mat
}

// NewTracker creates a new feature tracker.
// maxFeatures is the number of features to detect in the first image.
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

// Close releases the memory used by the tracker.
func (t *Tracker) Close() {
	t.prevImg.Close()
	t.prevPoints.Close()
}

// AddImage processes a new image in the sequence.
// img is the new image.
// timestamp is the time the image was captured.
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

// GetTracks returns the current set of active tracks.
func (t *Tracker) GetTracks() []*Track {
	activeTracks := []*Track{}
	for _, track := range t.tracks {
		if !track.Lost {
			activeTracks = append(activeTracks, track)
		}
	}
	return activeTracks
}
