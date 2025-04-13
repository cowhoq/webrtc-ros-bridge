package roschannel

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/3DRX/webrtc-ros-bridge/config"
	"github.com/3DRX/webrtc-ros-bridge/consts"
	geom_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/geometry_msgs/msg"
	nav_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/nav_msgs/msg"
	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	"github.com/tiiuae/rclgo/pkg/rclgo"
	"github.com/tiiuae/rclgo/pkg/rclgo/types"

	// 导入Autoware消息类型
	control_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/autoware_control_msgs/msg"
	planning_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/autoware_planning_msgs/msg"
	vehicle_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/autoware_vehicle_msgs/msg"
)

type ROSChannel struct {
	chanDispatcher func()
	messageChan    <-chan types.Message
	cfg            *config.Config
	topicIdx       int
}

func InitROSChannel(
	cfg *config.Config,
	topicIdx int,
	messageChan <-chan types.Message,
) *ROSChannel {
	return &ROSChannel{
		cfg:         cfg,
		topicIdx:    topicIdx,
		messageChan: messageChan,
	}
}

func (r *ROSChannel) Spin() {
	err := rclgo.Init(nil)
	if err != nil {
		panic(err)
	}

	// 创建一个有意义的节点名称
	topicName := r.cfg.Topics[r.topicIdx].NameOut
	topicType := r.cfg.Topics[r.topicIdx].Type

	// 替换不合法字符，ROS节点名称中不能包含 "/" 等特殊字符
	nodeName := "webrtc_ros_bridge_" + r.cfg.Mode + "_" + strings.ReplaceAll(topicType, "/", "_") + "_" + strings.ReplaceAll(topicName, "/", "_")
	node, err := rclgo.NewNode(nodeName, "")
	if err != nil {
		panic(err)
	}
	defer node.Close()
	defer rclgo.Uninit()

	// 创建相应类型的发布者
	switch r.cfg.Topics[r.topicIdx].Type {
	case consts.MSG_IMAGE:
		r.handleImageMessages(node)

	case consts.MSG_LASER_SCAN:
		r.handleLaserScanMessages(node)

	case consts.MSG_KINEMATIC:
		r.handleKinematicMessages(node)

	case consts.MSG_POSE_COV:
		r.handlePoseCovMessages(node)

	// Autoware特定的消息类型 - 当生成绑定后取消注释
	case consts.MSG_CONTROL_CMD:
		r.handleControlCmdMessages(node)

	case consts.MSG_TRAJECTORY:
		r.handleTrajectoryMessages(node)

	case consts.MSG_CONTROL_MODE:
		r.handleControlModeMessages(node)

	case consts.MSG_VELOCITY:
		r.handleVelocityMessages(node)

	case consts.MSG_STEERING:
		r.handleSteeringMessages(node)

	case consts.MSG_GEAR:
		r.handleGearMessages(node)

	default:
		slog.Error("Unsupported message type", "type", r.cfg.Topics[r.topicIdx].Type)
		return
	}
}

// 处理图像消息
func (r *ROSChannel) handleImageMessages(node *rclgo.Node) {
	pub, err := sensor_msgs_msg.NewImagePublisher(node, "/"+r.cfg.Topics[r.topicIdx].NameOut, nil)
	if err != nil {
		panic(err)
	}
	defer pub.Close()

	// FPS计算相关变量
	const windowSize = 30
	timestamps := make([]time.Time, windowSize)
	frameCount := 0
	idx := 0
	firstFrame := true
	lastPrintTime := time.Now()

	for {
		msg := <-r.messageChan
		img, ok := msg.(*sensor_msgs_msg.Image)
		if !ok {
			slog.Error("Received message is not an Image", "type", fmt.Sprintf("%T", msg))
			continue
		}

		now := time.Now()
		err := pub.Publish(img)
		if err != nil {
			slog.Error("Failed to publish image message", "error", err)
		}

		// FPS计算
		if firstFrame {
			timestamps[0] = now
			firstFrame = false
			frameCount = 1
			continue
		}
		idx = (idx + 1) % windowSize
		timestamps[idx] = now
		if frameCount < windowSize {
			frameCount++
		}
		if now.Sub(lastPrintTime) >= time.Second {
			if frameCount < 2 {
				slog.Info("FPS calculation pending, need more frames")
				continue
			}
			oldestIdx := (idx - frameCount + 1 + windowSize) % windowSize
			duration := timestamps[idx].Sub(timestamps[oldestIdx])
			if duration.Seconds() > 0 {
				fps := float64(frameCount-1) / duration.Seconds()
				slog.Info("Current FPS", "fps", fmt.Sprintf("%.2f", fps))
			}
			lastPrintTime = now
		}
	}
}

// 处理激光雷达消息
func (r *ROSChannel) handleLaserScanMessages(node *rclgo.Node) {
	pub, err := sensor_msgs_msg.NewLaserScanPublisher(node, "/"+r.cfg.Topics[r.topicIdx].NameOut, nil)
	if err != nil {
		panic(err)
	}
	defer pub.Close()

	for {
		msg := <-r.messageChan
		scan, ok := msg.(*sensor_msgs_msg.LaserScan)
		if !ok {
			slog.Error("Received message is not a LaserScan", "type", fmt.Sprintf("%T", msg))
			continue
		}

		err := pub.Publish(scan)
		if err != nil {
			slog.Error("Failed to publish laser scan message", "error", err)
		}
	}
}

// 处理运动学状态消息
func (r *ROSChannel) handleKinematicMessages(node *rclgo.Node) {
	pub, err := nav_msgs.NewOdometryPublisher(node, "/"+r.cfg.Topics[r.topicIdx].NameOut, nil)
	if err != nil {
		panic(err)
	}
	defer pub.Close()

	for {
		msg := <-r.messageChan
		odom, ok := msg.(*nav_msgs.Odometry)
		if !ok {
			slog.Error("Received message is not an Odometry", "type", fmt.Sprintf("%T", msg))
			continue
		}

		err := pub.Publish(odom)
		if err != nil {
			slog.Error("Failed to publish odometry message", "error", err)
		}
	}
}

// 处理位姿协方差消息
func (r *ROSChannel) handlePoseCovMessages(node *rclgo.Node) {
	pub, err := geom_msgs.NewPoseWithCovarianceStampedPublisher(node, "/"+r.cfg.Topics[r.topicIdx].NameOut, nil)
	if err != nil {
		panic(err)
	}
	defer pub.Close()

	for {
		msg := <-r.messageChan
		pose, ok := msg.(*geom_msgs.PoseWithCovarianceStamped)
		if !ok {
			slog.Error("Received message is not a PoseWithCovarianceStamped", "type", fmt.Sprintf("%T", msg))
			continue
		}

		err := pub.Publish(pose)
		if err != nil {
			slog.Error("Failed to publish pose message", "error", err)
		}
	}
}

// Autoware特定的消息处理函数 - 当生成绑定后取消注释
func (r *ROSChannel) handleControlCmdMessages(node *rclgo.Node) {
	pub, err := control_msgs.NewControlPublisher(node, "/"+r.cfg.Topics[r.topicIdx].NameOut, nil)
	if err != nil {
		panic(err)
	}
	defer pub.Close()

	for {
		msg := <-r.messageChan
		cmd, ok := msg.(*control_msgs.Control)
		if !ok {
			slog.Error("Received message is not a Control message", "type", fmt.Sprintf("%T", msg))
			continue
		}

		err := pub.Publish(cmd)
		if err != nil {
			slog.Error("Failed to publish control command message", "error", err)
		}
	}
}

func (r *ROSChannel) handleTrajectoryMessages(node *rclgo.Node) {
	pub, err := planning_msgs.NewTrajectoryPublisher(node, "/"+r.cfg.Topics[r.topicIdx].NameOut, nil)
	if err != nil {
		panic(err)
	}
	defer pub.Close()

	for {
		msg := <-r.messageChan
		traj, ok := msg.(*planning_msgs.Trajectory)
		if !ok {
			slog.Error("Received message is not a Trajectory", "type", fmt.Sprintf("%T", msg))
			continue
		}

		err := pub.Publish(traj)
		if err != nil {
			slog.Error("Failed to publish trajectory message", "error", err)
		}
	}
}

func (r *ROSChannel) handleControlModeMessages(node *rclgo.Node) {
	pub, err := vehicle_msgs.NewControlModeReportPublisher(node, "/"+r.cfg.Topics[r.topicIdx].NameOut, nil)
	if err != nil {
		panic(err)
	}
	defer pub.Close()

	for {
		msg := <-r.messageChan
		mode, ok := msg.(*vehicle_msgs.ControlModeReport)
		if !ok {
			slog.Error("Received message is not a ControlModeReport", "type", fmt.Sprintf("%T", msg))
			continue
		}

		err := pub.Publish(mode)
		if err != nil {
			slog.Error("Failed to publish control mode message", "error", err)
		}
	}
}

func (r *ROSChannel) handleVelocityMessages(node *rclgo.Node) {
	pub, err := vehicle_msgs.NewVelocityReportPublisher(node, "/"+r.cfg.Topics[r.topicIdx].NameOut, nil)
	if err != nil {
		panic(err)
	}
	defer pub.Close()

	for {
		msg := <-r.messageChan
		vel, ok := msg.(*vehicle_msgs.VelocityReport)
		if !ok {
			slog.Error("Received message is not a VelocityReport", "type", fmt.Sprintf("%T", msg))
			continue
		}

		err := pub.Publish(vel)
		if err != nil {
			slog.Error("Failed to publish velocity message", "error", err)
		}
	}
}

func (r *ROSChannel) handleSteeringMessages(node *rclgo.Node) {
	pub, err := vehicle_msgs.NewSteeringReportPublisher(node, "/"+r.cfg.Topics[r.topicIdx].NameOut, nil)
	if err != nil {
		panic(err)
	}
	defer pub.Close()

	for {
		msg := <-r.messageChan
		steering, ok := msg.(*vehicle_msgs.SteeringReport)
		if !ok {
			slog.Error("Received message is not a SteeringReport", "type", fmt.Sprintf("%T", msg))
			continue
		}

		err := pub.Publish(steering)
		if err != nil {
			slog.Error("Failed to publish steering message", "error", err)
		}
	}
}

func (r *ROSChannel) handleGearMessages(node *rclgo.Node) {
	pub, err := vehicle_msgs.NewGearReportPublisher(node, "/"+r.cfg.Topics[r.topicIdx].NameOut, nil)
	if err != nil {
		panic(err)
	}
	defer pub.Close()

	for {
		msg := <-r.messageChan
		gear, ok := msg.(*vehicle_msgs.GearReport)
		if !ok {
			slog.Error("Received message is not a GearReport", "type", fmt.Sprintf("%T", msg))
			continue
		}

		err := pub.Publish(gear)
		if err != nil {
			slog.Error("Failed to publish gear message", "error", err)
		}
	}
}
