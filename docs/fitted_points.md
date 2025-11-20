# Using Fitted Points for Smoother Flow Fields

## Overview

The `useFittedPoints` option replaces raw tracked points with polynomial-fitted points, reducing tracking noise and producing smoother flow fields.

## Benefits

Based on testing with real rainfall data:

- **87% reduction in tracking jitter** (1.91px → 0.27px average)
- **42% smoother flow fields** (gradient magnitude reduced by 42%)
- **Maintains magnitude accuracy** (only 3.8% difference in average magnitude)
- **Follows underlying motion trend** better than noisy raw points

## Usage

### Command Line (cast)

```bash
# Enable fitted points
go run cmd/cast/main.go -useFittedPoints -gridType average file1.png file2.png ...

# Combine with curve-fit filtering for best results
go run cmd/cast/main.go \
  -filterType curvefit \
  -useFittedPoints \
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
    "filterType": "curvefit",
    "useFittedPoints": true,
    "minTrackLength": 6
  }
}
```

## How It Works

1. **Track points are fitted** to a quadratic polynomial (degree 2)
2. **Each point is replaced** with its position on the fitted curve
3. **Noise is eliminated** while preserving the overall trajectory
4. **Flow grid is generated** from the smoothed points

## When to Use

✅ **Recommended when:**
- You want the smoothest possible flow fields
- Tracking noise is visible in the output
- Using for nowcasting/forecasting applications
- Combined with curve-fit filtering

❌ **Not recommended when:**
- You need to preserve exact tracked positions
- Tracks are very short (<4 points)
- Raw data accuracy is critical

## Comparison

### Raw Points
- Actual observed tracking data
- Contains tracking jitter/noise
- May have sudden jumps
- Average jitter: 1.91 pixels

### Fitted Points  
- Polynomial-smoothed trajectory
- Eliminates tracking noise
- Smooth, consistent motion
- Average jitter: 0.27 pixels

## Technical Details

- Uses **quadratic polynomial fit** (a*t² + b*t + c)
- Fitted separately for X and Y coordinates
- Points that can't be fitted are skipped
- Preserves the same number of points per track
- Polynomial coefficients stored in track for reuse

## Best Practices

1. **Use with curve-fit filtering** for highest quality
2. **Combine with moderate blur** (sigma=1.0) if needed
3. **Ensure minimum track length ≥6** for reliable fits
4. **Test with your data** to verify improvement

## Performance

- Minimal overhead (polynomial already fitted during filtering)
- Same grid generation speed
- Slightly fewer tracks if some can't be fitted
