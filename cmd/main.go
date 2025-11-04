package main

import (
	"example/goflow/flow"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"

	"gocv.io/x/gocv"
)

func main() {
	// Run with os.Args, which includes the command name as the first element
	RunMain(os.Args)
}

// RunMain processes the command line arguments and performs the appropriate action.
func RunMain(args []string) error {
	// Remove the program name from the args to parse only the flags and arguments
	if len(args) < 1 {
		return fmt.Errorf("not enough arguments provided")
	}
	
	// Since flag.Parse() uses os.Args directly, we need to temporarily modify os.Args
	// We'll create a new approach using a custom flag set for testing
	return runMainWithArgs(args[1:]) // Skip the program name
}

func runMainWithArgs(args []string) error {
	// Create a new flag set to avoid conflicts with the global flag package
	fs := flag.NewFlagSet("", flag.ExitOnError)

	// --- Standard Flow Generation Flags ---
	outputPath := fs.String("output", "output_flow_map.png", "Path to save the output flow map image.")
	resolutionFactor := fs.Int("resolution-factor", 4, "The factor by which to downscale the images before processing.")

	// --- Forward Flow Transformation Flags ---
	forwardMode := fs.Bool("forward", false, "Enable forward optical flow transformation.")
	forwardInput := fs.String("forward-input-image", "", "Path to the input image for forward transformation.")
	forwardOutput := fs.String("forward-output-image", "forward_output.png", "Path to save the forward-transformed image.")
	forwardFactor := fs.Float64("forward-factor", 1.0, "Factor to scale the flow vectors in forward transformation.")

	// Parse the provided arguments
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse arguments: %w", err)
	}

	// --- MAIN LOGIC ---
	// Determine which mode to run based on the '-forward' flag
	if *forwardMode {
		// --- Forward Transformation Mode ---
		imagePaths := fs.Args()
		if len(imagePaths) != 1 || *forwardInput == "" {
			return fmt.Errorf("usage for forward mode: go run . -forward -forward-input-image <input.png> [other-flags] <flow_map.png>")
		}
		flowMapPath := imagePaths[0]

		log.Printf("Starting forward optical flow transformation...")
		log.Printf("Input image: %s", *forwardInput)
		log.Printf("Flow map: %s", flowMapPath)
		log.Printf("Output image: %s", *forwardOutput)
		log.Printf("Forward factor: %.2f", *forwardFactor)

		// Call the new forward function from the 'flow' package
		img, err := flow.ForwardTransform(*forwardInput, flowMapPath, *forwardFactor)
		if err != nil {
			return fmt.Errorf("error during forward transformation: %w", err)
		}

		// Save the resulting image
		file, err := os.Create(*forwardOutput)
		if err != nil {
			return fmt.Errorf("error creating output file for forward image: %w", err)
		}
		defer file.Close()

		if err := png.Encode(file, img); err != nil {
			return fmt.Errorf("error encoding forward image: %w", err)
		}

		log.Printf("Successfully saved forward-transformed image to %s\n", *forwardOutput)

	} else {
		// --- Standard Flow Generation Mode ---
		imagePaths := fs.Args()
		if len(imagePaths) < 2 {
			return fmt.Errorf("usage for standard mode: go run . [flags] <frame1.png> <frame2.png> ...")
		}

		log.Printf("Starting average optical flow generation for %d frames...\n", len(imagePaths))

		img, err := flow.GenerateAverageFlowMap(imagePaths, *resolutionFactor)
		if err != nil {
			return fmt.Errorf("error generating flow map: %w", err)
		}

		file, err := os.Create(*outputPath)
		if err != nil {
			return fmt.Errorf("error creating output file: %w", err)
		}
		defer file.Close()

		if err := png.Encode(file, img); err != nil {
			return fmt.Errorf("error encoding image: %w", err)
		}

		log.Printf("Successfully generated average flow map: %s\n", *outputPath)
	}

	return nil
}

// RunFlowGeneration runs the flow generation logic with given parameters for testing
func RunFlowGeneration(imagePaths []string, resolutionFactor int, outputPath string) error {
	img, err := flow.GenerateAverageFlowMap(imagePaths, resolutionFactor)
	if err != nil {
		return fmt.Errorf("error generating flow map: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("error encoding image: %w", err)
	}

	log.Printf("Successfully generated average flow map: %s\n", outputPath)
	return nil
}

// RunForwardTransform runs the forward transformation logic with given parameters for testing
func RunForwardTransform(inputImagePath, flowMapPath string, factor float64, outputImagePath string) error {
	img, err := flow.ForwardTransform(inputImagePath, flowMapPath, factor)
	if err != nil {
		return fmt.Errorf("error during forward transformation: %w", err)
	}

	file, err := os.Create(outputImagePath)
	if err != nil {
		return fmt.Errorf("error creating output file for forward image: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("error encoding forward image: %w", err)
	}

	log.Printf("Successfully saved forward-transformed image to %s\n", outputImagePath)
	return nil
}

// resizeImage loads an image and resizes it using gocv.
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
