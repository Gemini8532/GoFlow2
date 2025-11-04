package main

import (
	"fmt"
	"image"
	"image/png"
	"math"
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

// TestForwardOpticalFlow tests the forward optical flow functionality by calling the functions directly
func TestForwardOpticalFlow(t *testing.T) {
	// Check if rainfall_data images exist
	image1Path := "../../rainfall_data/2025-10-03T14:40:00Z.png"
	image2Path := "../../rainfall_data/2025-10-03T14:45:00Z.png"
	image3Path := "../../rainfall_data/2025-10-03T14:50:00Z.png"

	for _, path := range []string{image1Path, image2Path, image3Path} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("Required test image does not exist: %s", path)
		}
	}

	// Use the same resolution factor as in the original test
	resolutionFactor := 4

	// Step 1: Generate flow map from image 1 to image 2 (at lower resolution)
	flowMapPath := "test_flow_map.png"

	err1 := RunFlowGeneration([]string{image1Path, image2Path}, resolutionFactor, flowMapPath)
	if err1 != nil {
		t.Fatalf("Failed to generate flow map: %v", err1)
	}

	// Step 2: Apply forward transformation to the full resolution image 2 using the lower resolution flow map
	outputPath := "forward_result.png"
	err2 := RunForwardTransform(image2Path, flowMapPath, 1.0, outputPath)
	if err2 != nil {
		t.Fatalf("Failed to apply forward transformation: %v", err2)
	}

	// Step 3: Verify that the output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Forward transformation output file was not created: %s", outputPath)
	}

	// Step 4: Compare the transformed image with the actual third image
	imgTransformed, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Failed to open transformed image: %v", err)
	}
	defer imgTransformed.Close()
	imgTransformedDecoded, err := png.Decode(imgTransformed)
	if err != nil {
		t.Fatalf("Failed to decode transformed image: %v", err)
	}

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

	mseTransformedVs3, err := calculateMSE(imgTransformedDecoded, img3Decoded)
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

	// Clean up test files
	defer os.Remove(flowMapPath)
	defer os.Remove(outputPath)

	t.Logf("Forward optical flow test completed successfully!")
	t.Logf("Generated flow map: %s", flowMapPath)
	t.Logf("Forward transformation result: %s", outputPath)
}

// TestForwardOpticalFlowArgs tests the forward optical flow functionality by using command line arguments
func TestForwardOpticalFlowArgs(t *testing.T) {
	// Check if rainfall_data images exist
	image1Path := "../../rainfall_data/2025-10-03T14:40:00Z.png"
	image2Path := "../../rainfall_data/2025-10-03T14:45:00Z.png"
	image3Path := "../../rainfall_data/2025-10-03T14:50:00Z.png"

	for _, path := range []string{image1Path, image2Path, image3Path} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("Required test image does not exist: %s", path)
		}
	}

	// Step 1: Generate flow map from image 1 to image 2 using command line arguments
	flowMapPath := "test_flow_map_args.png"

	args1 := []string{"-output", flowMapPath, "-resolution-factor", "4", image1Path, image2Path}
	err1 := runMainWithArgs(args1)
	if err1 != nil {
		t.Fatalf("Failed to generate flow map with cmd args: %v", err1)
	}

	// Step 2: Apply forward transformation to the full resolution image 2 using command line arguments
	outputPath := "forward_result_args.png"

	args2 := []string{"-forward", "-forward-input-image", image2Path, "-forward-output-image", outputPath, flowMapPath}
	err2 := runMainWithArgs(args2)
	if err2 != nil {
		t.Fatalf("Failed to apply forward transformation with cmd args: %v", err2)
	}

	// Step 3: Verify that the output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Forward transformation output file was not created: %s", outputPath)
	}

	// Step 4: Compare the transformed image with the actual third image
	imgTransformed, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Failed to open transformed image: %v", err)
	}
	defer imgTransformed.Close()
	imgTransformedDecoded, err := png.Decode(imgTransformed)
	if err != nil {
		t.Fatalf("Failed to decode transformed image: %v", err)
	}

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

	mseTransformedVs3, err := calculateMSE(imgTransformedDecoded, img3Decoded)
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

	// Clean up test files
	defer os.Remove(flowMapPath)
	defer os.Remove(outputPath)

	t.Logf("Forward optical flow test with command-line args completed successfully!")
	t.Logf("Generated flow map: %s", flowMapPath)
	t.Logf("Forward transformation result: %s", outputPath)
}