package main

import (
	"bytes"
	"encoding/json"
	"example/goflow/newcast"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gocv.io/x/gocv"
)

func main() {
	// Command-Line Flags
	maxFeatures := flag.Int("maxFeatures", 200, "Maximum number of features to track.")
	vectorScale := flag.Float64("vectorScale", 50.0, "Scaling factor for drawing velocity vectors.")
	minTrackLength := flag.Int("minTrackLength", 6, "Minimum number of points a track must have to be considered.")
	extrapolate := flag.Int("extrapolate", 0, "Number of future points to extrapolate and draw.")
	serverURL := flag.String("serverURL", "", "URL of the processing server (e.g., http://localhost:9093). If provided, processing is done remotely.")
	requestID := flag.String("id", "", "Optional: A custom ID for the remote processing request. Only used with -serverURL.")
	blurSigma := flag.Float64("blurSigma", 0.0, "Gaussian blur sigma for flow grid smoothing (0 = no blur, 1.0 = light, 2.0 = medium).")
	useFittedPoints := flag.Bool("useFittedPoints", false, "Use polynomial-fitted points instead of raw tracked points (reduces noise).")

	// Curve-fit filtering parameters (updated to recommended defaults)
	minRSquared := flag.Float64("minRSquared", 0.90, "Minimum R² for curve-fit filter (0-1, higher = stricter).")
	maxRMSE := flag.Float64("maxRMSE", 3.0, "Maximum RMSE in pixels for curve-fit filter.")
	maxDeviation := flag.Float64("maxDeviation", 7.0, "Maximum deviation from fitted curve in pixels.")
	maxAcceleration := flag.Float64("maxAcceleration", 1.5, "Maximum acceleration in pixels/frame² for curve-fit filter.")

	flag.Parse()

	fileArgs := flag.Args()
	if len(fileArgs) == 0 {
		fmt.Println("Error: No input files provided.")
		fmt.Println("Usage: go run main.go [flags] <file1.png> <file2.png> ...")
		os.Exit(1)
	}

	config := newcast.ProcessConfig{
		MaxFeatures:     *maxFeatures,
		MinTrackLength:  *minTrackLength,
		BlurSigma:       *blurSigma,
		UseFittedPoints: *useFittedPoints,
		MinRSquared:     *minRSquared,
		MaxRMSE:         *maxRMSE,
		MaxDeviation:    *maxDeviation,
		MaxAcceleration: *maxAcceleration,
	}

	if *serverURL != "" {
		processRemotely(*serverURL, *requestID, fileArgs, config)
	} else {
		processLocally(fileArgs, config, *vectorScale, *extrapolate)
	}
}

func processLocally(fileArgs []string, config newcast.ProcessConfig, vectorScale float64, extrapolate int) {
	fmt.Println("--- Processing locally ---")
	fmt.Printf("Processing %d input files\n", len(fileArgs))
	fmt.Printf("Running with parameters: maxFeatures=%d, vectorScale=%.2f, minTrackLength=%d, extrapolate=%d\n",
		config.MaxFeatures, vectorScale, config.MinTrackLength, extrapolate)
	fmt.Println("Using curvefit filtering:")
	fmt.Printf("  MinRSquared: %.2f\n", config.MinRSquared)
	fmt.Printf("  MaxRMSE: %.2f\n", config.MaxRMSE)
	fmt.Printf("  MaxDeviation: %.2f\n", config.MaxDeviation)
	fmt.Printf("  MaxAcceleration: %.2f\n", config.MaxAcceleration)

	filteredTracks, width, height, err := newcast.ProcessFilesToTracks(fileArgs, config)
	if err != nil {
		fmt.Printf("Error processing files: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d filtered tracks.\n", len(filteredTracks))

	// --- Generate Visualizations ---
	trackImg := newcast.VisualizeTracks(filteredTracks, width, height)
	defer trackImg.Close()
	trackImgPath := "rainfall_tracks.png"
	if ok := gocv.IMWrite(trackImgPath, trackImg); !ok {
		fmt.Printf("Error writing track visualization to %s\n", trackImgPath)
		os.Exit(1)
	}
	fmt.Printf("Track visualization saved to %s\n", trackImgPath)

	vectorImg := newcast.VisualizeVectors(filteredTracks, width, height, vectorScale)
	defer vectorImg.Close()
	vectorImgPath := "rainfall_vectors.png"
	if ok := gocv.IMWrite(vectorImgPath, vectorImg); !ok {
		fmt.Printf("Error writing vector visualization to %s\n", vectorImgPath)
		os.Exit(1)
	}
	fmt.Printf("Vector visualization saved to %s\n", vectorImgPath)

	if extrapolate > 0 {
		extrapolatedImg := newcast.VisualizeExtrapolatedTracks(filteredTracks, width, height, extrapolate)
		defer extrapolatedImg.Close()
		extrapolatedImgPath := "rainfall_tracks_extrapolated.png"
		if ok := gocv.IMWrite(extrapolatedImgPath, extrapolatedImg); !ok {
			fmt.Printf("Error writing extrapolated track visualization to %s\n", extrapolatedImgPath)
			os.Exit(1)
		}
		fmt.Printf("Extrapolated track visualization saved to %s\n", extrapolatedImgPath)
	}
}

func processRemotely(serverURL string, requestID string, fileArgs []string, config newcast.ProcessConfig) {
	fmt.Printf("--- Processing remotely on server: %s ---\n", serverURL)

	if requestID == "" {
		requestID = fmt.Sprintf("cast-%d", time.Now().Unix())
	}
	fmt.Printf("Using request ID: %s\n", requestID)

	absFileArgs := make([]string, len(fileArgs))
	for i, f := range fileArgs {
		absPath, err := filepath.Abs(f)
		if err != nil {
			fmt.Printf("Error converting to absolute path for %s: %v\n", f, err)
			os.Exit(1)
		}
		absFileArgs[i] = absPath
	}

	reqBody := map[string]interface{}{
		"filenames": absFileArgs,
		"id":        requestID,
		"config":    config,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("Error marshalling request body: %v\n", err)
		os.Exit(1)
	}

	processURL := serverURL + "/process"
	resp, err := http.Post(processURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("Error sending request to server: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("Server returned an error: %s\n", resp.Status)
		fmt.Printf("Response body: %s\n", string(bodyBytes))
		os.Exit(1)
	}

	fmt.Printf("Successfully submitted processing request with ID: %s\n", requestID)
	fmt.Println("\nTo fetch the flow grid, use the following command:")
	fmt.Printf("curl -o flow_grid.bin \"%s/average-flow-grid?id=%s\"\n", serverURL, requestID)
}
