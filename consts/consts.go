package consts

const (
	MSG_IMAGE      = "sensor_msgs/msg/Image"
	MSG_LASER_SCAN = "sensor_msgs/msg/LaserScan"

	// 高优先级消息类型
	MSG_CONTROL_CMD  = "autoware_control_msgs/msg/Control"
	MSG_TRAJECTORY   = "autoware_planning_msgs/msg/Trajectory"
	MSG_CONTROL_MODE = "autoware_vehicle_msgs/msg/ControlModeReport"
	MSG_VELOCITY     = "autoware_vehicle_msgs/msg/VelocityReport"
	MSG_STEERING     = "autoware_vehicle_msgs/msg/SteeringReport"
	MSG_GEAR         = "autoware_vehicle_msgs/msg/GearReport"
	MSG_KINEMATIC    = "nav_msgs/msg/Odometry"
	MSG_POSE_COV     = "geometry_msgs/msg/PoseWithCovarianceStamped"
)
