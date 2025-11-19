package newcast

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

var files = []string{
	"../test_data/centered.png",
	"../test_data/shifted.png",
}

func TestLow(t *testing.T) {

	maxFeatures := 1000

	tracker, err := NewTracker(maxFeatures)

	if err != nil {

		t.Fatalf("Failed to create tracker: %v", err)

	}

	defer tracker.Close()

	for _, f := range files {

		img, err := loadImageAsGrayscale(f)

		assert.NoError(t, err)

		err = tracker.AddImage(img)

	}

	initialTracks := tracker.GetTracks()

	log.Println("initialTracks", len(initialTracks))

}

func TestConstantFlow(t *testing.T) {
	width, height, timeFrames := 10, 10, 2
	processor := NewFlowProcessor(width, height, timeFrames)

	// A single track moving diagonally with constant velocity
	tracks := []*Track{
		{Points: []Point{
			{X: 1.0, Y: 1.0}, // T=0
			{X: 2.0, Y: 2.0}, // T=1
		}},
	}
	expectedVx, expectedVy := 1.0, 1.0

	// Run all processing steps
	processor.ProcessTracks(tracks)
	processor.CalculateAverages()
	processor.FillGaps() // Large enough to fill the whole grid

	// Verification
	resultGrid := processor.Grid
	for time_idx := 0; time_idx < timeFrames-1; time_idx++ { // Only check frames with flow data
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				idx := processor.getIndex(x, y, time_idx)
				vec := resultGrid.Data[idx]
				assert.InDelta(t, expectedVx, vec.Vx, 1e-9, "Vx at (%d,%d,%d) should be constant", x, y, time_idx)
				assert.InDelta(t, expectedVy, vec.Vy, 1e-9, "Vy at (%d,%d,%d) should be constant", x, y, time_idx)
			}
		}
	}
}

func TestMultipleFlows(t *testing.T) {
	width, height, timeFrames := 20, 20, 2
	processor := NewFlowProcessor(width, height, timeFrames)

	// Two tracks in different regions with different constant flows
	tracks := []*Track{
		// Top-left region, moving right
		{Points: []Point{{X: 2.0, Y: 2.0}, {X: 3.0, Y: 2.0}}},
		// Bottom-right region, moving up
		{Points: []Point{{X: 18.0, Y: 18.0}, {X: 18.0, Y: 17.0}}},
	}
	vx1, vy1 := 1.0, 0.0
	vx2, vy2 := 0.0, -1.0

	processor.ProcessTracks(tracks)
	processor.CalculateAverages()
	processor.FillGaps() // Fill the whole grid

	resultGrid := processor.Grid
	tFrame := 0 // Check the first time frame

	// Check the original flow regions
	idx1 := processor.getIndex(2, 2, tFrame)
	vec1 := resultGrid.Data[idx1]
	assert.InDelta(t, vx1, vec1.Vx, 0.5, "Vx for flow 1 should be correct")
	assert.InDelta(t, vy1, vec1.Vy, 0.5, "Vy for flow 1 should be correct")

	idx2 := processor.getIndex(18, 18, tFrame)
	vec2 := resultGrid.Data[idx2]
	assert.InDelta(t, vx2, vec2.Vx, 0.5, "Vx for flow 2 should be correct")
	assert.InDelta(t, vy2, vec2.Vy, 0.5, "Vy for flow 2 should be correct")

	// Check a point in the middle, expecting some interpolation
	midX, midY := width/2, height/2
	midIdx := processor.getIndex(midX, midY, tFrame)
	midVec := resultGrid.Data[midIdx]

	// The interpolated vector should be a blend of the two flows.
	// A simple check is that its components are between the min and max of the source flows.
	assert.True(t, midVec.Vx >= vx2 && midVec.Vx <= vx1, "Interpolated Vx should be between the source Vx values")
	assert.True(t, midVec.Vy >= vy2 && midVec.Vy <= vy1, "Interpolated Vy should be between the source Vy values")

	// Check a corner far from any data, it should have a reasonable interpolated value
	cornerIdx := processor.getIndex(0, height-1, tFrame)
	cornerVec := resultGrid.Data[cornerIdx]
	assert.True(t, cornerVec.Vx >= vx2 && cornerVec.Vx <= vx1, "Corner Vx should be interpolated")
	assert.True(t, cornerVec.Vy >= vy2 && cornerVec.Vy <= vy1, "Corner Vy should be interpolated")
}
