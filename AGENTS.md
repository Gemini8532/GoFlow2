# Agent Development Guidelines

## Dependencies

This project has a critical dependency on the native OpenCV (C++) libraries.

To ensure the Go build process can find the necessary headers and libraries, you must install the `libopencv-dev` package.

On Debian-based systems like Ubuntu, you can install it with:

```bash
sudo apt-get update && sudo apt-get install -y libopencv-dev
```

**Important:** Always run this command before attempting to build or run the project for the first time.

## Build Practices

### Building Go Applications

When building Go applications, **always build binaries in their respective subdirectories** to keep them gitignored:

```bash
# Option 1: cd into directory first
cd cmd/cast && go build
cd cmd/serve && go build

# Option 2: use -o flag to specify output location
go build -o cmd/cast/cast ./cmd/cast/...
go build -o cmd/serve/serve ./cmd/serve/...
```

**Never** build at the repository root, as this creates executables that may not be properly gitignored.

## Architecture

### Data Transmission

- **Use binary format, NOT PNG** for flow grid data
- PNG causes alpha premultiplication corruption in browsers
- Binary format: int32 width + int32 height + float32 vector pairs (vx, vy)

### Track Filtering

- **Recommended**: Curve-fit filtering (`filterType: "curvefit"`)
- Evaluates polynomial fit quality (R², RMSE, max deviation, acceleration)
- More principled than ad-hoc smoothness checks
- Default params: `MinRSquared=0.85, MaxRMSE=3.0, MaxDeviation=8.0, MaxAcceleration=2.0`

### Flow Grid Generation

- Always work at **256x256 resolution** (no resizing to avoid interpolation artifacts)
- **Use fitted points** (`useFittedPoints: true`) for 87% noise reduction
- Gaussian blur is optional (smooth fill already provides smoothing)

## Configuration Best Practices

```json
{
  "filterType": "curvefit",
  "useFittedPoints": true,
  "blurSigma": 0.0,
  "minRSquared": 0.85,
  "maxRMSE": 3.0,
  "maxDeviation": 8.0,
  "maxAcceleration": 2.0
}
```

## Common Pitfalls

1. ❌ Don't use PNG for data transmission (alpha premultiplication)
2. ❌ Don't resize grids (interpolation artifacts)
3. ❌ Don't build at repository root
4. ✅ Do use curve-fit filtering
5. ✅ Do use fitted points
6. ✅ Do test with real rainfall data

## Recent Changes (2025-11-20)

- Switched PNG → binary format (eliminated encoding artifacts)
- Added curve-fit filtering (polynomial quality metrics)
- Added fitted points option (87% jitter reduction)
- Removed multigrid (caused banding from bilinear interpolation)
