package main

import (
	"encoding/json"
	"example/goflow/flow"
	"flag"
	"fmt"
	"image/png"
	"log"
	"net/http"
	"strconv"
)

type FlowRequest struct {
	ImagePaths []string `json:"image_paths"`
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
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
