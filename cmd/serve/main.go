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

// Define a struct to hold the global state for our server
type Server struct {
	flowGrids    map[string]*newcast.FlowGrid
	trackVisuals map[string]TrackVisuals
	mu           sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		flowGrids:    make(map[string]*newcast.FlowGrid),
		trackVisuals: make(map[string]TrackVisuals),
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
		Filenames []string             `json:"filenames"`
		ID        string               `json:"id"`
		Config    *newcast.ProcessConfig `json:"config"`
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
		MaxFeatures:      200,
		Smoothness:       0.5,
		FilterType:       "smoothness",
		MaxAngle:         0.8,
		GridCellSize:     64,
		MinTracksPerCell: 2,
		MaxTracksPerCell: 5,
		MinTrackLength:   6,
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
	numTimeSteps := len(req.Filenames) - 1 // Flow is generated between frames

	flowProcessor := newcast.NewFlowProcessor(width, height, numTimeSteps)
	flowProcessor.ProcessTracks(filteredTracks)
	flowProcessor.CalculateAverages()
	flowProcessor.FillGaps(1) // Fill gaps with a radius of 1

	s.mu.Lock()
	s.flowGrids[req.ID] = flowProcessor.Grid
	s.trackVisuals[req.ID] = TrackVisuals{
		Tracks: filteredTracks,
		Width:  width,
		Height: height,
	}
	s.mu.Unlock()

	log.Printf("FlowGrid and track visuals for ID %s generated and stored successfully.", req.ID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "FlowGrid generated and stored"})
}

func (s *Server) vectorFrameHandler(w http.ResponseWriter, r *http.Request) {
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
	tStr := r.URL.Query().Get("t")

	if id == "" || tStr == "" {
		http.Error(w, "ID and time index 't' are required query parameters", http.StatusBadRequest)
		return
	}

	t, err := strconv.Atoi(tStr)
	if err != nil {
		http.Error(w, "Invalid time index 't'", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	flowGrid, ok := s.flowGrids[id]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, fmt.Sprintf("No FlowGrid found for ID: %s", id), http.StatusNotFound)
		return
	}

	if t < 0 || t >= flowGrid.Time {
		http.Error(w, fmt.Sprintf("Time index %d out of bounds (0 to %d)", t, flowGrid.Time-1), http.StatusBadRequest)
		return
	}

	// Create a newcast.Frame from the FlowGrid time slice
	frame := newcast.NewFrame(flowGrid.Width, flowGrid.Height)
	startIndex := (t * flowGrid.Height) * flowGrid.Width
	endIndex := startIndex + (flowGrid.Height * flowGrid.Width)
	if endIndex > len(flowGrid.Data) {
		endIndex = len(flowGrid.Data)
	}

	// Copy the relevant time slice data into the frame
	copy(frame.Data, flowGrid.Data[startIndex:endIndex])

	pngBytes, err := frame.MarshalPNG()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal PNG: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(pngBytes)
	log.Printf("Served vector frame for ID: %s, time: %d", id, t)
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
	visuals, ok := s.trackVisuals[id]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, fmt.Sprintf("No track visualization data found for ID: %s", id), http.StatusNotFound)
		return
	}

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

func main() {
	server := NewServer()

	http.HandleFunc("/process", server.processHandler)
	http.HandleFunc("/vector-frame", server.vectorFrameHandler)
	http.HandleFunc("/tracks-visualization", server.trackVisHandler)

	fmt.Println("Serving vector data API on :9093")
	log.Fatal(http.ListenAndServe(":9093", nil))
}
