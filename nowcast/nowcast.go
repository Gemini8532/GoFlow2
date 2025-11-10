package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png" // Using png format as specified
	"log"
	"math"
	"os"
	"sort"

	"gocv.io/x/gocv"
)

// GridVector holds the extrapolated motion parameters for a single grid cell.
// Vx, Vy are the velocities (pixels/frame) at t=0 (the last frame).
// Ax, Ay are the accelerations (pixels/frame^2) derived from the fit.
type GridVector struct {
	Vx float64
	Vy float64
	Ax float64
	Ay float64
}

// ExtrapolationData holds the complete set of motion vectors for the grid.
type ExtrapolationData struct {
	GridRes int // The resolution (e.g., 64) of the grid
	// Data maps a grid coordinate (e.g., [0,0], [0,1]) to its motion vector
	Data map[image.Point]GridVector
}

// LoadGrayscaleImage loads a PNG, decodes it, and converts it to a grayscale gocv.Mat.
func LoadGrayscaleImage(filePath string) (gocv.Mat, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("failed to open image file %s: %w", filePath, err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("failed to decode png %s: %w", filePath, err)
	}

	// Convert to RGBA first, as gocv.ImageToMatRGBA is robust
	rgbaImg, ok := img.(*image.RGBA)
	if !ok {
		// If not RGBA, create a new RGBA image and draw the decoded image onto it
		bounds := img.Bounds()
		rgbaImg = image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgbaImg.Set(x, y, img.At(x, y))
			}
		}
	}

	matRGBA, err := gocv.ImageToMatRGBA(rgbaImg)
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("failed to convert image to MatRGBA: %w", err)
	}
	defer matRGBA.Close()

	matGray := gocv.NewMat()
	gocv.CvtColor(matRGBA, &matGray, gocv.ColorRGBAToGray)
	return matGray, nil
}

// TrimmedMean calculates the mean of a slice of float64s, excluding outliers.
// trimFactor (0.0 to 0.5) specifies the fraction of data to trim from each end.
func TrimmedMean(data []float64, trimFactor float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	// Create a copy to avoid modifying the original slice
	sortedData := make([]float64, len(data))
	copy(sortedData, data)
	sort.Float64s(sortedData)

	trim := int(math.Floor(float64(len(sortedData)) * trimFactor))
	if trim*2 >= len(sortedData) {
		// Trim factor is too high, return median or simple mean
		if len(sortedData) > 0 {
			return sortedData[len(sortedData)/2] // Return median as a fallback
		}
		return 0.0
	}

	trimmedSlice := sortedData[trim : len(sortedData)-trim]

	sum := 0.0
	for _, v := range trimmedSlice {
		sum += v
	}

	return sum / float64(len(trimmedSlice))
}

// CalculateGridVelocities aggregates pixel-wise flow into a grid using trimmed mean.
func CalculateGridVelocities(flow gocv.Mat, gridRes int) (map[image.Point]GridVector, error) {
	if flow.Empty() || flow.Type() != gocv.MatTypeCV32FC2 {
		return nil, fmt.Errorf("invalid flow matrix: empty or wrong type")
	}

	rows := flow.Rows()
	cols := flow.Cols()
	if rows == 0 || cols == 0 {
		return nil, fmt.Errorf("flow matrix has zero dimensions")
	}

	// This map will temporarily hold all flow vectors for each grid cell
	// The key is the grid coordinate (e.g., 0,0)
	// The value contains slices of all Vx and Vy values in that cell
	type cellData struct {
		Vx []float64
		Vy []float64
	}
	gridData := make(map[image.Point]*cellData)

	// Determine block size
	blockWidth := float64(cols) / float64(gridRes)
	blockHeight := float64(rows) / float64(gridRes)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Calculate which grid cell this pixel belongs to
			gridX := int(math.Floor(float64(x) / blockWidth))
			gridY := int(math.Floor(float64(y) / blockHeight))
			pt := image.Point{X: gridX, Y: gridY}

			// Get the flow vector at (y, x)
			// Farneback returns a 2-channel float matrix (CV_32FC2)
			vec := flow.GetVecfAt(y, x)
			vx := float64(vec[0])
			vy := float64(vec[1])

			// Initialize the struct for this grid cell if it doesn't exist
			if _, ok := gridData[pt]; !ok {
				gridData[pt] = &cellData{
					Vx: make([]float64, 0, 100), // Pre-allocate
					Vy: make([]float64, 0, 100),
				}
			}

			// Append the flow vector
			gridData[pt].Vx = append(gridData[pt].Vx, vx)
			gridData[pt].Vy = append(gridData[pt].Vy, vy)
		}
	}

	// Now, calculate the trimmed mean for each grid cell
	gridVelocities := make(map[image.Point]GridVector)
	for pt, data := range gridData {
		if len(data.Vx) > 0 { // Ensure there's data to process
			vxMean := TrimmedMean(data.Vx, 0.1) // Trim 10% from each end
			vyMean := TrimmedMean(data.Vy, 0.1)
			// We only have velocity (Vx, Vy) at this stage. Ax, Ay will be 0.
			gridVelocities[pt] = GridVector{Vx: vxMean, Vy: vyMean, Ax: 0, Ay: 0}
		}
	}

	return gridVelocities, nil
}

// FitPolynomial performs a linear regression (1st-order polynomial fit) on the data.
// v(t) = a*t + b
// Returns 'b' (intercept, value at t=0) and 'a' (slope, acceleration).
func FitPolynomial(times []float64, values []float64) (intercept float64, slope float64) {
	n := float64(len(times))
	if n == 0 {
		return 0, 0
	}
	if n == 1 {
		return values[0], 0 // No slope possible, return the value as intercept
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i := range times {
		sumX += times[i]
		sumY += values[i]
		sumXY += times[i] * values[i]
		sumX2 += times[i] * times[i]
	}

	// Standard linear regression formulas
	denominator := (n*sumX2 - sumX*sumX)
	if math.Abs(denominator) < 1e-9 {
		// Avoid division by zero; vertical line, likely all times are the same.
		// Return average value as intercept and zero slope.
		return sumY / n, 0
	}

	slope = (n*sumXY - sumX*sumY) / denominator
	intercept = (sumY - slope*sumX) / n

	return intercept, slope
}

// ProcessImages is the main function to generate the extrapolation data.
// imagePaths: A list of file paths to the radar images, ordered from oldest to newest.
// gridRes: The desired grid resolution (e.g., 64 for a 64x64 grid).
// timeStep: The time in minutes (or any unit) between frames (e.g., 5.0).
func ProcessImages(imagePaths []string, gridRes int, timeStep float64) (ExtrapolationData, error) {
	numFrames := len(imagePaths)
	if numFrames < 3 {
		// Need at least 3 frames to get 2 flow fields to fit a line (v, a)
		return ExtrapolationData{}, fmt.Errorf("at least 3 image frames are required, but got %d", numFrames)
	}
	numFlows := numFrames - 1

	// --- 1. Calculate all flow fields ---
	flowFields := make([]gocv.Mat, numFlows)
	prevImg, err := LoadGrayscaleImage(imagePaths[0])
	if err != nil {
		return ExtrapolationData{}, err
	}
	defer prevImg.Close()

	for i := 1; i < numFrames; i++ {
		currImg, err := LoadGrayscaleImage(imagePaths[i])
		if err != nil {
			return ExtrapolationData{}, err
		}
		// Note: currImg will be closed at the end of the loop iteration

		flow := gocv.NewMat()
		// Farneback parameters (tuned for general use)
		// pyr_scale=0.5, levels=3, winsize=15, iterations=3, poly_n=5, poly_sigma=1.2, flags=0
		gocv.CalcOpticalFlowFarneback(prevImg, currImg, &flow, 0.5, 3, 15, 3, 5, 1.2, 0)

		flowFields[i-1] = flow // Store the calculated flow

		prevImg.Close()   // Close the previous image
		prevImg = currImg // The current image becomes the next previous image
		// We defer closing the *last* currImg (which is now prevImg) until after the loop
	}
	defer prevImg.Close() // Clean up the final image

	// --- 2. Calculate grid velocities for each flow field ---
	// This will be a slice of maps
	gridVelocitiesHistory := make([]map[image.Point]GridVector, numFlows)
	for i, flow := range flowFields {
		gridVels, err := CalculateGridVelocities(flow, gridRes)
		if err != nil {
			// Clean up allocated flow mats before returning error
			for j := 0; j <= i; j++ {
				flowFields[j].Close()
			}
			return ExtrapolationData{}, fmt.Errorf("error calculating grid velocities for flow %d: %w", i, err)
		}
		gridVelocitiesHistory[i] = gridVels
		flow.Close() // We are done with this flow field
	}

	// --- 3. Fit polynomial to find velocity and acceleration ---
	extrapolation := ExtrapolationData{
		GridRes: gridRes,
		Data:    make(map[image.Point]GridVector),
	}

	// Define the time coordinates for the fit.
	// We want t=0 to be the *last* flow field.
	// E.g., for 4 frames (3 flows): times = [-10, -5, 0]
	times := make([]float64, numFlows)
	for i := 0; i < numFlows; i++ {
		times[i] = (float64(i) - float64(numFlows-1)) * timeStep
	}

	// Iterate over all grid points present in the *last* flow field
	// Assumes the grid is mostly stable
	lastGridVels := gridVelocitiesHistory[numFlows-1]
	if lastGridVels == nil {
		return ExtrapolationData{}, fmt.Errorf("no grid velocities found for the last frame")
	}

	for pt := range lastGridVels {
		vxValues := make([]float64, numFlows)
		vyValues := make([]float64, numFlows)

		// Gather the history of Vx and Vy for this specific grid point 'pt'
		for j := 0; j < numFlows; j++ {
			if vel, ok := gridVelocitiesHistory[j][pt]; ok {
				vxValues[j] = vel.Vx
				vyValues[j] = vel.Vy
			}
			// if not ok, it defaults to 0.0, which is acceptable
		}

		// Fit v(t) = a*t + b
		// b (intercept) is the velocity at t=0 (Vx/Vy)
		// a (slope) is the acceleration (Ax/Ay)
		v0x, accelX := FitPolynomial(times, vxValues)
		v0y, accelY := FitPolynomial(times, vyValues)

		extrapolation.Data[pt] = GridVector{
			Vx: v0x,
			Vy: v0y,
			Ax: accelX,
			Ay: accelY,
		}
	}

	return extrapolation, nil
}

// Example main function (replace with your actual image paths)
func main() {
	// This requires OpenCV to be installed on your system
	// And the gocv library: go get -u gocv.io/x/gocv

	// Create dummy images for testing (e.g., 1024x1024)
	// In a real scenario, you would have file paths
	imagePaths := []string{"frame0.png", "frame1.png", "frame2.png", "frame3.png"}
	log.Println("Warning: Using dummy images. Please create frame0.png, frame1.png, etc.")

	// Create dummy png files if they don't exist
	for _, path := range imagePaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			img := image.NewRGBA(image.Rect(0, 0, 1024, 1024))
			// Draw something simple
			for x := 200; x < 400; x++ {
				for y := 200; y < 400; y++ {
					img.Set(x, y, color.Gray{Y: 150})
				}
			}
			file, _ := os.Create(path)
			png.Encode(file, img)
			file.Close()
			log.Printf("Created dummy image: %s", path)
		}
	}

	gridResolution := 64
	timeStepMinutes := 5.0

	log.Println("Processing images...")
	extrapolationData, err := ProcessImages(imagePaths, gridResolution, timeStepMinutes)
	if err != nil {
		log.Fatalf("Failed to process images: %v", err)
	}

	log.Printf("Successfully generated extrapolation data for %d grid cells.", len(extrapolationData.Data))
	// Print data for a few cells
	count := 0
	for pt, data := range extrapolationData.Data {
		if count > 5 {
			break
		}
		log.Printf("Grid[%d, %d]: V=(%.2f, %.2f), A=(%.2f, %.2f)", pt.X, pt.Y, data.Vx, data.Vy, data.Ax, data.Ay)
		count++
	}
}
