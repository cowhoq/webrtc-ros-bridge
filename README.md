# webrtc-ros-bridge

## Prerequisites

- ROS2 humble, merge installed
(tested on debian 12 with ROS2 humble build from source
with command `colcon build --merge-install`)
- libvpx-dev (deb package)
- `go mod tidy` to get all go deps

## Build

Source your ros2 workspace setup.sh, and then
```
make
```

## Dev

For editor use, it's better to `source ./cgo-flags.env`
before opening editor for language server to work.
