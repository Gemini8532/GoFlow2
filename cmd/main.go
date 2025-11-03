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
	// --- Standard Flow Generation Flags ---
	outputPath := flag.String("output", "output_flow_map.png", "Path to save the output flow map image.")
	resolutionFactor := flag.Int("resolution-factor", 4, "The factor by which to downscale the images before processing.")

	// --- Reverse Flow Transformation Flags ---
	reverseMode := flag.Bool("reverse", false, "Enable reverse optical flow transformation.")
	reverseInput := flag.String("reverse-input-image", "", "Path to the input image for reverse transformation.")
	reverseOutput := flag.String("reverse-output-image", "reverse_output.png", "Path to save the reverse-transformed image.")
	reverseFactor := flag.Float64("reverse-factor", 1.0, "Factor to scale the flow vectors in reverse transformation.")

	flag.Parse()

	// --- MAIN LOGIC ---
	// Determine which mode to run based on the '-reverse' flag
	if *reverseMode {
		// --- Reverse Transformation Mode ---
		imagePaths := flag.Args()
		if len(imagePaths) != 1 || *reverseInput == "" {
			fmt.Println("Usage for reverse mode: go run . -reverse -reverse-input-image <input.png> [other-flags] <flow_map.png>")
			flag.PrintDefaults()
			os.Exit(1)
		}
		flowMapPath := imagePaths[0]

		log.Printf("Starting reverse optical flow transformation...")
		log.Printf("Input image: %s", *reverseInput)
		log.Printf("Flow map: %s", flowMapPath)
		log.Printf("Output image: %s", *reverseOutput)
		log.Printf("Reverse factor: %.2f", *reverseFactor)

		// Call the new reverse function from the 'flow' package
		img, err := flow.ReverseTransform(*reverseInput, flowMapPath, *reverseFactor)
		if err != nil {
			log.Fatalf("Error during reverse transformation: %v", err)
		}

		// Save the resulting image
		file, err := os.Create(*reverseOutput)
		if err != nil {
			log.Fatalf("Error creating output file for reverse image: %v", err)
		}
		defer file.Close()

		if err := png.Encode(file, img); err != nil {
			log.Fatalf("Error encoding reverse image: %v", err)
		}

		log.Printf("Successfully saved reverse-transformed image to %s\n", *reverseOutput)

	} else {
		// --- Standard Flow Generation Mode ---
		imagePaths := flag.Args()
		if len(imagePaths) < 2 {
			fmt.Println("Usage for standard mode: go run . [flags] <frame1.png> <frame2.png> ...")
			flag.PrintDefaults()
			os.Exit(1)
		}

		log.Printf("Starting average optical flow generation for %d frames...\n", len(imagePaths))

		img, err := flow.GenerateAverageFlowMap(imagePaths, *resolutionFactor)
		if err != nil {
			log.Fatalf("Error generating flow map: %v", err)
		}

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
}
