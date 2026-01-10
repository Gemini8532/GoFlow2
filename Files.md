# Project Files Summary

## Root Directory

* [AGENTS.md](AGENTS.md) - Contains development guidelines, dependency instructions (OpenCV), and architecture notes.
* [README.md](README.md) - Project overview and general usage instructions.
* [go.mod](go.mod) - Defines the Go module `example/goflow` and its dependencies.
* [go.sum](go.sum) - Checksums for Go module dependencies.

## cmd

* [cmd/cast/main.go](cmd/cast/main.go) - CLI entry point. Processes rainfall images locally or sends requests to the server. Handles flags for filtering, smoothing, and output generation.
* [cmd/serve/main.go](cmd/serve/main.go) - HTTP API server entry point. Provides endpoints for processing (`/process`) and retrieving flow data (`/vector-frame`, `/average-flow-grid`) or visualizations.

## docs

* [docs/curvefit_filtering.md](docs/curvefit_filtering.md) - Documentation explaining the curve-fitting based track filtering methodology.
* [docs/fitted_points.md](docs/fitted_points.md) - Documentation on using polynomial-fitted points for smoother trajectories.

## exp

* [exp/smooth_diffusion_fix.go](exp/smooth_diffusion_fix.go) - Experimental implementation of a fast smooth filling algorithm using a multigrid solver to propagate values.

## flowvis

* [flowvis/package.json](flowvis/package.json) - Configuration for the frontend application (React, Vite, TypeScript).
* [flowvis/src/App.tsx](flowvis/src/App.tsx) - Main React component for the vector sequence viewer. Handles fetching, CBOR decoding, and rendering vector fields on a canvas.
* [flowvis/src/main.tsx](flowvis/src/main.tsx) - Entry point for the React frontend application.

## newcast

* [newcast/cbor_encode.go](newcast/cbor_encode.go) - Implements CBOR encoding and decoding for `Frame` and `AverageFlowGrid` data, using gzip compression.
* [newcast/curvefit.go](newcast/curvefit.go) - Implements `FitQuadratic` to fit polynomial curves to track points and `ReplacWithFittedPoints` to smooth trajectories.
* [newcast/curvefit_filter.go](newcast/curvefit_filter.go) - Logic for filtering tracks based on the quality of their polynomial fit (R², RMSE, deviation).
* [newcast/fast_smooth.go](newcast/fast_smooth.go) - Implements `SmoothFill`, a multigrid solver for filling missing data in flow grids by solving Laplace's equation.
* [newcast/filter.go](newcast/filter.go) - Contains standard track filtering logic: smoothness (angle change), spatial density, and maximum angle.
* [newcast/flow_grid.go](newcast/flow_grid.go) - Defines `FlowGrid` structure and `FlowProcessor` for accumulating sparse track data into a dense grid.
* [newcast/grid_encode.go](newcast/grid_encode.go) - Handles PNG encoding/decoding of flow frames (legacy/alternative to CBOR). Defines `Frame` struct and resizing logic.
* [newcast/newcast.go](newcast/newcast.go) - Core `Tracker` implementation using OpenCV's Optical Flow (Lucas-Kanade). Defines `Track` and `Tracker` types.
* [newcast/process.go](newcast/process.go) - High-level orchestration for processing a sequence of file paths into tracks (`ProcessFilesToTracks`).
* [newcast/track_average.go](newcast/track_average.go) - Helper functions for calculating averaged tracks and generating average flow grids.
* [newcast/track_direct.go](newcast/track_direct.go) - Implements `GenerateFlowGridFromTracks` to create flow grids directly from raw tracks without pre-averaging.
* [newcast/types.go](newcast/types.go) - Defines fundamental types like `Vector`.
* [newcast/visualize.go](newcast/visualize.go) - Visualization functions for drawing tracks, extrapolated paths, and velocity vectors using OpenCV.
* [newcast/*_test.go](newcast/) - Various unit tests for the package (e.g., `newcast_test.go`, `flow_grid_test.go`).

## Data Directories

* [rainfall_data/](rainfall_data/) - Directory containing sample rainfall image sequences.
* [rainfall_data_uk/](rainfall_data_uk/) - Directory containing UK rainfall data.
* [test_data/](test_data/) - Directory containing images used for testing.
