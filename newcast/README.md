# Newcast Package

This package provides tools for tracking features in image sequences and extrapolating their future positions. It is designed for tasks like weather radar nowcasting, where the short-term movement of objects (like rain cells) needs to be predicted.

## Core Components

- **`Tracker`**: The main object for managing the feature tracking process. It uses the Lucas-Kanade optical flow algorithm to follow features from one frame to the next.
- **`Track`**: Represents the history of a single feature's movement across multiple image frames.
- **`Point`**: Represents a single detection of a feature at a specific time and location.
- **`Polynomial`**: A struct for representing a quadratic curve, used for motion estimation.

## Functionality

- **Feature Tracking**: The `Tracker` can identify "good features to track" in an initial image and then follow them through a sequence of subsequent images.
- **Motion Estimation**: The package can estimate the velocity and acceleration of each tracked feature. It does this by fitting a quadratic polynomial to the recent history of a track's positions. If the polynomial fit fails, it falls back to using finite differences.
- **Track Filtering**: The package provides several functions for filtering tracks based on criteria like smoothness and spatial density. This is useful for removing noise and keeping only the most significant and coherent motion tracks.
- **Visualization**: The package includes functions for visualizing the tracked features, including their historical paths, their current velocity vectors, and their extrapolated future paths.

## Demo Application

The `app` subdirectory contains a command-line application that demonstrates the use of the `newcast` package. It can be run from the `newcast/app` directory:

```bash
go run .
```

The application processes a sequence of rainfall radar images from the `rainfall_data` directory, tracks the features, filters them, and generates several visualizations of the results.

You can customize the behavior of the application using command-line flags. For example, to change the number of images processed and the filtering method, you can run:

```bash
go run . -numImages=8 -filterType=density
```

For a full list of available flags, run:

```bash
go run . -h
```
