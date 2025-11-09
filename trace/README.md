# Trace Module for Angular Image Search

The trace module implements an efficient algorithm for searching an image in a specific direction using a triangular search region with a fixed angular field of view.

## Features

- **Angular Search**: Searches within a triangular region defined by origin, direction, angle, and distance
- **Maximum Projection**: Projects pixel values along a specified direction to create a 1D profile
- **Palette Image Support**: Works with paletted PNG images, preserving original palette indices as meaningful values

## Usage with Palette Images

The trace module is designed to work with paletted images like those in the `rainfall_data` directory. Each pixel in these images represents an index into a color palette where each index corresponds to specific quantitative values (e.g., rainfall intensity levels).

### Loading Paletted Images

```go
// Load a paletted image while preserving the original palette indices
imageData, err := LoadPalettedImageFromRaw("rainfall_data/2025-10-03T14:40:00Z.png")
if err != nil {
    log.Fatal(err)
}

// The imageData now contains the palette indices as float64 values
// Index 0 might represent no rainfall, index 5 might represent moderate rainfall, etc.
```

### Performing Angular Search

```go
origin := trace.Point{X: 10, Y: 10}
direction := trace.Point{X: 1, Y: 0} // Horizontal direction
fov := 0.2 // Field of view in radians
distance := 50.0

projection, triangle, err := trace.ProjectAngularSearch(imageData, origin, direction, fov, distance)
if err != nil {
    log.Fatal(err)
}

// The projection contains the maximum values found along the search direction
// These values correspond to the original palette indices, preserving the semantic meaning
```

## Key Benefits

1. **Preserves Original Data Meaning**: Unlike converting paletted images to grayscale or RGB, this approach maintains the quantitative meaning of palette indices.

2. **Efficient Processing**: The triangular rasterization algorithm efficiently processes only the relevant pixels in the search area.

3. **Maximum Projection**: Finds the highest values in each bin along the search direction, useful for detecting peaks or maximum intensities.

## Tests

Run the tests to verify functionality:

```bash
go test -v ./trace
```

The tests include specific scenarios for palette images to ensure the original index values are preserved during the trace operation.