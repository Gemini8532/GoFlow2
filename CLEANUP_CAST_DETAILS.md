# cmd/cast/main.go Cleanup Details

## Overview
The `cast` CLI tool currently supports all four filter types and both grid types. It needs to be simplified to use only curvefit filtering and the direct grid generation method.

## Changes Required

### 1. Remove Command-Line Flags (Lines 20-32)

**Remove these flags:**
```go
// Line 21
smoothness := flag.Float64("smoothness", 0.5, "...")

// Line 23
filterType := flag.String("filterType", "smoothness", "...")

// Line 24
maxAngle := flag.Float64("maxAngle", 0.8, "...")

// Line 25
gridCellSize := flag.Int("gridCellSize", 64, "...")

// Line 26
minTracksPerCell := flag.Int("minTracksPerCell", 2, "...")

// Line 27
maxTracksPerCell := flag.Int("maxTracksPerCell", 5, "...")

// Line 32
gridType := flag.String("gridType", "flow", "...")
```

**Keep these flags:**
```go
// Lines 20, 28-34
maxFeatures := flag.Int("maxFeatures", 200, "...")
minTrackLength := flag.Int("minTrackLength", 6, "...")
extrapolate := flag.Int("extrapolate", 0, "...")
serverURL := flag.String("serverURL", "", "...")
requestID := flag.String("id", "", "...")
blurSigma := flag.Float64("blurSigma", 0.0, "...")
useFittedPoints := flag.Bool("useFittedPoints", false, "...")

// Lines 37-40 (curvefit parameters)
minRSquared := flag.Float64("minRSquared", 0.90, "...") // Update default to 0.90
maxRMSE := flag.Float64("maxRMSE", 3.0, "...")
maxDeviation := flag.Float64("maxDeviation", 7.0, "...") // Update default to 7.0 
maxAcceleration := flag.Float64("maxAcceleration", 1.5, "...") // Update default to 1.5
```

### 2. Update ProcessConfig Construction (Lines 51-66)

**Before:**
```go
config := newcast.ProcessConfig{
    MaxFeatures:      *maxFeatures,
    Smoothness:       *smoothness,        // REMOVE
    FilterType:       *filterType,         // REMOVE
    MaxAngle:         *maxAngle,          // REMOVE
    GridCellSize:     *gridCellSize,      // REMOVE
    MinTracksPerCell: *minTracksPerCell,  // REMOVE
    MaxTracksPerCell: *maxTracksPerCell,  // REMOVE
    MinTrackLength:   *minTrackLength,
    BlurSigma:        *blurSigma,
    UseFittedPoints:  *useFittedPoints,
    MinRSquared:      *minRSquared,
    MaxRMSE:          *maxRMSE,
    MaxDeviation:     *maxDeviation,
    MaxAcceleration:  *maxAcceleration,
}
```

**After:**
```go
config := newcast.ProcessConfig{
    MaxFeatures:      *maxFeatures,
    MinTrackLength:   *minTrackLength,
    BlurSigma:        *blurSigma,
    UseFittedPoints:  *useFittedPoints,
    MinRSquared:      *minRSquared,
    MaxRMSE:          *maxRMSE,
    MaxDeviation:     *maxDeviation,
    MaxAcceleration:  *maxAcceleration,
}
```

### 3. Update processRemotely() Function

**Change function signature (Line 129):**
```go
// Before:
func processRemotely(serverURL string, requestID string, gridType string, fileArgs []string, config newcast.ProcessConfig)

// After:
func processRemotely(serverURL string, requestID string, fileArgs []string, config newcast.ProcessConfig)
```

**Update function call (Line 69):**
```go
// Before:
processRemotely(*serverURL, *requestID, *gridType, fileArgs, config)

// After:
processRemotely(*serverURL, *requestID, fileArgs, config)
```

**Update request body (Lines 147-152):**
```go
// Before:
reqBody := map[string]interface{}{
    "filenames": absFileArgs,
    "id":        requestID,
    "config":    config,
    "gridType":  gridType,  // REMOVE this line
}

// After:
reqBody := map[string]interface{}{
    "filenames": absFileArgs,
    "id":        requestID,
    "config":    config,
}
```

**Simplify output message (Lines 177-186):**
```go
// Before:
switch gridType {
case "average":
    fmt.Println("\nTo fetch the average flow grid, use the following command:")
    fmt.Printf("curl -o average_flow_grid.png \"%s/average-flow-grid?id=%s\"\n", serverURL, requestID)
default: // "flow"
    fmt.Println("\nTo fetch the vector frames, use the following commands:")
    for i := 0; i < len(fileArgs)-1; i++ {
        fmt.Printf("curl -o frame_%d.png \"%s/vector-frame?id=%s&t=%d\"\n", i, serverURL, requestID, i)
    }
}

// After:
fmt.Println("\nTo fetch the flow grid, use the following command:")
fmt.Printf("curl -o flow_grid.bin \"%s/average-flow-grid?id=%s\"\n", serverURL, requestID)
```

### 4. Update processLocally() Function (Lines 75-127)

**Remove filter type logging (Lines 80-89):**
```go
// REMOVE:
fmt.Printf("Filter type: %s\n", config.FilterType)
switch config.FilterType {
case "smoothness":
    fmt.Printf("Smoothness filter params: smoothness=%.2f\n", config.Smoothness)
case "density":
    fmt.Printf("Density filter params: gridCellSize=%d, minTracksPerCell=%d, maxTracksPerCell=%d\n",
        config.GridCellSize, config.MinTracksPerCell, config.MaxTracksPerCell)
case "max_angle":
    fmt.Printf("Max Angle filter params: maxAngle=%.2f\n", config.MaxAngle)
}
```

**Add curvefit logging:**
```go
// ADD:
fmt.Println("Using curvefit filtering:")
fmt.Printf("  MinRSquared: %.2f\n", config.MinRSquared)
fmt.Printf("  MaxRMSE: %.2f\n", config.MaxRMSE)
fmt.Printf("  MaxDeviation: %.2f\n", config.MaxDeviation)
fmt.Printf("  MaxAcceleration: %.2f\n", config.MaxAcceleration)
```

## Example Usage

### Before Cleanup
```bash
# Different filter types
./cast -filterType smoothness -smoothness 0.5 file1.png file2.png
./cast -filterType density -gridCellSize 64 file1.png file2.png
./cast -filterType max_angle -maxAngle 0.8 file1.png file2.png
./cast -filterType curvefit -minRSquared 0.85 file1.png file2.png

# Different grid types
./cast -gridType flow -serverURL http://localhost:9093 file1.png file2.png
./cast -gridType average -serverURL http://localhost:9093 file1.png file2.png
```

### After Cleanup
```bash
# Default curvefit parameters
./cast file1.png file2.png

# Custom curvefit parameters
./cast -minRSquared 0.90 -maxRMSE 3.0 -maxDeviation 7.0 file1.png file2.png

# Remote processing (always uses direct grid generation)
./cast -serverURL http://localhost:9093 file1.png file2.png

# With optional smoothing
./cast -blurSigma 1.0 -useFittedPoints file1.png file2.png
```

## Help Text Changes

### Before
```
Usage: ./cast [flags] <file1.png> <file2.png> ...

Flags:
  -filterType string
        Type of filter to use: 'smoothness', 'density', 'max_angle', or 'curvefit'. (default "smoothness")
  -smoothness float
        Smoothness threshold (max average angle change in radians). (default 0.5)
  -maxAngle float
        Maximum allowed angle change (in radians) for the max_angle filter. (default 0.8)
  -gridCellSize int
        Grid cell size for density filter. (default 64)
  -minTracksPerCell int
        Minimum number of tracks in a cell to be considered dense. (default 2)
  -maxTracksPerCell int
        Maximum number of smoothest tracks to keep from a dense cell. (default 5)
  -gridType string
        Type of grid to generate: 'flow' or 'average'. (default "flow")
  ...
```

### After
```
Usage: ./cast [flags] <file1.png> <file2.png> ...

Flags:
  -minRSquared float
        Minimum R² for curve-fit filter (0-1, higher = stricter). (default 0.9)
  -maxRMSE float
        Maximum RMSE in pixels for curve-fit filter. (default 3.0)
  -maxDeviation float
        Maximum deviation from fitted curve in pixels. (default 7.0)
  -maxAcceleration float
        Maximum acceleration in pixels/frame² for curve-fit filter. (default 1.5)
  -blurSigma float
        Gaussian blur sigma for flow grid smoothing (0 = no blur, 1.0 = light, 2.0 = medium). (default 0.0)
  -useFittedPoints
        Use polynomial-fitted points instead of raw tracked points (reduces noise).
  ...
```

## Lines to Change Summary

| Lines | Action | Description |
|-------|--------|-------------|
| 21, 23-27, 32 | DELETE | Remove old filter and grid type flags |
| 37-40 | UPDATE | Update default values for curvefit flags |
| 51-66 | UPDATE | Remove fields from ProcessConfig construction |
| 69 | UPDATE | Remove gridType parameter from function call |
| 80-89 | DELETE | Remove filter type logging |
| 80-89 | ADD | Add curvefit parameter logging |
| 129 | UPDATE | Remove gridType parameter from function signature |
| 151 | DELETE | Remove gridType from request body |
| 177-186 | SIMPLIFY | Remove switch statement, always show average-flow-grid command |

## Testing After Changes

```bash
# Build
go build -o cast ./cmd/cast

# Test local processing
./cast ../rainfall_data_uk/2025-11-14/*.png

# Test with custom parameters
./cast -minRSquared 0.95 -maxRMSE 2.0 ../rainfall_data_uk/2025-11-14/*.png

# Test remote processing (requires server running)
./cast -serverURL http://localhost:9093 ../rainfall_data_uk/2025-11-14/*.png

# Verify help text
./cast -help
```
