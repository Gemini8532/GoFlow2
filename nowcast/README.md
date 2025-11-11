# Nowcast Package

The `nowcast` package provides functionalities for extrapolating motion from a sequence of images, primarily designed for short-term weather forecasting (nowcasting) based on radar imagery. It uses the Farneback optical flow algorithm from `gocv` to determine motion fields and then fits a simple physics model (velocity and acceleration) to predict future movement.

## Core Concepts

The process is broken down into several key steps:

1.  **Image Loading**: A sequence of images (e.g., radar scans) is loaded and converted to grayscale, as optical flow operates on intensity values.
2.  **Optical Flow Calculation**: For each consecutive pair of images, a dense optical flow field is computed. This field represents the apparent motion of pixels between the frames.
3.  **Grid Aggregation**: The pixel-level motion vectors are aggregated into a coarser grid (e.g., 64x64). This is done using a trimmed mean to filter out noise and outliers, resulting in a single, robust motion vector for each grid cell.
4.  **Temporal Fitting**: For each grid cell, the history of its motion vectors over the image sequence is analyzed. A first-order polynomial (linear regression) is fitted to this history to calculate the initial velocity (at the time of the last image) and the acceleration.

This provides a predictive model of how each grid cell is expected to move in the immediate future.

## Key Data Structures

### `GridVector`

Represents the extrapolated motion parameters for a single cell in the grid.

```go
type GridVector struct {
    Vx float64 // Velocity in the x-direction (pixels/frame) at t=0
    Vy float64 // Velocity in the y-direction (pixels/frame) at t=0
    Ax float64 // Acceleration in the x-direction (pixels/frame^2)
    Ay float64 // Acceleration in the y-direction (pixels/frame^2)
}
```

### `ExtrapolationData`

Holds the complete set of motion vectors for the entire grid.

```go
type ExtrapolationData struct {
    GridRes int                    // The resolution of the grid (e.g., 64)
    Data    map[image.Point]GridVector // Maps a grid coordinate to its motion vector
}
```

## Main Functions

### `ProcessImages`

This is the primary entry point for the package. It takes a sequence of image file paths and orchestrates the entire nowcasting process.

```go
func ProcessImages(imagePaths []string, gridRes int, timeStep float64) (ExtrapolationData, error)
```

-   `imagePaths`: A slice of strings containing the file paths to the images, ordered from oldest to newest.
-   `gridRes`: The desired resolution for the aggregation grid (e.g., 64 for a 64x64 grid).
-   `timeStep`: The time interval between consecutive frames (e.g., 5.0 for 5 minutes).

### `LoadGrayscaleImage`

A helper function to load an image from a file path and convert it into a grayscale `gocv.Mat` suitable for processing.

```go
func LoadGrayscaleImage(filePath string) (gocv.Mat, error)
```

## Usage Example

The typical workflow involves providing a time-ordered list of radar images to the `ProcessImages` function.

```go
import "github.com/user/project/nowcast"

func main() {
    imagePaths := []string{"frame0.png", "frame1.png", "frame2.png", "frame3.png"}
    gridResolution := 64
    timeStepMinutes := 5.0

    extrapolationData, err := nowcast.ProcessImages(imagePaths, gridResolution, timeStepMinutes)
    if err != nil {
        log.Fatalf("Failed to process images: %v", err)
    }

    // You can now use extrapolationData.Data to predict future frames.
    log.Printf("Successfully generated extrapolation data for %d grid cells.", len(extrapolationData.Data))
}
```
