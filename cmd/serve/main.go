package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"example/goflow/newcast" // Adjust this import path if your module name is different
)

// Add a struct to hold the data needed for track visualization
type TrackVisuals struct {
	Tracks []*newcast.Track
	Width  int
	Height int
}

// RequestData holds all the data associated with a single processing request.
type RequestData struct {
	AverageFlowGrid *newcast.AverageFlowGrid
	Visuals         TrackVisuals
}

// Define a struct to hold the global state for our server
type Server struct {
	requests map[string]*RequestData
	mu       sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		requests: make(map[string]*RequestData),
	}
}

func (s *Server) processHandler(w http.ResponseWriter, r *http.Request) {
	// 1. CORS Headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Only POST method is supported", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Filenames []string               `json:"filenames"`
		ID        string                 `json:"id"`
		Config    *newcast.ProcessConfig `json:"config"`
		// GridType field removed, assumed "average"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode request body: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Filenames) == 0 || req.ID == "" {
		http.Error(w, "Filenames and ID are required", http.StatusBadRequest)
		return
	}

	log.Printf("Received process request for ID: %s with %d files", req.ID, len(req.Filenames))

	// Use provided config or default
	config := newcast.ProcessConfig{
		MaxFeatures:    200,
		MinTrackLength: 6,
	}
	if req.Config != nil {
		config = *req.Config
	}

	filteredTracks, width, height, err := newcast.ProcessFilesToTracks(req.Filenames, config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process files to tracks: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Processed %d tracks. Image dimensions: %dx%d", len(filteredTracks), width, height)

	if len(req.Filenames) <= 1 {
		http.Error(w, "At least two filenames are required to generate flow (need current and next frame)", http.StatusBadRequest)
		return
	}

	requestData := &RequestData{
		Visuals: TrackVisuals{
			Tracks: filteredTracks,
			Width:  width,
			Height: height,
		},
	}

	log.Println("Generating AverageFlowGrid from fitted tracks")
	// Generate average flow grid from tracks (tracks are already fitted)
	averageFlowGrid := newcast.GenerateFlowGridFromTracks(filteredTracks, width, height, config.BlurSigma)
	requestData.AverageFlowGrid = averageFlowGrid

	s.mu.Lock()
	s.requests[req.ID] = requestData
	s.mu.Unlock()

	log.Printf("Data for ID %s generated and stored successfully.", req.ID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Data generated and stored"})
}

func (s *Server) trackVisHandler(w http.ResponseWriter, r *http.Request) {
	// 1. CORS Headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Only GET method is supported", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID is a required query parameter", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	requestData, ok := s.requests[id]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, fmt.Sprintf("No track visualization data found for ID: %s", id), http.StatusNotFound)
		return
	}
	visuals := requestData.Visuals

	trackImg := newcast.VisualizeTracks(visuals.Tracks, visuals.Width, visuals.Height)
	defer trackImg.Close()

	buf, err := newcast.MatToPNG(trackImg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode visualization to PNG: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(buf)
	log.Printf("Served track visualization for ID: %s", id)
}

func (s *Server) averageFlowGridHandler(w http.ResponseWriter, r *http.Request) {
	// 1. CORS Headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Only GET method is supported", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID is a required query parameter", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	requestData, ok := s.requests[id]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, fmt.Sprintf("No data found for ID: %s", id), http.StatusNotFound)
		return
	}
	if requestData.AverageFlowGrid == nil {
		http.Error(w, fmt.Sprintf("AverageFlowGrid not generated for ID: %s", id), http.StatusNotFound)
		return
	}
	avgFlowGrid := requestData.AverageFlowGrid

	// Handle resizing
	heightStr := r.URL.Query().Get("height")
	if heightStr != "" {
		height, err := strconv.Atoi(heightStr)
		if err != nil {
			http.Error(w, "Invalid height parameter", http.StatusBadRequest)
			return
		}
		if height > 0 {
			aspectRatio := float64(avgFlowGrid.Width) / float64(avgFlowGrid.Height)
			width := int(float64(height) * aspectRatio)
			avgFlowGrid = avgFlowGrid.Resize(width, height)
		}
	}
	// TEMPORARILY DISABLED: Testing if resize is causing corruption
	// else {
	// 	// Default resizing if no height is specified
	// 	aspectRatio := float64(avgFlowGrid.Width) / float64(avgFlowGrid.Height)
	// 	height := 256
	// 	width := int(float64(height) * aspectRatio)
	// 	avgFlowGrid = avgFlowGrid.Resize(width, height)
	// }

	// Send gzipped CBOR format instead of the old binary format to improve efficiency and data integrity
	// Use avgFlowGrid directly for CBOR encoding
	cborData, err := avgFlowGrid.MarshalGzippedCBOR()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal to gzipped CBOR: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("X-Grid-Width", strconv.Itoa(avgFlowGrid.Width))
	w.Header().Set("X-Grid-Height", strconv.Itoa(avgFlowGrid.Height))
	w.Write(cborData)
	log.Printf("Served average flow grid for ID: %s", id)
}

func main() {
	server := NewServer()

	http.HandleFunc("/process", server.processHandler)
	// /vector-frame removed
	http.HandleFunc("/tracks-visualization", server.trackVisHandler)
	http.HandleFunc("/average-flow-grid", server.averageFlowGridHandler)

	fmt.Println("Serving vector data API on :9093")
	log.Fatal(http.ListenAndServe(":9093", nil))
}