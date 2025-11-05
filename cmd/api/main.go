package main

import (
	"encoding/json"
	"example/goflow/flow"
	"example/goflow/trace"
	"flag"
	"fmt"
	"image/png"
	"log"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"gocv.io/x/gocv"
)

type FlowRequest struct {
	ImagePaths []string `json:"image_paths"`
}

type TraceRequest struct {
	ImagePath           string      `json:"image_path"`
	Origin              trace.Point `json:"origin"`
	Direction           trace.Point `json:"direction"`
	FieldOfViewAngleDEG float64     `json:"fov_deg"`
	Distance            float64     `json:"distance"`
}

type TraceResponse struct {
	Projection []float64      `json:"projection"`
	Triangle   trace.Triangle `json:"triangle"`
}

func traceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TraceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cleanPath := filepath.Clean(req.ImagePath)
	if !strings.HasPrefix(cleanPath, "rainfall_data/") {
		http.Error(w, "Invalid image path", http.StatusBadRequest)
		return
	}

	mat := gocv.IMRead(cleanPath, gocv.IMReadGrayScale)
	if mat.Empty() {
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		return
	}
	defer mat.Close()

	img := make([][]float64, mat.Rows())
	for i := 0; i < mat.Rows(); i++ {
		img[i] = make([]float64, mat.Cols())
		for j := 0; j < mat.Cols(); j++ {
			img[i][j] = float64(mat.GetUCharAt(i, j))
		}
	}

	projection, triangle, err := trace.ProjectAngularSearch(img, req.Origin, req.Direction, req.FieldOfViewAngleDEG*math.Pi/180.0, req.Distance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := TraceResponse{
		Projection: projection,
		Triangle:   triangle,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func flowHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req FlowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.ImagePaths) < 2 {
		http.Error(w, "At least two image paths are required", http.StatusBadRequest)
		return
	}

	resolutionFactorStr := r.URL.Query().Get("resn")
	resolutionFactor, err := strconv.Atoi(resolutionFactorStr)
	if err != nil || resolutionFactor <= 0 {
		resolutionFactor = 4
	}

	img, err := flow.GenerateAverageFlowMap(req.ImagePaths, resolutionFactor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	if err := png.Encode(w, img); err != nil {
		http.Error(w, "Failed to encode image", http.StatusInternalServerError)
		return
	}
}

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	http.HandleFunc("/flow", flowHandler)
	http.HandleFunc("/trace", traceHandler)
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
