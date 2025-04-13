# Autoware与AWSIM消息传输实施计划

## 1. 当前状态

我们已经完成了WebRTC-ROS桥接系统的基础扩展，使其支持传输高优先级的Autoware消息类型。具体包括：

1. **配置文件更新**
   - 修改了sender.json和receiver.json文件，添加了对高优先级Autoware话题的支持
   - 调整了QoS设置，确保可靠传输

2. **消息类型定义**
   - 在consts/consts.go中添加了高优先级Autoware消息类型的常量定义
   - 现支持控制命令、轨迹、车辆状态和定位信息等核心消息类型

3. **代码结构优化**
   - 重构了sender/ros_channel/ros_channel.go，使其更模块化
   - 重构了receiver/ros_channel/ros_channel.go，为每种消息类型创建了专门的处理函数

4. **自动化工具**
   - 创建了generate_autoware_interfaces.sh脚本，用于自动生成Autoware消息类型的Go绑定

## 2. 下一步任务

要完成整个系统的实施，还需要执行以下任务：

### 2.1 生成Autoware消息绑定

按顺序执行以下命令，生成必要的Autoware消息绑定：

```bash
# 确保脚本有执行权限
chmod +x scripts/generate_autoware_interfaces.sh

# 生成Autoware消息绑定
./scripts/generate_autoware_interfaces.sh

# 或者，如果以上脚本无法找到rclgo-gen工具，请先安装
go install github.com/tiiuae/rclgo/cmd/rclgo-gen@latest
```

### 2.2 启用Autoware消息处理代码

一旦消息绑定生成完成，需要取消sender/ros_channel/ros_channel.go和receiver/ros_channel/ros_channel.go中被注释的Autoware消息处理代码：

1. 取消导入语句的注释
   ```go
   import (
       control_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/autoware_auto_control_msgs/msg"
       planning_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/autoware_auto_planning_msgs/msg"
       vehicle_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/autoware_auto_vehicle_msgs/msg"
   )
   ```

2. 取消相应的消息处理函数的注释
   - 取消case分支中的处理代码
   - 取消相应的handleXXXMessages函数

### 2.3 编译和测试

编译并测试系统：

```bash
# 编译项目
make

# 在两台机器上分别运行发送端和接收端
# 发送端（Autoware侧）
./wrb -c sender.json

# 接收端（AWSIM侧）
./wrb -c receiver.json
```

## 3. 扩展计划

### 3.1 中优先级话题扩展

在高优先级话题成功传输后，下一步是添加中优先级话题（包括传感器数据和感知结果）：

1. 在consts/consts.go中添加新的消息类型常量
2. 更新sender.json和receiver.json文件，添加中优先级话题配置
3. 修改ROSChannel代码，添加对这些消息类型的处理

### 3.2 性能优化

为确保系统在有限带宽下高效运行，应实施以下优化：

1. **消息优先级调度**
   - 实现一个优先级队列，优先传输高优先级消息
   - 在网络拥塞时，可以降低或暂停低优先级消息的传输

2. **数据压缩**
   - 为点云数据实现体素降采样
   - 调整图像质量和分辨率以适应网络条件

3. **自适应传输**
   - 监控网络状况，动态调整消息传输频率
   - 在带宽紧张时，降低图像和点云数据的质量

## 4. 部署指南

### 4.1 系统要求

部署WebRTC-ROS桥接系统需要满足以下要求：

- ROS2 Humble
- Autoware消息包
- Go 1.19+
- rclgo

### 4.2 网络配置

为了确保WebRTC连接正常工作，需要注意以下网络配置：

1. 确保防火墙允许UDP流量通过
2. 在NAT网络中，可能需要使用TURN服务器辅助连接
3. 在配置文件中正确设置发送方和接收方的地址

### 4.3 性能监控

在运行系统时，建议监控以下指标：

1. 消息延迟
2. 带宽使用情况
3. 丢包率
4. CPU和内存使用情况

可以通过添加日志记录和监控代码来收集这些指标。

## 5. 潜在问题与解决方案

### 5.1 消息同步

问题：不同类型的消息可能以不同的频率发布，可能导致接收端处理不同步。

解决方案：
- 实现消息缓冲机制
- 考虑添加时间戳，在接收端按时间顺序处理消息

### 5.2 网络不稳定

问题：网络连接可能不稳定，导致消息丢失或延迟。

解决方案：
- 实现消息重发机制
- 对关键消息使用确认机制
- 实现断线重连功能

### 5.3 消息类型不匹配

问题：如果发送端和接收端的消息定义不完全一致，可能导致反序列化错误。

解决方案：
- 确保两端使用相同版本的消息包
- 添加版本检查和兼容性处理

## 6. 结论

WebRTC-ROS桥接系统的扩展为Autoware和AWSIM之间的远程通信提供了一个强大的解决方案。通过分阶段实施，我们可以确保系统的稳定性和性能。

当前，我们已完成了高优先级消息传输的基础设计，下一步是生成必要的消息绑定，并进行实际测试。在成功传输高优先级消息后，可以逐步添加更多的消息类型，最终实现完整的系统功能。 