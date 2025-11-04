package main

import (
	"image/png"
	"os"
	"testing"
)

// TestForwardOpticalFlow tests the forward optical flow functionality by calling the functions directly
func TestForwardOpticalFlow(t *testing.T) {
	// Check if rainfall_data images exist
	image1Path := "../rainfall_data/2025-10-03T14:40:00Z.png"
	image2Path := "../rainfall_data/2025-10-03T14:45:00Z.png"
	image3Path := "../rainfall_data/2025-10-03T14:50:00Z.png"
	
	for _, path := range []string{image1Path, image2Path, image3Path} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Skipf("Required test image does not exist: %s", path)
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
	// The updated ForwardTransform function can now handle different resolutions
	outputPath := "forward_result.png"
	err2 := RunForwardTransform(image2Path, flowMapPath, 1.0, outputPath)
	if err2 != nil {
		t.Fatalf("Failed to apply forward transformation: %v", err2)
	}

	// Step 3: Verify that the output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Forward transformation output file was not created: %s", outputPath)
	}

	// Step 4: Optionally, save the third image for visual comparison
	tempImage3Path := "actual_third_image.png"
	// Just copy the third image for comparison, no processing needed
	inputFile, err := os.Open(image3Path)
	if err == nil {
		defer inputFile.Close()
		img, err := png.Decode(inputFile)
		if err == nil {
			outputFile, err := os.Create(tempImage3Path)
			if err == nil {
				defer os.Remove(tempImage3Path)
				defer outputFile.Close()
				if err := png.Encode(outputFile, img); err != nil {
					t.Logf("Could not save third image for comparison: %v", err)
				} else {
					t.Logf("For comparison, third image saved to: %s", tempImage3Path)
				}
			}
		}
	}

	// Clean up test files
	defer os.Remove(flowMapPath)

	t.Logf("Forward optical flow test completed successfully!")
	t.Logf("Generated flow map: %s", flowMapPath)
	t.Logf("Forward transformation result: %s", outputPath)
}

// TestForwardOpticalFlowArgs tests the forward optical flow functionality by using command line arguments
func TestForwardOpticalFlowArgs(t *testing.T) {
	// Check if rainfall_data images exist
	image1Path := "../rainfall_data/2025-10-03T14:40:00Z.png"
	image2Path := "../rainfall_data/2025-10-03T14:45:00Z.png"
	image3Path := "../rainfall_data/2025-10-03T14:50:00Z.png"
	
	for _, path := range []string{image1Path, image2Path, image3Path} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Skipf("Required test image does not exist: %s", path)
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

	// Step 4: Optionally, save the third image for visual comparison
	tempImage3Path := "actual_third_image_args.png"
	// Just copy the third image for comparison, no processing needed
	inputFile, err := os.Open(image3Path)
	if err == nil {
		defer inputFile.Close()
		img, err := png.Decode(inputFile)
		if err == nil {
			outputFile, err := os.Create(tempImage3Path)
			if err == nil {
				defer os.Remove(tempImage3Path)
				defer outputFile.Close()
				if err := png.Encode(outputFile, img); err != nil {
					t.Logf("Could not save third image for comparison: %v", err)
				} else {
					t.Logf("For comparison, third image saved to: %s", tempImage3Path)
				}
			}
		}
	}

	// Clean up test files
	defer os.Remove(flowMapPath)

	t.Logf("Forward optical flow test with command-line args completed successfully!")
	t.Logf("Generated flow map: %s", flowMapPath)
	t.Logf("Forward transformation result: %s", outputPath)
}