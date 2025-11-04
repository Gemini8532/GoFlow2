# Go Rainflow Optical Flow Module

This module uses the Lucas-Kanade algorithm (via `gocv`) to predict optical flow
by tracking features across a series of rainfall images.

## Important Dependency: OpenCV

This module requires `gocv`, which in turn requires the native OpenCV (C++)
libraries to be installed on your system.

If you're on a Debian-based Linux distribution (like Ubuntu), you can install the necessary dependencies like this:

```bash
# Install OpenCV and its dependencies
sudo apt-get update
sudo apt-get install -y libopencv-dev
```

### `gocv` Compatibility

Note that the version of `gocv` used in this project (`v0.31.0`) is known to be compatible with OpenCV `4.6.0`, which is the version typically available through `apt` on recent Ubuntu distributions. If you have a different version of OpenCV installed, you may need to adjust the `gocv` version in `go.mod`.

This is only partially true - the version in ubuntu 22.04 is  4.5.4, this seems to be incompatible....
I built 4.6.0 from scratch and installed it and it didn't work with v0.31.0 - but did with v0.30.0...
Then I tried building the versoin of opencv in v0.31.0 from /home/jdp/go/pkg/mod/gocv.io/x/gocv@v0.31.0/
the make install failed because libdc1394-22-dev is not available - so I edited the Makefile to use libdc1394-dev and make install worked...
v0.31.0 now works

I got jules to test whether v0.30.0 works with the version of opencv in ubuntu 24.4 and it does - so that would have been an alternative

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
-   `-resolution-factor <int>`: The factor by which to downscale the final output image. (Default: `4`)

## Module Structure

-   `go.mod`: Defines the module and its `gocv` dependency.
-   `cmd/main.go`: The main executable for running the flow prediction.
-   `flow/`: The core package containing the optical flow logic.
  - `lk.go`: Sparse feature tracking.
  - `denseflow.go`: Dense flow map generation.
  - `visualize.go`: Visualization utility functions.
-   `rainfall_data/`: A directory containing sample rainfall data.
