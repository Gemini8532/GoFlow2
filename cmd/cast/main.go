package main

import (
	"example/goflow/newcast"
	"flag"
	"fmt"
	"os"
	"sort"

	"gocv.io/x/gocv"
)

func main() {
	// --- Command-Line Flags ---
	maxFeatures := flag.Int("maxFeatures", 200, "Maximum number of features to track.")
	smoothness := flag.Float64("smoothness", 0.5, "Smoothness threshold (max average angle change in radians).")
	vectorScale := flag.Float64("vectorScale", 50.0, "Scaling factor for drawing velocity vectors.")
	filterType := flag.String("filterType", "smoothness", "Type of filter to use: 'smoothness', 'density', or 'max_angle'.")
	maxAngle := flag.Float64("maxAngle", 0.8, "Maximum allowed angle change (in radians) for the max_angle filter.")
	gridCellSize := flag.Int("gridCellSize", 64, "Grid cell size for density filter.")
	minTracksPerCell := flag.Int("minTracksPerCell", 2, "Minimum number of tracks in a cell to be considered dense.")
	maxTracksPerCell := flag.Int("maxTracksPerCell", 5, "Maximum number of smoothest tracks to keep from a dense cell.")
	minTrackLength := flag.Int("minTrackLength", 6, "Minimum number of points a track must have to be considered.")
	extrapolate := flag.Int("extrapolate", 0, "Number of future points to extrapolate and draw.")
	flag.Parse()

	// Get file paths from command line arguments
	fileArgs := flag.Args()
	if len(fileArgs) == 0 {
		fmt.Println("Error: No input files provided.")
		fmt.Println("Usage: go run main.go [flags] <file1.png> <file2.png> ...")
		os.Exit(1)
	}

	fmt.Printf("Running with parameters: maxFeatures=%d, vectorScale=%.2f, minTrackLength=%d, extrapolate=%d\n",
		*maxFeatures, *vectorScale, *minTrackLength, *extrapolate)
	fmt.Printf("Filter type: %s\n", *filterType)
	switch *filterType {
	case "smoothness":
		fmt.Printf("Smoothness filter params: smoothness=%.2f\n", *smoothness)
	case "density":
		fmt.Printf("Density filter params: gridCellSize=%d, minTracksPerCell=%d, maxTracksPerCell=%d\n",
			*gridCellSize, *minTracksPerCell, *maxTracksPerCell)
	case "max_angle":
		fmt.Printf("Max Angle filter params: maxAngle=%.2f\n", *maxAngle)
	}

	fmt.Printf("Processing %d input files\n", len(fileArgs))

	// Sort the input files to ensure consistent processing order
	testImagePaths := make([]string, len(fileArgs))
	copy(testImagePaths, fileArgs)
	sort.Strings(testImagePaths)

	// --- Run Tracker ---
	fmt.Println("Running tracker on rainfall data...")
	tracker, err := newcast.NewTracker(*maxFeatures)
	if err != nil {
		fmt.Printf("Error creating tracker: %v\n", err)
		os.Exit(1)
	}
	defer tracker.Close()

	var width, height int
	for i, imgPath := range testImagePaths {
		img, err := loadImageAsGrayscale(imgPath)
		if err != nil {
			fmt.Printf("Error loading image %s: %v\n", imgPath, err)
			os.Exit(1)
		}
		defer img.Close()

		if i == 0 {
			width = img.Cols()
			height = img.Rows()
		}

		if err := tracker.AddImage(img); err != nil {
			fmt.Printf("Error adding image %s: %v\n", imgPath, err)
			os.Exit(1)
		}
	}
	fmt.Println("Tracking complete.")

	// --- Filter and Generate Visualizations ---
	allTracks := tracker.GetTracks()
	fmt.Printf("Found %d surviving tracks.\n", len(allTracks))

	// Pre-filter by track length
	var longTracks []*newcast.Track
	for _, track := range allTracks {
		if len(track.Points) >= *minTrackLength {
			longTracks = append(longTracks, track)
		}
	}
	fmt.Printf("Found %d tracks with at least %d points.\n", len(longTracks), *minTrackLength)

	var filteredTracks []*newcast.Track
	switch *filterType {
	case "density":
		// First, apply a baseline smoothness filter
		smoothTracks := newcast.FilterTracksBySmoothness(longTracks, *smoothness)
		fmt.Printf("Found %d tracks passing the smoothness threshold.\n", len(smoothTracks))
		// Then, apply the density filter to the smooth tracks
		filteredTracks = newcast.FilterTracksByDensityAndSmoothness(smoothTracks, *gridCellSize, *minTracksPerCell, *maxTracksPerCell)
		fmt.Printf("Filtered down to %d tracks using density filter.\n", len(filteredTracks))
	case "max_angle":
		filteredTracks = newcast.FilterTracksByMaxAngleChange(longTracks, *maxAngle)
		fmt.Printf("Filtered down to %d tracks using max_angle filter.\n", len(filteredTracks))
	default: // "smoothness"
		filteredTracks = newcast.FilterTracksBySmoothness(longTracks, *smoothness)
		fmt.Printf("Filtered down to %d tracks using smoothness filter.\n", len(filteredTracks))
	}

	// Visualize tracks as lines
	trackImg := newcast.VisualizeTracks(filteredTracks, width, height)
	defer trackImg.Close()
	trackImgPath := "rainfall_tracks.png"
	if ok := gocv.IMWrite(trackImgPath, trackImg); !ok {
		fmt.Printf("Error writing track visualization to %s\n", trackImgPath)
		os.Exit(1)
	}
	fmt.Printf("Track visualization saved to %s\n", trackImgPath)

	// Visualize final velocity vectors
	vectorImg := newcast.VisualizeVectors(filteredTracks, width, height, float32(*vectorScale))
	defer vectorImg.Close()
	vectorImgPath := "rainfall_vectors.png"
	if ok := gocv.IMWrite(vectorImgPath, vectorImg); !ok {
		fmt.Printf("Error writing vector visualization to %s\n", vectorImgPath)
		os.Exit(1)
	}
	fmt.Printf("Vector visualization saved to %s\n", vectorImgPath)

	// Visualize extrapolated tracks if requested
	if *extrapolate > 0 {
		extrapolatedImg := newcast.VisualizeExtrapolatedTracks(filteredTracks, width, height, *extrapolate)
		defer extrapolatedImg.Close()
		extrapolatedImgPath := "rainfall_tracks_extrapolated.png"
		if ok := gocv.IMWrite(extrapolatedImgPath, extrapolatedImg); !ok {
			fmt.Printf("Error writing extrapolated track visualization to %s\n", extrapolatedImgPath)
			os.Exit(1)
		}
		fmt.Printf("Extrapolated track visualization saved to %s\n", extrapolatedImgPath)
	}
}

// loadImageAsGrayscale loads an image from the given path and converts it to a grayscale gocv.Mat.
func loadImageAsGrayscale(path string) (gocv.Mat, error) {
	imgMat := gocv.IMRead(path, gocv.IMReadGrayScale)
	if imgMat.Empty() {
		return gocv.NewMat(), fmt.Errorf("failed to read image %s", path)
	}
	return imgMat, nil
}
