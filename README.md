# Goflow

This repository contains tools for optical flow analysis on sequences of images, particularly geared towards rainfall data. It generates high-quality, smoothed vector fields by tracking features across image sequences and fitting polynomial curves to their trajectories.

## Project Structure

- `cmd/cast`: Command-line application to process image sequences into tracks and generate average flow grids.
- `cmd/serve`: HTTP API server to process image sequences and serve vector flow data as gzipped CBOR.
- `newcast`: Core Go package containing the optical flow tracking, curve fitting, and grid generation logic.
- `flowvis`: A web-based visualization client (React/TypeScript) for the `cmd/serve` API.
- `rainfall_data/`: Sample rainfall image data.
- `test_data/`: Test image data.

## Building and Running

To build the Go applications:

```bash
go build ./...
```

### `cmd/cast` - Command-line Tool

This tool takes a series of PNG images, tracks features, applies curve fitting to filter noise, and generates an Average Flow Grid.

#### Local Processing

This is the default mode. It generates a gzipped CBOR file containing the average flow vectors.

**Usage:**

```bash
# Example: Process a sequence of rainfall images and generate an average flow grid
go run cmd/cast/main.go -outputGrid flow_grid.cbor.gz rainfall_data/*.png

# With customization
go run cmd/cast/main.go \
  -outputGrid flow_grid.cbor.gz \
  -maxFeatures 500 \
  -minTrackLength 10 \
  -blurSigma 1.0 \
  rainfall_data/*.png
```

**Key Flags:**
- `-outputGrid`: Output file path for the gzipped CBOR data (required).
- `-maxFeatures`: Maximum number of features to track (default: 200).
- `-minTrackLength`: Minimum number of points a track must have to be considered (default: 6).
- `-blurSigma`: Gaussian blur sigma for final grid smoothing (default: 0.0, meaning no blur).

**Curve Fitting Parameters:**
The tool automatically applies curve fitting to smooth tracks. You can tune these if needed:
- `-minRSquared`: Minimum R² for curve fit (default: 0.85).
- `-maxRMSE`: Maximum RMSE in pixels (default: 3.0).
- `-maxDeviation`: Maximum deviation from fitted curve (default: 8.0).
- `-maxAcceleration`: Maximum acceleration in pixels/frame² (default: 2.0).

#### Remote Processing

By providing the `-serverURL` flag, the tool will submit the processing request to a running `goflow` server.

**Usage:**

```bash
# Submit processing to a local server
go run cmd/cast/main.go -serverURL http://localhost:9093 rainfall_data/*.png
```

The output will provide a command to download the result, e.g.:
```
Successfully submitted processing request with ID: cast-1668792631
To fetch the average flow grid, use the following command:
curl -o average_flow_grid.png "http://localhost:9093/average-flow-grid?id=cast-1668792631"
```

### `cmd/serve` - HTTP API Server

This server provides an API to process image sequences and serve the resulting flow grids.

#### Running the server:

```bash
go run cmd/serve/main.go
```

The server will start on `http://localhost:9093`.

#### API Endpoints:

1.  **`/process` (POST)**
    *   **Description**: Initiates the processing of an image sequence. Tracks are generated, filtered via curve fitting, and an `AverageFlowGrid` is computed.
    *   **Method**: `POST`
    *   **Content-Type**: `application/json`
    *   **Request Body**:
        ```json
        {
          "filenames": ["/abs/path/to/image1.png", "/abs/path/to/image2.png", ...],
          "id": "unique-request-id",
          "config": {
            "MaxFeatures": 500,
            "MinTrackLength": 6,
            "BlurSigma": 1.0
          }
        }
        ```
    *   **Response**: `200 OK` with status message.

2.  **`/average-flow-grid` (GET)**
    *   **Description**: Retrieves the generated Average Flow Grid as gzipped CBOR data.
    *   **Method**: `GET`
    *   **Query Parameters**:
        *   `id`: The unique identifier provided in `/process`.
        *   `height` (optional): Request the grid be resized to a specific height (width is scaled to maintain aspect ratio).
    *   **Response**: `200 OK` with `Content-Type: application/octet-stream` and `Content-Encoding: gzip`.

3.  **`/tracks-visualization` (GET)**
    *   **Description**: Returns a PNG visualization of the tracked features.
    *   **Method**: `GET`
    *   **Query Parameters**: `id`
    *   **Response**: `200 OK` (PNG image).

## Go Module

This project uses Go Modules. The module name is `example/goflow`.
