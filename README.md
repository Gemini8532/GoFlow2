Go Rainflow Optical Flow ModuleThis module uses the Lucas-Kanade algorithm (via gocv) to predict optical flowby tracking features across a series of rainfall images.⚠️ Important Dependency: OpenCVThis module requires gocv, which in turn requires the native OpenCV (C++) libraries to be installed on your system.Since you're on Ubuntu, you can install the necessary dependencies like this:# Install OpenCV and its dependencies
sudo apt-get update
sudo apt-get install -y libopencv-dev opencv-contrib

# Install Go (if not already present)
# (Your Go version may vary)
sudo apt-get install -y golang
Module Structurego.mod: Defines the module and its gocv dependency.main.go: An example executable to run the flow prediction.flow/lk.go: The core package containing the optical flow logic.How to RunMake sure the dependencies (Go, libopencv-dev) are installed.Get the gocv package:go get gocv.io/x/gocv
Place your sequential rain images (e.g., frame01.png, frame02.png, etc.) in the root directory. These must be 1024x1024 PNGs.Run the example, providing the output path first, followed by all input images in order:go run . output_flow_map.png frame01.png frame02.png frame03.png
This will create output_flow_map.png, a reduced-resolution image showing the motion vectors. Each vector represents the total displacement of a feature tracked from the first frame to the last frame.
