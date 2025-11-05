package main

import (
	"example/goflow/flow"
	"fmt"
	"image"
	"image/png"
	"os"

	"gocv.io/x/gocv"
)

const originalWidth = 1024
const originalHeight = 1024

// resizeImage loads an image and resizes it using gocv to match flow map dimensions.
func resizeImage(imgPath string, width, height int) (image.Image, error) {
	mat := gocv.IMRead(imgPath, gocv.IMReadColor)
	if mat.Empty() {
		return nil, fmt.Errorf("failed to read image %s with gocv", imgPath)
	}
	defer mat.Close()

	resizedMat := gocv.NewMat()
	defer resizedMat.Close()

	gocv.Resize(mat, &resizedMat, image.Pt(width, height), 0, 0, gocv.InterpolationArea)

	return resizedMat.ToImage()
}

func main() {
	// Define the paths for the three images - need to go up 2 directories to reach rainfall_data
	image1Path := "../../rainfall_data/2025-10-03T14:40:00Z.png"
	image2Path := "../../rainfall_data/2025-10-03T14:45:00Z.png"
	image3Path := "../../rainfall_data/2025-10-03T14:50:00Z.png"

	// Check if files exist
	for _, path := range []string{image1Path, image2Path, image3Path} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("Error: File %s does not exist\n", path)
			os.Exit(1)
		}
	}

	// Use a resolution factor to determine flow map size
	resolutionFactor := 4
	scaledWidth := originalWidth / resolutionFactor
	scaledHeight := originalHeight / resolutionFactor

	// 1. Generate the optical flow from image1 to image2
	fmt.Println("Generating optical flow from image 1 to image 2...")
	flowMap, err := flow.GenerateAverageFlowMap([]string{image1Path, image2Path}, resolutionFactor)
	if err != nil {
		fmt.Printf("Error generating flow map: %v\n", err)
		os.Exit(1)
	}

	// 2. Save the flow map to a temporary file for the forward transform
	tempFlowPath := "temp_flow_map.png"
	file, err := os.Create(tempFlowPath)
	if err != nil {
		fmt.Printf("Error creating temporary flow map file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tempFlowPath) // Clean up the temp file
	defer file.Close()

	if err := png.Encode(file, flowMap); err != nil {
		fmt.Printf("Error encoding flow map: %v\n", err)
		os.Exit(1)
	}

	// 3. Resize image2 to match the flow map dimensions
	fmt.Printf("Resizing image 2 to match flow map dimensions (%d x %d)...\n", scaledWidth, scaledHeight)
	resizedImage2, err := resizeImage(image2Path, scaledWidth, scaledHeight)
	if err != nil {
		fmt.Printf("Error resizing image 2: %v\n", err)
		os.Exit(1)
	}

	// 4. Save the resized image 2 to a temporary file
	tempImage2Path := "temp_image2_resized.png"
	tempImage2File, err := os.Create(tempImage2Path)
	if err != nil {
		fmt.Printf("Error creating temporary resized image 2 file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tempImage2Path) // Clean up the temp file
	defer tempImage2File.Close()

	if err := png.Encode(tempImage2File, resizedImage2); err != nil {
		fmt.Printf("Error encoding resized image 2: %v\n", err)
		os.Exit(1)
	}

	// 5. Apply forward transformation to the resized image2 using the flow map from image1->image2
	// This should generate something similar to the resized version of image3 if the motion pattern continues
	fmt.Println("Applying forward transformation to resized image 2...")
	forwardedImage, err := flow.ForwardTransform(tempImage2Path, tempFlowPath, 1.0)
	if err != nil {
		fmt.Printf("Error applying forward transform: %v\n", err)
		os.Exit(1)
	}

	// 6. Save the forward-transformed image
	forwardedPath := "forwarded_result.png"
	forwardedFile, err := os.Create(forwardedPath)
	if err != nil {
		fmt.Printf("Error creating forwarded result file: %v\n", err)
		os.Exit(1)
	}
	defer forwardedFile.Close()

	if err := png.Encode(forwardedFile, forwardedImage); err != nil {
		fmt.Printf("Error encoding forwarded image: %v\n", err)
		os.Exit(1)
	}

	// 7. Resize image3 to match the flow map dimensions for comparison
	fmt.Printf("Resizing image 3 to match flow map dimensions (%d x %d) for comparison...\n", scaledWidth, scaledHeight)
	resizedImage3, err := resizeImage(image3Path, scaledWidth, scaledHeight)
	if err != nil {
		fmt.Printf("Error resizing image 3: %v\n", err)
		os.Exit(1)
	}

	// 8. Save the resized image 3 for comparison
	resizedImage3Path := "actual_image3_resized.png"
	resizedImage3File, err := os.Create(resizedImage3Path)
	if err != nil {
		fmt.Printf("Error creating resized image 3 file: %v\n", err)
		os.Exit(1)
	}
	defer resizedImage3File.Close()

	if err := png.Encode(resizedImage3File, resizedImage3); err != nil {
		fmt.Printf("Error encoding resized image 3: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Forward transformation complete!")
	fmt.Printf("Generated: %s\n", forwardedPath)
	fmt.Printf("Expected: %s\n", resizedImage3Path)
	fmt.Println("Compare the generated forwarded_result.png with the actual resized third image to see similarity.")
}
