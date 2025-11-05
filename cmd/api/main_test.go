package main

import (
	"bytes"
	"encoding/json"
	"example/goflow/flow"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Helper function to calculate Mean Squared Error (MSE) between two images
func calculateMSE(img1, img2 image.Image) (float64, error) {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()
	if !bounds1.Eq(bounds2) {
		return 0, fmt.Errorf("image bounds are not equal: %v vs %v", bounds1, bounds2)
	}

	var mse float64
	for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
		for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
			r1, g1, b1, _ := img1.At(x, y).RGBA()
			r2, g2, b2, _ := img2.At(x, y).RGBA()
			mse += math.Pow(float64(int(r1)-int(r2)), 2)
			mse += math.Pow(float64(int(g1)-int(g2)), 2)
			mse += math.Pow(float64(int(b1)-int(b2)), 2)
		}
	}
	return mse / (float64(bounds1.Dx()*bounds1.Dy()) * 3), nil
}

func TestFlowHandler(t *testing.T) {
	image1Path := "../../rainfall_data/2025-10-03T14:40:00Z.png"
	image2Path := "../../rainfall_data/2025-10-03T14:45:00Z.png"
	image3Path := "../../rainfall_data/2025-10-03T14:50:00Z.png"

	for _, path := range []string{image1Path, image2Path, image3Path} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("Required test image does not exist: %s", path)
		}
	}

	imagePaths := []string{image1Path, image2Path}
	requestBody, _ := json.Marshal(map[string][]string{
		"image_paths": imagePaths,
	})
	req, err := http.NewRequest("POST", "/flow", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(flowHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "image/png" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "image/png")
	}

	// Save the flow map to a temporary file
	flowMapFile, err := os.CreateTemp("", "flowmap-*.png")
	if err != nil {
		t.Fatalf("Failed to create temp file for flow map: %v", err)
	}
	defer os.Remove(flowMapFile.Name())

	// TeeReader is used to write to the file while also allowing the body to be decoded
	tee := io.TeeReader(rr.Body, flowMapFile)

	img, _, err := image.Decode(tee)
	if err != nil {
		t.Fatalf("Failed to decode response body as image: %v", err)
	}
	flowMapFile.Close() // Close the file so ForwardTransform can open it

	// Check that the flow map is not a zero image
	bounds := img.Bounds()
	isZeroImage := true
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if r != 0 || g != 0 || b != 0 || a != 0 {
				isZeroImage = false
				break
			}
		}
		if !isZeroImage {
			break
		}
	}
	if isZeroImage {
		t.Error("handler returned a zero image for the flow map")
	}

	// Apply forward transformation to the second image to predict the third
	transformedImage, err := flow.ForwardTransform(image2Path, flowMapFile.Name(), 1.0)
	if err != nil {
		t.Fatalf("ForwardTransform failed: %v", err)
	}

	// Load the original images for comparison
	img2, err := os.Open(image2Path)
	if err != nil {
		t.Fatalf("Failed to open second image: %v", err)
	}
	defer img2.Close()
	img2Decoded, err := png.Decode(img2)
	if err != nil {
		t.Fatalf("Failed to decode second image: %v", err)
	}

	img3, err := os.Open(image3Path)
	if err != nil {
		t.Fatalf("Failed to open third image: %v", err)
	}
	defer img3.Close()
	img3Decoded, err := png.Decode(img3)
	if err != nil {
		t.Fatalf("Failed to decode third image: %v", err)
	}

	// Compare the transformed image with the third image
	mseTransformedVs3, err := calculateMSE(transformedImage, img3Decoded)
	if err != nil {
		t.Fatalf("Failed to calculate MSE for transformed vs 3rd image: %v", err)
	}

	mse2Vs3, err := calculateMSE(img2Decoded, img3Decoded)
	if err != nil {
		t.Fatalf("Failed to calculate MSE for 2nd vs 3rd image: %v", err)
	}

	if mseTransformedVs3 >= mse2Vs3 {
		t.Errorf("Transformed image is not more similar to the 3rd image. MSE Transformed vs 3: %f, MSE 2 vs 3: %f", mseTransformedVs3, mse2Vs3)
	}
}

func TestTraceHandler(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("Failed to change directory to project root: %v", err)
	}
	defer os.Chdir(wd)

	imagePath := "rainfall_data/2025-10-03T14:40:00Z.png"
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		t.Fatalf("Required test image does not exist: %s", imagePath)
	}

	requestBody, _ := json.Marshal(map[string]interface{}{
		"image_path": imagePath,
		"origin": map[string]float64{
			"X": 10,
			"Y": 10,
		},
		"direction": map[string]float64{
			"X": 1,
			"Y": 0,
		},
		"fov_deg":  10,
		"distance": 100,
	})
	req, err := http.NewRequest("POST", "/trace", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(traceHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/json")
	}

	var respBody map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&respBody); err != nil {
		t.Fatalf("Failed to decode response body as JSON: %v", err)
	}

	if _, ok := respBody["projection"]; !ok {
		t.Error("response body does not contain 'projection' field")
	}

	if _, ok := respBody["triangle"]; !ok {
		t.Error("response body does not contain 'triangle' field")
	}
}

func TestTraceHandler_InvalidPath(t *testing.T) {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"image_path": "../../../../etc/passwd",
		"origin": map[string]float64{
			"X": 10,
			"Y": 10,
		},
		"direction": map[string]float64{
			"X": 1,
			"Y": 0,
		},
		"fov_deg":  10,
		"distance": 100,
	})
	req, err := http.NewRequest("POST", "/trace", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(traceHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}
