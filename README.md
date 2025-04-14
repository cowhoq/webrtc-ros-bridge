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

## Features

### Autoware Integration

This project now supports Autoware message types for remote monitoring and control of autonomous vehicles. The following message types are supported:

- `autoware_control_msgs/msg/Control`
- `autoware_planning_msgs/msg/Trajectory`
- `autoware_vehicle_msgs/msg/ControlModeReport`
- `autoware_vehicle_msgs/msg/VelocityReport`
- `autoware_vehicle_msgs/msg/SteeringReport`
- `autoware_vehicle_msgs/msg/GearReport`

[详细文档](./docs/autoware_integration.md)

### Adaptive Bandwidth Management

The WebRTC-ROS bridge implements an intelligent bandwidth management system that dynamically allocates bandwidth between video streams and ROS message data channels. This system optimizes both visual quality and control responsiveness based on real-time network conditions and traffic patterns.

Key benefits:

- Prioritizes critical control messages in bandwidth-constrained environments
- Dynamically adjusts video quality to utilize available bandwidth efficiently
- Provides smooth quality transitions through advanced filtering algorithms
- Ensures robust operation across various network conditions

[详细文档](./docs/adaptive_bandwidth_management.md)

### Generate Autoware Message Bindings

Before using Autoware message types, you need to generate the Go bindings:

```
./scripts/generate_autoware_interfaces.sh
```

### Example Autoware Configurations

For your convenience, we've provided example configurations for both sender and receiver:

- `example_autoware_sender.json` - Configuration for the sender node that forwards Autoware messages
- `example_autoware_receiver.json` - Configuration for the receiver node that receives Autoware messages

To use these configurations:

```bash
# On the Autoware vehicle or simulation machine
./wrb example_autoware_sender.json

# On the remote monitoring machine
./wrb example_autoware_receiver.json
```

Make sure to adjust the `addr` field in both configurations to match your network setup.

### Example Sender Configuration for Autoware

Here's an example configuration for setting up a sender that forwards Autoware vehicle status messages:

```json
{
    "mode": "sender",
    "addr": "192.168.1.100:8080",
    "topics": [
        {
            "name_in": "vehicle/status/velocity_status", 
            "name_out": "velocity_status",
            "type": "autoware_vehicle_msgs/msg/VelocityReport",
            "qos": {
                "depth": 10,
                "history": 1,
                "reliability": 1,
                "durability": 2
            }
        },
        {
            "name_in": "planning/trajectory", 
            "name_out": "trajectory",
            "type": "autoware_planning_msgs/msg/Trajectory",
            "qos": {
                "depth": 10,
                "history": 1,
                "reliability": 1,
                "durability": 2
            }
        }
    ]
}
```

### Example Receiver Configuration for Autoware

```json
{
    "mode": "receiver",
    "addr": "192.168.1.100:8080",
    "topics": [
        {
            "name_in": "velocity_status", 
            "name_out": "remote/vehicle/status/velocity_status",
            "type": "autoware_vehicle_msgs/msg/VelocityReport",
            "qos": {
                "depth": 10,
                "history": 1,
                "reliability": 1,
                "durability": 2
            }
        },
        {
            "name_in": "trajectory", 
            "name_out": "remote/planning/trajectory",
            "type": "autoware_planning_msgs/msg/Trajectory",
            "qos": {
                "depth": 10,
                "history": 1,
                "reliability": 1,
                "durability": 2
            }
        }
    ]
}
```

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
- [Autoware](https://github.com/autowarefoundation/autoware). The open-source autonomous driving stack.
