# Autoware与AWSIM跨网络通信实施计划

## 1. 概述

本文档概述了如何扩展WebRTC-ROS桥接系统，以支持Autoware和AWSIM之间的全部话题通信。通过此扩展，可以实现两个系统之间的完整数据传输，无论它们是否在同一网络中。

## 2. 关键话题

Autoware和AWSIM之间需要传输的关键话题包括（但不限于）：

### 传感器数据
- `/sensing/lidar/top/pointcloud` (sensor_msgs/msg/PointCloud2)
- `/sensing/camera/*/image_raw` (sensor_msgs/msg/Image)
- `/sensing/gnss/pose` (geometry_msgs/msg/PoseStamped)
- `/sensing/imu/imu_data` (sensor_msgs/msg/Imu)

### 控制和规划
- `/control/command/control_cmd` (autoware_auto_control_msgs/msg/AckermannControlCommand)
- `/planning/scenario_planning/trajectory` (autoware_auto_planning_msgs/msg/Trajectory)

### 车辆状态
- `/vehicle/status/control_mode` (autoware_auto_vehicle_msgs/msg/ControlModeReport)
- `/vehicle/status/velocity_status` (autoware_auto_vehicle_msgs/msg/VelocityReport)
- `/vehicle/status/steering_status` (autoware_auto_vehicle_msgs/msg/SteeringReport)

## 3. 技术挑战

扩展系统面临以下技术挑战：

1. **多种消息类型支持**：
   - 需要增加对Autoware自定义消息类型的支持
   - 需要为每种消息类型实现序列化/反序列化逻辑

2. **大数据量传输**：
   - 点云数据(PointCloud2)通常很大，需要优化传输效率
   - 考虑压缩或降采样策略

3. **消息优先级**：
   - 控制命令需要低延迟，而某些传感器数据可接受较高延迟
   - 需要实现消息优先级机制

4. **网络带宽限制**：
   - WebRTC有带宽限制，需要实现智能调度算法
   - 可能需要动态调整传输质量

## 4. 实施步骤

### 4.1 扩展常量定义

在`consts/consts.go`中添加新的消息类型常量：

```go
package consts

const (
    MSG_IMAGE       = "sensor_msgs/msg/Image"
    MSG_LASER_SCAN  = "sensor_msgs/msg/LaserScan"
    MSG_POINTCLOUD2 = "sensor_msgs/msg/PointCloud2"
    
    // Autoware specific messages
    MSG_CONTROL_MODE_REPORT = "autoware_auto_vehicle_msgs/msg/ControlModeReport"
    MSG_VELOCITY_REPORT     = "autoware_auto_vehicle_msgs/msg/VelocityReport"
    MSG_TRAJECTORY          = "autoware_auto_planning_msgs/msg/Trajectory"
    MSG_CONTROL_COMMAND     = "autoware_auto_control_msgs/msg/AckermannControlCommand"
)
```

### 4.2 生成Autoware消息绑定

使用rclgo为Autoware消息生成Go绑定：

```bash
# 安装Autoware消息包
sudo apt install ros-humble-autoware-auto-msgs

# 使用rclgo生成绑定
cd ${PROJECT_ROOT}
./scripts/generate_interfaces.sh autoware_auto_vehicle_msgs
./scripts/generate_interfaces.sh autoware_auto_planning_msgs
./scripts/generate_interfaces.sh autoware_auto_control_msgs
```

### 4.3 修改ROS Channel

修改`sender/ros_channel/ros_channel.go`以支持新的消息类型：

```go
// 在imports部分添加
autoware_vehicle_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/autoware_auto_vehicle_msgs/msg"
autoware_planning_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/autoware_auto_planning_msgs/msg"
autoware_control_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/autoware_auto_control_msgs/msg"

// 在switch语句中添加新的case
case consts.MSG_CONTROL_MODE_REPORT:
    sub, err := autoware_vehicle_msgs.NewControlModeReportSubscription(
        node,
        "/"+cfg.Topics[i].NameIn,
        &rclgo.SubscriptionOptions{Qos: *(topic.Qos)},
        func(msg *autoware_vehicle_msgs.ControlModeReport, info *rclgo.MessageInfo, err error) {
            messageChan <- msg
        },
    )
    subs[i] = sub.Subscription
    if err != nil {
        panic(err)
    }
// 添加其他消息类型的处理...
```

类似地，修改`receiver/ros_channel/ros_channel.go`以支持发布新的消息类型。

### 4.4 实现消息优先级调度

在`sender/peer_connection_channel/peer_connection_channel.go`中实现消息优先级：

```go
// 定义消息优先级
var messagePriorities = map[string]int{
    consts.MSG_CONTROL_COMMAND: 10,  // 最高优先级
    consts.MSG_VELOCITY_REPORT: 8,
    consts.MSG_TRAJECTORY: 6,
    consts.MSG_IMAGE: 4,
    consts.MSG_POINTCLOUD2: 2,       // 最低优先级
}

// 在消息处理逻辑中应用优先级
```

### 4.5 实现大数据压缩

为点云数据实现压缩策略：

```go
// 点云压缩函数
func compressPointCloud(pc *sensor_msgs.PointCloud2) *sensor_msgs.PointCloud2 {
    // 实现点云降采样或压缩
    // ...
}
```

## 5. 测试计划

1. **单元测试**：
   - 为每种新增的消息类型编写序列化/反序列化测试
   - 测试消息优先级机制

2. **集成测试**：
   - 在本地网络测试全部话题的传输
   - 测量端到端延迟和带宽使用情况

3. **压力测试**：
   - 模拟网络拥塞条件下的性能
   - 测试长时间运行的稳定性

## 6. 部署计划

1. **开发环境配置**：
   - 确保两台机器都安装了ROS2 Humble
   - 安装Autoware和AWSIM

2. **编译和部署**：
   - 在发送端和接收端机器上编译和部署扩展后的WebRTC-ROS桥接系统
   - 配置适当的防火墙规则以允许WebRTC通信

3. **启动步骤**：
   - 先启动接收端（AWSIM侧）
   - 再启动发送端（Autoware侧）
   - 启动Autoware和AWSIM并验证通信

## 7. 监控和优化

1. **性能监控**：
   - 监控网络带宽使用情况
   - 监控消息延迟和丢失率

2. **持续优化**：
   - 根据实际性能数据调整消息优先级
   - 优化压缩算法以提高传输效率 