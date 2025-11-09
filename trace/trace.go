// Package trace implements an efficient algorithm for searching an image in a specific direction
// using a triangular search region with a fixed angular field of view.
// The algorithm rasterizes a triangle representing the search area and projects
// pixel values along a specified direction to create a 1D maximum projection profile.
//
// The main entry point is ProjectAngularSearch() which:
// 1. Constructs a triangular search region based on origin, direction, angle, and distance
// 2. Rasterizes the triangle to determine which pixels are inside it
// 3. Projects those pixel values along the specified direction using maximum projection
//
// The package works well with paletted images (such as those in rainfall_data) by treating
// palette indices as numeric values. For example, in paletted rainfall images, each pixel
// value represents an index into a color palette where each index corresponds to a specific
// rainfall intensity level. The trace algorithm preserves these original indices rather than
// converting them to RGB values, maintaining the semantic meaning of the original data.
//
// For paletted images, a helper function LoadPalettedImageFromRaw() is provided in
// palette_image.go to properly extract the palette indices as float64 values.

package trace

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// Point defines a 2D coordinate or vector
type Point struct {
	X, Y float64
}

// Triangle defines three vertices
type Triangle struct {
	V1, V2, V3 Point
}

// ProjectAngularSearch performs an angular search of an image using a triangular region.
// It creates a triangle with the apex at 'origin', pointing in 'direction' with 
// a field of view specified by 'fieldOfViewAngleRadians' and extending to 'distance'.
// It then rasterizes this triangle and projects the pixel values inside it 
// along the specified direction using maximum projection to create a 1D profile.
//
// Parameters:
//   - image: 2D array of pixel values to be searched
//   - origin: starting point of the search triangle
//   - direction: unit vector indicating the primary search direction
//   - fieldOfViewAngleRadians: total angular width of the search cone in radians (0 < fov < π)
//   - distance: length of the search triangle from origin
//
// Returns:
//   - []float64: 1D maximum projection profile along the search direction
//   - Triangle: the triangle that was searched
//   - error: any error encountered during the search
func ProjectAngularSearch(
	image [][]float64,
	origin Point,
	direction Point,
	fieldOfViewAngleRadians float64,
	distance float64,
) ([]float64, Triangle, error) {

	// --- 1. Validate Inputs ---
	if distance <= 0 {
		return nil, Triangle{}, errors.New("distance must be positive")
	}
	if fieldOfViewAngleRadians <= 0 || fieldOfViewAngleRadians >= math.Pi {
		return nil, Triangle{}, errors.New("fieldOfViewAngleRadians must be between 0 and Pi (180 degrees)")
	}

	// Normalize the direction vector
	dirUnitVec, mag := normalize(direction)
	if mag == 0 {
		return nil, Triangle{}, errors.New("direction vector cannot be zero")
	}

	// --- 2. Calculate Triangle Vertices ---
	v1 := origin
	baseCenter := Point{
		X: origin.X + dirUnitVec.X*distance,
		Y: origin.Y + dirUnitVec.Y*distance,
	}
	halfAngle := fieldOfViewAngleRadians / 2.0
	halfWidth := distance * math.Tan(halfAngle)
	perpVec := Point{X: -dirUnitVec.Y, Y: dirUnitVec.X}
	v2 := Point{
		X: baseCenter.X + perpVec.X*halfWidth,
		Y: baseCenter.Y + perpVec.Y*halfWidth,
	}
	v3 := Point{
		X: baseCenter.X - perpVec.X*halfWidth,
		Y: baseCenter.Y - perpVec.Y*halfWidth,
	}
	tri := Triangle{V1: v1, V2: v2, V3: v3}

	// --- 3. Call Core Projection Algorithm ---
	projection := ProjectTriangleMax(image, tri, dirUnitVec)

	return projection, tri, nil
}

// ProjectTriangleMax sets up the 1D projection array and calls the
// high-performance scan-line rasterizer.
func ProjectTriangleMax(image [][]float64, tri Triangle, dirUnitVec Point) []float64 {
	imgHeight := len(image)
	if imgHeight == 0 {
		return nil
	}
	imgWidth := len(image[0])
	if imgWidth == 0 {
		return nil
	}

	// --- 1. Create 1D Result Array ---
	p1 := dot(tri.V1, dirUnitVec)
	p2 := dot(tri.V2, dirUnitVec)
	p3 := dot(tri.V3, dirUnitVec)

	uMin := math.Min(p1, math.Min(p2, p3))
	uMax := math.Max(p1, math.Max(p2, p3))

	arraySize := int(math.Ceil(uMax)) - int(math.Floor(uMin)) + 1
	if arraySize <= 0 {
		return nil
	}

	maxValues := make([]float64, arraySize)
	for i := range maxValues {
		maxValues[i] = math.Inf(-1) // Initialize with negative infinity
	}

	// --- 2. Run Scan-line Rasterizer ---
	// This function does all the work and modifies maxValues in place
	rasterizeTriangleAndProject(image, tri, dirUnitVec, uMin, maxValues)

	return maxValues
}

// rasterizeTriangleAndProject implements the scan-line algorithm.
// It sorts the vertices by Y and splits the triangle into a
// flat-top and flat-bottom part, then fills them.
func rasterizeTriangleAndProject(
	image [][]float64,
	tri Triangle,
	dirUnitVec Point,
	uMin float64,
	maxValues []float64,
) {
	imgHeight := len(image)
	imgWidth := len(image[0])
	uMinFloored := math.Floor(uMin)

	// Put vertices into a slice and sort them by Y-coordinate (v[0] is top)
	vertices := []Point{tri.V1, tri.V2, tri.V3}
	sort.Slice(vertices, func(i, j int) bool {
		return vertices[i].Y < vertices[j].Y
	})
	v1, v2, v3 := vertices[0], vertices[1], vertices[2]

	// Handle degenerate triangle (horizontal line)
	if v1.Y == v3.Y {
		return // Or handle as a single line, but for 2D it has no area
	}

	// Create a "processor" function to pass into the fillers.
	// This avoids duplicating the projection logic.
	processPixel := func(image [][]float64, x, y int) {
		pixelValue := image[y][x]
		u := (float64(x)*dirUnitVec.X + float64(y)*dirUnitVec.Y)
		i := int(math.Floor(u) - uMinFloored)
		if i >= 0 && i < len(maxValues) {
			maxValues[i] = math.Max(maxValues[i], pixelValue)
		}
	}

	// --- Split the triangle into flat-bottom and flat-top ---

	// Case 1: Flat-bottom triangle (v2.Y == v3.Y)
	if v2.Y == v3.Y {
		fillFlatBottomTriangle(image, v1, v2, v3, imgWidth, imgHeight, processPixel)
		return
	}

	// Case 2: Flat-top triangle (v1.Y == v2.Y)
	if v1.Y == v2.Y {
		fillFlatTopTriangle(image, v1, v2, v3, imgWidth, imgHeight, processPixel)
		return
	}

	// Case 3: General triangle. Need to split it.
	// Find the split point V4 on the long edge (V1-V3) at V2's Y-level
	t := (v2.Y - v1.Y) / (v3.Y - v1.Y)
	v4 := Point{
		X: v1.X + t*(v3.X-v1.X),
		Y: v2.Y,
	}

	// Split into two triangles and fill them
	if v2.X < v4.X {
		// V2 is left, V4 is right
		fillFlatBottomTriangle(image, v1, v2, v4, imgWidth, imgHeight, processPixel)
		fillFlatTopTriangle(image, v2, v4, v3, imgWidth, imgHeight, processPixel)
	} else {
		// V4 is left, V2 is right
		fillFlatBottomTriangle(image, v1, v4, v2, imgWidth, imgHeight, processPixel)
		fillFlatTopTriangle(image, v4, v2, v3, imgWidth, imgHeight, processPixel)
	}
}

// fillFlatBottomTriangle fills a triangle where vBotLeft and vBotRight are at the same Y
func fillFlatBottomTriangle(
	image [][]float64,
	vTop, vBotLeft, vBotRight Point,
	imgWidth, imgHeight int,
	processPixel func(image [][]float64, x, y int),
) {
	dy := vBotLeft.Y - vTop.Y
	if dy == 0 {
		return // Avoid divide-by-zero, zero-height triangle
	}

	// Calculate inverse slopes (dx/dy)
	slope1 := (vBotLeft.X - vTop.X) / dy
	slope2 := (vBotRight.X - vTop.X) / dy

	// Get Y scan range (pixel centers)
	yStart := int(math.Ceil(vTop.Y))
	yEnd := int(math.Floor(vBotLeft.Y))

	// Clamp Y to image bounds
	yStart = max(0, yStart)
	yEnd = min(imgHeight-1, yEnd)

	for y := yStart; y <= yEnd; y++ {
		// Get x start/end for this scan-line using absolute interpolation
		// (more stable than incremental curX += slope)
		x1 := vTop.X + float64(y)*slope1 - vTop.Y*slope1
		x2 := vTop.X + float64(y)*slope2 - vTop.Y*slope2

		xStart := int(math.Ceil(minF64(x1, x2)))
		xEnd := int(math.Floor(maxF64(x1, x2)))

		// Clamp X to image bounds
		xStart = max(0, xStart)
		xEnd = min(imgWidth-1, xEnd)

		// Fill the span
		for x := xStart; x <= xEnd; x++ {
			processPixel(image, x, y)
		}
	}
}

// fillFlatTopTriangle fills a triangle where vTopLeft and vTopRight are at the same Y
func fillFlatTopTriangle(
	image [][]float64,
	vTopLeft, vTopRight, vBot Point,
	imgWidth, imgHeight int,
	processPixel func(image [][]float64, x, y int),
) {
	dy := vBot.Y - vTopLeft.Y
	if dy == 0 {
		return // Avoid divide-by-zero, zero-height triangle
	}

	// Calculate inverse slopes (dx/dy)
	slope1 := (vBot.X - vTopLeft.X) / dy
	slope2 := (vBot.X - vTopRight.X) / dy

	// Get Y scan range (pixel centers)
	yStart := int(math.Ceil(vTopLeft.Y))
	yEnd := int(math.Floor(vBot.Y))

	// Clamp Y to image bounds
	yStart = max(0, yStart)
	yEnd = min(imgHeight-1, yEnd)

	for y := yStart; y <= yEnd; y++ {
		// Get x start/end for this scan-line
		x1 := vTopLeft.X + float64(y)*slope1 - vTopLeft.Y*slope1
		x2 := vTopRight.X + float64(y)*slope2 - vTopRight.Y*slope2

		xStart := int(math.Ceil(minF64(x1, x2)))
		xEnd := int(math.Floor(maxF64(x1, x2)))

		// Clamp X to image bounds
		xStart = max(0, xStart)
		xEnd = min(imgWidth-1, xEnd)

		// Fill the span
		for x := xStart; x <= xEnd; x++ {
			processPixel(image, x, y)
		}
	}
}

// --- Helper Functions ---

// normalize returns a unit vector and its original magnitude
func normalize(v Point) (Point, float64) {
	mag := math.Sqrt(v.X*v.X + v.Y*v.Y)
	if mag == 0 {
		return Point{X: 0, Y: 0}, 0
	}
	return Point{X: v.X / mag, Y: v.Y / mag}, mag
}

// dot calculates the dot product of two points (as vectors)
func dot(v1, v2 Point) float64 {
	return v1.X*v2.X + v1.Y*v2.Y
}

// Standard library 'max' for int
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Standard library 'min' for int
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// minF64 for float64
func minF64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// maxF64 for float64
func maxF64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func TestRun() {
	// 1. Create a simple mock image (e.g., 20x20)
	image := make([][]float64, 20)
	for i := range image {
		image[i] = make([]float64, 20)
	}

	// Create a diagonal line of 100s
	// and another spot of 50s
	for i := range 20 {
		image[i][i] = 100.0 // diagonal
		if i < 19 {
			image[i][i+1] = 100.0
		}
	}
	image[5][15] = 50.0
	image[6][15] = 50.0
	image[7][15] = 50.0

	// 2. Define search parameters
	origin := Point{X: 1, Y: 1}
	direction := Point{X: 1, Y: 1} // Search along the 45-degree diagonal
	fovDegrees := 30.0
	fovRadians := fovDegrees * math.Pi / 180.0
	searchDistance := 25.0 // Long enough to cross the image

	// 3. Run the new angular search
	// This will generate a long, narrow triangle at ~45 degrees
	// This is the "bad case" for the old algorithm, but fast for the new one.
	projection, tri, err := ProjectAngularSearch(image, origin, direction, fovRadians, searchDistance)

	// 4. Print the result
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Search Origin: (%.2f, %.2f)\n", origin.X, origin.Y)
		fmt.Printf("Search Direction: (%.2f, %.2f)\n", direction.X, direction.Y)
		fmt.Println("--- Calculated Triangle Vertices ---")
		fmt.Printf("V1 (Apex): (%.2f, %.2f)\n", tri.V1.X, tri.V1.Y)
		fmt.Printf("V2 (Base): (%.2f, %.2f)\n", tri.V2.X, tri.V2.Y)
		fmt.Printf("V3 (Base): (%.2f, %.2f)\n", tri.V3.X, tri.V3.Y)

		fmt.Println("\n--- 1D Maximum Projection Profile ---")
		fmt.Println("Bin | Max Value")
		fmt.Println("-----------------")
		for i, val := range projection {
			// Print non-empty bins for clarity
			if val != math.Inf(-1) {
				fmt.Printf("%3d | %.2f\n", i, val)
			}
		}
	}
}
