# Go Rainflow Optical Flow Module

This module uses the Lucas-Kanade algorithm (via `gocv`) to predict optical flow
by tracking features across a series of rainfall images.

## ⚠️ Important Dependency: OpenCV

This module requires `gocv`, which in turn requires the native OpenCV (C++)
libraries to be installed on your system.

If you're on a Debian-based Linux distribution (like Ubuntu), you can install the necessary dependencies like this:

```bash
# Install OpenCV and its dependencies
sudo apt-get update
sudo apt-get install -y libopencv-dev opencv-contrib
```

## How to Run

1.  **Install Dependencies:** Make sure Go and OpenCV are installed on your system.

2.  **Get `gocv`:**
    ```bash
    go get gocv.io/x/gocv
    ```

3.  **Prepare Images:** Place your sequential rainfall images (e.g., `frame01.png`, `frame02.png`, etc.) in a directory. These should be 1024x1024 PNG files.

4.  **Run the Executable:**
    ```bash
    go run ./cmd/main.go [flags] <frame1.png> <frame2.png> ...
    ```

    This will create an `output_flow_map.png` file, which is a reduced-resolution
    image showing the motion vectors. Each vector represents the total displacement
    of a feature tracked from the first frame to the last.

## Command-Line Flags

-   `-output <path>`: The path to save the output flow map image. (Default: `output_flow_map.png`)
-   `-resolution-factor <int>`: The factor by which to downscale the images before processing. (Default: `4`)

## Module Structure

-   `go.mod`: Defines the module and its `gocv` dependency.
-   `cmd/main.go`: The main executable for running the flow prediction.
-   `flow/lk.go`: The core package containing the optical flow logic.
-   `rainfall_data/`: A directory containing sample rainfall data.
