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
)

func main() {
	// --- Command-Line Flags ---
	maxFeatures := flag.Int("maxFeatures", 200, "Maximum number of features to track.")
	minTrackLength := flag.Int("minTrackLength", 6, "Minimum number of points a track must have to be considered.")
	serverURL := flag.String("serverURL", "", "URL of the processing server (e.g., http://localhost:8080). If provided, processing is done remotely.")
	requestID := flag.String("id", "", "Optional: A custom ID for the remote processing request. Only used with -serverURL.")
	outputGrid := flag.String("outputGrid", "", "Output file for saving average flow grid (required when serverURL is not specified).")
	blurSigma := flag.Float64("blurSigma", 0.0, "Gaussian blur sigma for flow grid smoothing (0 = no blur, 1.0 = light, 2.0 = medium).")

	// Curve-fit filtering parameters
	minRSquared := flag.Float64("minRSquared", 0.85, "Minimum R² for curve-fit filter (0-1, higher = stricter).")
	maxRMSE := flag.Float64("maxRMSE", 3.0, "Maximum RMSE in pixels for curve-fit filter.")
	maxDeviation := flag.Float64("maxDeviation", 8.0, "Maximum deviation from fitted curve in pixels.")
	maxAcceleration := flag.Float64("maxAcceleration", 2.0, "Maximum acceleration in pixels/frame² for curve-fit filter.")

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
		MinRSquared:     *minRSquared,
		MaxRMSE:         *maxRMSE,
		MaxDeviation:    *maxDeviation,
		MaxAcceleration: *maxAcceleration,
	}

	// Validate that outputGrid is specified when serverURL is not provided
	if *serverURL == "" && *outputGrid == "" {
		fmt.Println("Error: -outputGrid is required when serverURL is not specified")
		fmt.Println("Usage: go run main.go [flags] <file1.png> <file2.png> ...")
		os.Exit(1)
	}

	if *serverURL != "" {
		processRemotely(*serverURL, *requestID, fileArgs, config)
	} else {
		processLocally(fileArgs, config, *outputGrid)
	}
}

func processLocally(fileArgs []string, config newcast.ProcessConfig, outputGrid string) {
	fmt.Println("--- Processing locally ---")
	fmt.Printf("Processing %d input files\n", len(fileArgs))
	fmt.Printf("Running with parameters: maxFeatures=%d, minTrackLength=%d\n",
		config.MaxFeatures, config.MinTrackLength)
	fmt.Println("Filter type: curvefit (forced)")

	filteredTracks, width, height, err := newcast.ProcessFilesToTracks(fileArgs, config)
	if err != nil {
		fmt.Printf("Error processing files: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d filtered tracks.\n", len(filteredTracks))

	fmt.Println("Generating average flow grid from fitted tracks...")

	// Generate average flow grid from tracks (filteredTracks are already fitted)
	averageFlowGrid := newcast.GenerateFlowGridFromTracks(filteredTracks, width, height, config.BlurSigma)

	// Save the average flow grid to output file using the helper function
	if err := newcast.SaveAverageFlowGridToGzippedCBOR(averageFlowGrid, outputGrid); err != nil {
		fmt.Printf("Error saving average flow grid: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Average flow grid saved to %s\n", outputGrid)
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
		// "gridType":  "average", // Server should assume average or handle removal
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
	fmt.Println("\nTo fetch the average flow grid, use the following command:")
	fmt.Printf("curl -o average_flow_grid.png \"%s/average-flow-grid?id=%s\"\n", serverURL, requestID)
}