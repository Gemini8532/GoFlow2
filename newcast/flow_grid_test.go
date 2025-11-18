package newcast

import (

	"fmt"

	"log"

	"testing"



	"github.com/stretchr/testify/assert"

)



var files = []string{

	"/home/jdp/maps/weather/uk/2025-11-18/2025-11-18T13:45:00Z.png",

	"/home/jdp/maps/weather/uk/2025-11-18/2025-11-18T13:50:00Z.png",

	"/home/jdp/maps/weather/uk/2025-11-18/2025-11-18T13:55:00Z.png",

	"/home/jdp/maps/weather/uk/2025-11-18/2025-11-18T14:00:00Z.png",

	"/home/jdp/maps/weather/uk/2025-11-18/2025-11-18T14:05:00Z.png",

	"/home/jdp/maps/weather/uk/2025-11-18/2025-11-18T14:10:00Z.png",

	"/home/jdp/maps/weather/uk/2025-11-18/2025-11-18T14:15:00Z.png",

	"/home/jdp/maps/weather/uk/2025-11-18/2025-11-18T14:20:00Z.png",

	"/home/jdp/maps/weather/uk/2025-11-18/2025-11-18T14:25:00Z.png",

	"/home/jdp/maps/weather/uk/2025-11-18/2025-11-18T14:30:00Z.png",

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

func TestMain(t *testing.T) {
	width, height, timeFrames := 100, 100, 5

	// 1. Create Processor (allocates internal build buffers)
	processor := NewFlowProcessor(width, height, timeFrames)

	track1 := LTrack{
		{2.1, 2.1},
		{3.1, 2.1},
		{4.1, 2.1},
	}
	tracks := []LTrack{track1}

	// 2. Run Operations
	processor.ProcessTracks(tracks)
	processor.CalculateAverages()
	processor.FillGaps(2)

	// 3. Extract Result
	// The Grid field is now a clean struct without temp data
	resultGrid := processor.Grid

	fmt.Printf("Result Grid Size: %d vectors\n", len(resultGrid.Data))

	// Verify output
	idx := (0*height+2)*width + 2 // T=0, Y=2, X=2
	vec := resultGrid.Data[idx]
	if vec.Vx != 0 || vec.Vy != 0 {
		fmt.Printf("Value at (2,2): %v\n", vec)
	}
}
