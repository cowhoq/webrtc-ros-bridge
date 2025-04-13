# 技术上下文

## 开发环境
- 操作系统：Linux 6.8.0-57-generic
- 编程语言：Go 1.21+, C (用于VP8解码)
- ROS版本：ROS2
- 依赖管理：Go Modules

## 主要技术栈
- WebRTC
- rclgo
- VP8编解码器
  - 编码：使用Go mediadevices库
  - 解码：使用C语言libvpx库
- CGO

## 技术约束
- 需要支持实时音视频传输
- 需要保证低延迟
- 需要跨平台兼容性
- 需要支持多种ROS消息类型

## 架构设计决策
### Sender-Receiver架构差异
1. Sender (纯Go实现):
   - 使用mediadevices库处理媒体流
   - 通过webrtc库进行VP8编码
   - 性能依赖硬件加速

2. Receiver (Go+C混合实现):
   - 使用libvpx进行VP8解码
   - 通过CGO实现Go和C的互操作
   - 直接调用C代码以获得更好的解码性能 