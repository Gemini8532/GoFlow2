package main

import (
	"example/goflow/flow"
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
)

func main() {
	// Define and parse command-line flags
	outputPath := flag.String("output", "output_flow_map.png", "Path to save the output flow map image.")
	resolutionFactor := flag.Int("resolution-factor", 4, "The factor by which to downscale the images before processing.")
	flag.Parse()

	imagePaths := flag.Args()

	if len(imagePaths) < 2 {
		fmt.Println("Usage: go run . [flags] <frame1.png> <frame2.png> [frame3.png ...]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	log.Printf("Starting average optical flow generation for %d frames...\n", len(imagePaths))

	// Call the module's new function
	img, err := flow.GenerateAverageFlowMap(imagePaths, *resolutionFactor)
	if err != nil {
		log.Fatalf("Error generating flow map: %v", err)
	}

	// Save the resulting image to the output path
	file, err := os.Create(*outputPath)
	if err != nil {
		log.Fatalf("Error creating output file: %v", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		log.Fatalf("Error encoding image: %v", err)
	}

	log.Printf("Successfully generated average flow map: %s\n", *outputPath)
}
