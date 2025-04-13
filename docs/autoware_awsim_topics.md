# Autoware与AWSIM关键话题传输计划

根据系统中的话题列表，以下是Autoware和AWSIM之间需要传输的关键话题，按优先级分类。

## 1. 高优先级话题（核心功能）

这些话题对于基本的自动驾驶功能至关重要，必须保证低延迟和高可靠性传输。

### 控制与规划
- `/control/command/control_cmd` - 控制命令，发送给车辆执行
- `/planning/scenario_planning/trajectory` - 规划的轨迹
- `/vehicle/status/control_mode` - 控制模式状态信息

### 车辆状态
- `/vehicle/status/velocity_status` - 车辆速度状态
- `/vehicle/status/steering_status` - 车辆转向状态
- `/vehicle/status/gear_status` - 车辆档位状态

### 定位信息
- `/localization/kinematic_state` - 车辆运动学状态
- `/localization/pose_with_covariance` - 带协方差的位姿信息

## 2. 中优先级话题（增强感知）

这些话题对于完整的感知功能很重要，但可以接受较高的延迟。

### 传感器数据
- `/sensing/lidar/top/pointcloud` - 主激光雷达点云数据
- `/sensing/camera/traffic_light/image_raw` - 交通灯相机图像
- `/sensing/imu/imu_data` - IMU数据
- `/sensing/gnss/pose` - GNSS位置数据

### 感知结果
- `/perception/object_recognition/objects` - 感知到的物体信息
- `/perception/traffic_light_recognition/traffic_signals` - 交通灯识别结果

## 3. 低优先级话题（辅助功能）

这些话题对于辅助功能和可视化有用，但不是核心功能的必需品。

### 调试与可视化
- `/planning/debug/objects_of_interest/*` - 规划中关注的物体
- `/control/trajectory_follower/debug/*` - 轨迹跟踪调试信息
- `/perception/occupancy_grid_map/map` - 占用栅格地图

### 系统状态
- `/system/component_state_monitor/component/*` - 系统组件状态
- `/system/operation_mode/state` - 系统运行模式

## 4. 传输优化策略

为了在有限的网络带宽下有效传输这些话题，建议采取以下策略：

1. **消息优先级**：
   - 高优先级消息应获得最高传输优先级
   - 在带宽受限情况下，可以暂时降低或停止低优先级消息的传输

2. **消息压缩**：
   - 对于大型数据（如点云和图像），实施有效的压缩策略
   - 点云数据可以采用体素降采样减小数据量
   - 图像数据可以使用JPEG或H.264压缩

3. **消息频率控制**：
   - 根据实际需求和网络状况调整消息发布频率
   - 高优先级消息保持高频率（如20-30Hz）
   - 中优先级消息可以降低到5-10Hz
   - 低优先级消息可以降低到1-2Hz

4. **消息过滤**：
   - 对于调试类话题，可以仅传输关键信息或摘要
   - 对于大型数据结构，可以提取最重要的部分传输

## 5. 实施计划

### 阶段1：基础连接（核心功能）
传输高优先级话题以实现基本的远程控制和监视功能：
- 车辆控制命令
- 车辆状态信息
- 定位信息

### 阶段2：增强感知（完整功能）
增加中优先级话题以提供更全面的系统状态和感知数据：
- 传感器原始数据
- 感知结果
- 规划轨迹

### 阶段3：完整系统（全功能）
增加低优先级话题以支持调试和高级功能：
- 调试信息
- 可视化数据
- 系统状态监控

## 6. 技术注意事项

1. **WebRTC带宽**：
   - 监控WebRTC连接的带宽使用情况
   - 实现动态调整机制，根据网络状况调整传输质量

2. **消息序列化**：
   - 为每种消息类型实现高效的序列化/反序列化逻辑
   - 考虑使用Protocol Buffers或FlatBuffers等高效序列化工具

3. **连接稳定性**：
   - 实现心跳机制确保连接稳定
   - 设计断线重连和会话恢复机制

4. **延迟监控**：
   - 监控每种消息类型的端到端延迟
   - 根据延迟数据动态调整传输策略 