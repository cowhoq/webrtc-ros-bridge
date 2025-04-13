package roschannel

import (
	"context"
	"log/slog"

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
	subscriptions []*rclgo.Subscription
	node          *rclgo.Node
}

func InitROSChannel(
	cfg *config.Config,
	messageChan chan<- types.Message,
) *ROSChannel {
	nodeName := "webrtc_ros_bridge_" + cfg.Mode
	slog.Info("creating node", "name", nodeName)
	err := rclgo.Init(nil)
	if err != nil {
		panic(err)
	}
	node, err := rclgo.NewNode(nodeName, "")
	if err != nil {
		panic(err)
	}
	// create subscriptions based on topic types
	subs := make([]*rclgo.Subscription, len(cfg.Topics))
	for i, topic := range cfg.Topics {
		topicPath := "/" + cfg.Topics[i].NameIn
		opts := &rclgo.SubscriptionOptions{Qos: *(topic.Qos)}

		switch topic.Type {
		case consts.MSG_IMAGE:
			imgSub, err := sensor_msgs_msg.NewImageSubscription(
				node,
				topicPath,
				opts,
				func(msg *sensor_msgs_msg.Image, info *rclgo.MessageInfo, err error) {
					messageChan <- msg
				},
			)
			subs[i] = imgSub.Subscription
			if err != nil {
				panic(err)
			}

		case consts.MSG_LASER_SCAN:
			laserScanSub, err := sensor_msgs_msg.NewLaserScanSubscription(
				node,
				topicPath,
				opts,
				func(msg *sensor_msgs_msg.LaserScan, info *rclgo.MessageInfo, err error) {
					messageChan <- msg
				},
			)
			subs[i] = laserScanSub.Subscription
			if err != nil {
				panic(err)
			}

		// Odometry类型的消息 - 运动学状态
		case consts.MSG_KINEMATIC:
			sub, err := nav_msgs.NewOdometrySubscription(
				node,
				topicPath,
				opts,
				func(msg *nav_msgs.Odometry, info *rclgo.MessageInfo, err error) {
					messageChan <- msg
				},
			)
			subs[i] = sub.Subscription
			if err != nil {
				panic(err)
			}

		// 带协方差的位姿
		case consts.MSG_POSE_COV:
			sub, err := geom_msgs.NewPoseWithCovarianceStampedSubscription(
				node,
				topicPath,
				opts,
				func(msg *geom_msgs.PoseWithCovarianceStamped, info *rclgo.MessageInfo, err error) {
					messageChan <- msg
				},
			)
			subs[i] = sub.Subscription
			if err != nil {
				panic(err)
			}

		// Autoware特定的消息类型 - 当生成绑定后取消注释
		case consts.MSG_CONTROL_CMD:
			sub, err := control_msgs.NewControlSubscription(
				node,
				topicPath,
				opts,
				func(msg *control_msgs.Control, info *rclgo.MessageInfo, err error) {
					messageChan <- msg
				},
			)
			subs[i] = sub.Subscription
			if err != nil {
				panic(err)
			}

		case consts.MSG_TRAJECTORY:
			sub, err := planning_msgs.NewTrajectorySubscription(
				node,
				topicPath,
				opts,
				func(msg *planning_msgs.Trajectory, info *rclgo.MessageInfo, err error) {
					messageChan <- msg
				},
			)
			subs[i] = sub.Subscription
			if err != nil {
				panic(err)
			}

		case consts.MSG_CONTROL_MODE:
			sub, err := vehicle_msgs.NewControlModeReportSubscription(
				node,
				topicPath,
				opts,
				func(msg *vehicle_msgs.ControlModeReport, info *rclgo.MessageInfo, err error) {
					messageChan <- msg
				},
			)
			subs[i] = sub.Subscription
			if err != nil {
				panic(err)
			}

		case consts.MSG_VELOCITY:
			sub, err := vehicle_msgs.NewVelocityReportSubscription(
				node,
				topicPath,
				opts,
				func(msg *vehicle_msgs.VelocityReport, info *rclgo.MessageInfo, err error) {
					messageChan <- msg
				},
			)
			subs[i] = sub.Subscription
			if err != nil {
				panic(err)
			}

		case consts.MSG_STEERING:
			sub, err := vehicle_msgs.NewSteeringReportSubscription(
				node,
				topicPath,
				opts,
				func(msg *vehicle_msgs.SteeringReport, info *rclgo.MessageInfo, err error) {
					messageChan <- msg
				},
			)
			subs[i] = sub.Subscription
			if err != nil {
				panic(err)
			}

		case consts.MSG_GEAR:
			sub, err := vehicle_msgs.NewGearReportSubscription(
				node,
				topicPath,
				opts,
				func(msg *vehicle_msgs.GearReport, info *rclgo.MessageInfo, err error) {
					messageChan <- msg
				},
			)
			subs[i] = sub.Subscription
			if err != nil {
				panic(err)
			}

		default:
			slog.Warn("unsupported topic type", "type", topic.Type)
			continue // 跳过不支持的类型
		}
	}
	return &ROSChannel{
		subscriptions: subs,
		node:          node,
	}
}

func (r *ROSChannel) Spin() {
	defer r.node.Close()
	defer func() {
		for _, sub := range r.subscriptions {
			sub.Close()
		}
	}()
	ws, err := rclgo.NewWaitSet()
	if err != nil {
		panic(err)
	}
	defer ws.Close()
	ws.AddSubscriptions(r.subscriptions...)
	ws.Run(context.Background())
}
