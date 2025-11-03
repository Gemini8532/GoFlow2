## Agent Instructions

This project has a critical dependency on the native OpenCV (C++) libraries.

To ensure the Go build process can find the necessary headers and libraries, you must install the `libopencv-dev` package.

On Debian-based systems like Ubuntu, you can install it with:

```bash
sudo apt-get update && sudo apt-get install -y libopencv-dev
```

**Important:** Always run this command before attempting to build or run the project for the first time.
