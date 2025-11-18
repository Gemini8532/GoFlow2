# Goflow

This repository contains tools for optical flow analysis on sequences of images, particularly geared towards rainfall data.

## Project Structure

- `cmd/cast`: Command-line application to process image sequences into tracks, filter them, and visualize the results.
- `cmd/serve`: HTTP API server to process image sequences and serve vector flow data as PNG images.
- `newcast`: Core Go package containing the optical flow tracking, filtering, and grid generation logic.
- `flowvis`: A web-based visualization client (React/TypeScript) for the `cmd/serve` API.
- `rainfall_data/`: Sample rainfall image data.
- `test_data/`: Test image data.

## Building and Running

To build the Go applications:

```bash
go build ./...
```

### `cmd/cast` - Command-line Tracking and Visualization

This tool takes a series of PNG images, tracks features, filters them, and can either generate visualizations locally or submit them to a `goflow` server for remote processing.

#### Local Processing

This is the default mode. It generates visualization images directly.

**Usage:**

```bash
# Example: Process a sequence of rainfall images and generate visualizations
go run cmd/cast/main.go -maxFeatures 200 -smoothness 0.5 -minTrackLength 6 rainfall_data/*.png
```

The output will be `rainfall_tracks.png` and `rainfall_vectors.png` in the current directory.

#### Remote Processing

By providing the `-serverURL` flag, the tool will send the processing request to a running `goflow` server. Instead of creating local image files, it will output a series of `curl` commands that you can use to fetch the processed vector frames from the server.

An optional `-id` flag can be used to specify a custom ID for the remote processing request. If not provided, a unique ID based on the current timestamp will be generated.

**Usage:**

```bash
# Example: Send a sequence of rainfall images to a local server for processing
go run cmd/cast/main.go -serverURL http://localhost:8080 rainfall_data/*.png

# Example: Send a sequence of rainfall images with a custom ID
go run cmd/cast/main.go -serverURL http://localhost:8080 -id "my-custom-id" rainfall_data/*.png
```

The output will be a success message and a list of `curl` commands, for example:
```
Successfully submitted processing request with ID: cast-1668792631
To fetch the vector frames, use the following commands:
curl -o frame_0.png "http://localhost:8080/vector-frame?id=cast-1668792631&t=0"
curl -o frame_1.png "http://localhost:8080/vector-frame?id=cast-1668792631&t=1"
...
```

### `cmd/serve` - HTTP API Server

This server provides an API to process image sequences into vector flow grids and serve individual time-slices of these grids as PNG images.

#### Running the server:

```bash
go run cmd/serve/main.go
```

The server will start on `http://localhost:8080`.

#### API Endpoints:

1.  **`/process` (POST)**
    *   **Description**: Initiates the processing of an image sequence to generate a `FlowGrid`. The resulting `FlowGrid` is stored in memory, associated with a provided `id`.
    *   **Method**: `POST`
    *   **Content-Type**: `application/json`
    *   **Request Body**:
        ```json
        {
          "filenames": ["path/to/image1.png", "path/to/image2.png", ...],
          "id": "your-unique-request-id"
        }
        ```
        *   `filenames`: An array of paths to the image files to be processed.
        *   `id`: A unique identifier for this processing request. This `id` will be used to retrieve the processed vector frames.
    *   **Example `curl` command:**
        ```bash
        curl -X POST -H "Content-Type: application/json" \
             -d '{"filenames": ["rainfall_data/2025-10-03T14:40:00Z.png", "rainfall_data/2025-10-03T14:45:00Z.png"], "id": "my-test-flow"}' \
             http://localhost:8080/process
        ```
    *   **Response**: `200 OK` with `{"status": "success", "message": "FlowGrid generated and stored"}` on success, or an error message and appropriate status code on failure.

2.  **`/vector-frame` (GET)**
    *   **Description**: Retrieves a single vector flow frame (as a PNG image) for a given `id` and time index `t`. The image data is encoded using the `grid_encode` format, which can be interpreted by the `flowvis` frontend.
    *   **Method**: `GET`
    *   **Query Parameters**:
        *   `id`: The unique identifier of the processed `FlowGrid` (as provided in the `/process` request).
        *   `t`: The zero-based time index of the desired frame. This corresponds to the index in the `filenames` array provided during the `/process` step, minus one (since flow is calculated between frames).
    *   **Example `curl` command:**
        ```bash
        curl -o frame_0.png "http://localhost:8080/vector-frame?id=my-test-flow&t=0"
        ```
        This would download the first vector frame (flow between image 0 and image 1) for the `my-test-flow` ID.
    *   **Response**: `200 OK` with `Content-Type: image/png` and the binary PNG image data on success, or an error message and appropriate status code on failure.

### `flowvis` - Web Visualization Client

(Details about `flowvis` would go here, e.g., how to build and run it, and how it interacts with the `cmd/serve` API.)

## Go Module

This project uses Go Modules. The module name is `example/goflow`.