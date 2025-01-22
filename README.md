# webrtc-ros-bridge

![Tests](https://github.com/3DRX/webrtc-ros-bridge/actions/workflows/test.yml/badge.svg)

## Prerequisites

- ROS2 humble, merge installed
(tested on debian 12 with ROS2 humble build from source
with command `colcon build --merge-install`)
- libvpx-dev (deb package)
- `go mod tidy` to get all go deps
    - Note that it's expected to see errors of not finding package `github.com/3DRX/webrtc-ros-bridge/rclgo_gen`,
    since it's part of the codegen using `github.com/tiiuae/rclgo`, you can just ignore it.

## Build

Source your ros2 workspace setup.sh, and then
```
make
```

## Dev

For editor use, it's better to `source ./cgo-flags.env`
before opening editor for language server to work.

## Run

`wrb` cli can be configured to be either the sender or the receiver,
the config is load from a json file specified by `wrb <path_to.json>`.

> [!IMPORTANT]  
> Don't forget to remove comments in json.

### Sender

Firstly, if you're going to accept a image topic, you'd better to specify the img specification just like the following, otherwise we will use default img specification.

Secondly, you can configure the input topic name, the qos profile if necessary, otherwise we will use default qos profile like the following.

```json
{
    "mode": "sender",
    "addr": "localhost:8080",
    "topics": [
        {
            "name_in": "image_raw", // input image topic name
            "name_out": "image_out", // doesn't really matter
            "type": "sensor_msgs/msg/Image",
            "image_spec": {
                "width": 640,
                "height": 480,
                "frame_rate": 30
            },
            "qos": {
                "depth": 10,
                "history": 1,       // KeepLast
                "reliability": 2    // BestEffort
            }
        }
    ]
}
```

### Receiver

Like the sender.

```json
{
    "mode": "receiver",
    "addr": "localhost:8080",
    "topics": [
        {
            "name_in": "image_raw", // doesn't really matter
            "name_out": "image", // output image topic name
            "type": "sensor_msgs/msg/Image",
            "image_spec": {
                "width": 640,
                "height": 480,
                "frame_rate": 30
            },
            "qos": {
                "depth": 10,
                "history": 1,       // KeepLast
                "reliability": 2    // BestEffort
            }
        }
    ]
}
```

### Ros QosProfile

You can look up the official code for QosProfile. `https://github.com/tiiuae/rclgo/blob/main/pkg/rclgo/qos.go`

## Acknowledgment

- [pion](https://github.com/pion). It's awesome.
- [webrtc_ros](https://github.com/RobotWebTools/webrtc_ros).
Infact, the receiver of this project is compatable with webrtc_ros server node.
- [ros2](https://github.com/ros2)
- [tiiuae/rclgo](https://github.com/tiiuae/rclgo). The ROS client library for golang.
