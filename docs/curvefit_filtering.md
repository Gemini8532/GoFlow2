# Curve-Fit Track Filtering

The curve-fit filtering provides a more principled approach to track selection based on polynomial fit quality rather than ad-hoc smoothness checks.

## Usage

### Command Line (cast)

```bash
# Use curve-fit filtering with default parameters
go run cmd/cast/main.go -filterType curvefit -gridType average file1.png file2.png ...

# Customize curve-fit parameters
go run cmd/cast/main.go \
  -filterType curvefit \
  -minRSquared 0.90 \
  -maxRMSE 2.5 \
  -maxDeviation 6.0 \
  -maxAcceleration 1.5 \
  -gridType average \
  file1.png file2.png ...
```

### HTTP API

```json
{
  "filenames": ["file1.png", "file2.png"],
  "id": "123",
  "gridType": "average",
  "config": {
    "maxFeatures": 200,
    "filterType": "curvefit",
    "minTrackLength": 6,
    "minRSquared": 0.85,
    "maxRMSE": 3.0,
    "maxDeviation": 8.0,
    "maxAcceleration": 2.0
  }
}
```

## Parameters

- **minRSquared** (default: 0.85): Minimum R² coefficient of determination (0-1). Higher values require better polynomial fits.
- **maxRMSE** (default: 3.0): Maximum root mean square error in pixels. Lower values reject tracks with poor fits.
- **maxDeviation** (default: 8.0): Maximum distance any point can deviate from the fitted curve in pixels.
- **maxAcceleration** (default: 2.0): Maximum acceleration in pixels/frame². Ensures physically plausible motion.

## Quality Metrics

The curve-fit filter evaluates each track using:

1. **R² (Coefficient of Determination)**: How well the polynomial fits the track (1.0 = perfect)
2. **RMSE (Root Mean Square Error)**: Average distance from the fitted curve
3. **Max Deviation**: Largest distance any point deviates from the curve
4. **Acceleration**: Constant acceleration from the quadratic fit

## Comparison with Smoothness Filter

Based on testing with real rainfall data:

**Smoothness Filter (traditional)**:
- Keeps 56.4% of tracks
- Max RMSE: 14.15 pixels
- Max deviation: 25.37 pixels  
- Max acceleration: 5.08 px/frame²

**Curve-Fit Filter (default)**:
- Keeps 41.8% of tracks (more selective)
- Max RMSE: 3.00 pixels ✓
- Max deviation: 6.58 pixels ✓
- Max acceleration: 1.84 px/frame² ✓

The curve-fit filter produces higher-quality tracks with better fits and more physically plausible motion.

## Presets

- **Strict**: `minRSquared=0.95, maxRMSE=2.0, maxDeviation=5.0, maxAcceleration=1.0` (20% of tracks)
- **Default**: `minRSquared=0.85, maxRMSE=3.0, maxDeviation=8.0, maxAcceleration=2.0` (42% of tracks)
- **Relaxed**: `minRSquared=0.75, maxRMSE=5.0, maxDeviation=12.0, maxAcceleration=3.0` (73% of tracks)
