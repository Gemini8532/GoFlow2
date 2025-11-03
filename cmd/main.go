package main

import (
	"example/goflow/flow"
	"fmt"
	"image/png"
	"log"
	"os"
)

func main() {
	// Updated usage: output path is first, followed by all image frames.
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run . <output.png> <frame1.png> <frame2.png> [frame3.png ...]")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	imagePaths := os.Args[2:]

	// Define the downscaling factor.
	// 4 will reduce 1024x1024 to 256x256.
	const resolutionFactor = 4

	log.Printf("Starting average optical flow generation for %d frames...\n", len(imagePaths))

	// Call the module's new function
	img, err := flow.GenerateAverageFlowMap(imagePaths, resolutionFactor)
	if err != nil {
		log.Fatalf("Error generating flow map: %v", err)
	}

	// Save the resulting image to the output path
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Error creating output file: %v", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		log.Fatalf("Error encoding image: %v", err)
	}

	log.Printf("Successfully generated average flow map: %s\n", outputPath)
}
