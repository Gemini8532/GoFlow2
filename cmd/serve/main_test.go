package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"example/goflow/newcast"
	"github.com/stretchr/testify/require"
)

func TestProcessAndFetchRainfallData(t *testing.T) {
	// 1. Setup Server
	server := NewServer()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/process" {
			server.processHandler(w, r)
		} else if r.URL.Path == "/vector-frame" {
			server.vectorFrameHandler(w, r)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	// 2. Find rainfall data files
	files, err := filepath.Glob("../../rainfall_data/*.png")
	require.NoError(t, err)
	require.True(t, len(files) > 1, "Not enough rainfall data found to test")

	// 3. Make /process request
	testID := "rainfall-test"
	processReqBody := map[string]interface{}{
		"filenames": files,
		"id":        testID,
	}
	reqBytes, _ := json.Marshal(processReqBody)

	resp, err := http.Post(ts.URL+"/process", "application/json", bytes.NewBuffer(reqBytes))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	// 4. Make /vector-frame request
	resp, err = http.Get(ts.URL + "/vector-frame?id=" + testID + "&t=0")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	// 5. Verify response
	require.Equal(t, "image/png", resp.Header.Get("Content-Type"))
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.True(t, len(bodyBytes) > 0, "Response body should not be empty")

	// Optional: Decode to ensure it's a valid grid-encoded PNG
	_, err = newcast.UnmarshalPNG(bodyBytes)
	require.NoError(t, err, "Failed to decode the returned PNG")
}

func TestProcessAndFetchWellKnownData(t *testing.T) {
	// 1. Setup Server
	server := NewServer()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/process" {
			server.processHandler(w, r)
		} else if r.URL.Path == "/vector-frame" {
			server.vectorFrameHandler(w, r)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	// 2. Define well-known files
	files := []string{"../../test_data/centered.png", "../../test_data/shifted.png"}
	
	// Ensure files exist
	for _, f := range files {
		_, err := os.Stat(f)
		require.NoError(t, err, "Test file %s not found", f)
	}

	// 3. Make /process request
	testID := "well-known-test"
	config := newcast.ProcessConfig{
		MaxFeatures:    50,
		MinTrackLength: 2,
		FilterType:     "smoothness",
		Smoothness:     1.0, // Very lenient
	}
	processReqBody := map[string]interface{}{
		"filenames": files,
		"id":        testID,
		"config":    config,
	}
	reqBytes, _ := json.Marshal(processReqBody)
	
	resp, err := http.Post(ts.URL+"/process", "application/json", bytes.NewBuffer(reqBytes))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	// 4. Make /vector-frame request
	resp, err = http.Get(ts.URL + "/vector-frame?id=" + testID + "&t=0")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	// 5. Verify response and vector values
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	frame, err := newcast.UnmarshalPNG(bodyBytes)
	require.NoError(t, err)
	
	// The features in shifted.png are moved 20px right and 10px down from centered.png
	// We expect the vectors to reflect this shift: Vx=+20, Vy=+10
	foundVector := false
	for i, vec := range frame.Data {
		if vec.Vx != 0 || vec.Vy != 0 {
			foundVector = true
			// Allow for some tolerance due to the nature of the optical flow algorithm
			require.InDelta(t, 20.0, vec.Vx, 2.0, "Vx should be close to 20 for vector %d", i)
			require.InDelta(t, 10.0, vec.Vy, 2.0, "Vy should be close to 10 for vector %d", i)
			// If we found one correct vector, we can stop.
			break
		}
	}
	require.True(t, foundVector, "Expected to find at least one non-zero vector with the correct properties")
}
