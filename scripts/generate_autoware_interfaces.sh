#!/bin/bash

# 设置颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 检查是否已经source了ROS2环境
if [ -z "$ROS_DISTRO" ]; then
    echo -e "${YELLOW}ROS2环境未激活，正在source...${NC}"
    source /opt/ros/humble/setup.bash
fi

# 检查是否已经source了Autoware环境
if [ ! -f "/home/bupt/autoware/install/setup.bash" ]; then
    echo -e "${RED}错误: Autoware环境未找到!${NC}"
    exit 1
fi

source /home/bupt/autoware/install/setup.bash

echo -e "${GREEN}开始生成Autoware消息的Go绑定...${NC}"

# 项目根目录
PROJECT_ROOT=$(pwd)
OUTPUT_DIR="$PROJECT_ROOT/rclgo_gen"

# 使用绝对路径指定rclgo-gen工具
RCLGO_GEN="/home/bupt/go/bin/rclgo-gen"

if [ ! -f "$RCLGO_GEN" ]; then
    echo -e "${RED}错误: rclgo-gen工具不存在于 $RCLGO_GEN!${NC}"
    echo -e "${YELLOW}请先使用以下命令安装:${NC}"
    echo -e "go install github.com/tiiuae/rclgo/cmd/rclgo-gen@latest"
    exit 1
fi

# 添加Autoware路径
AUTOWARE_PATH="/home/bupt/autoware/install"

# 检查Autoware消息包
echo -e "${GREEN}检查Autoware消息包...${NC}"
PACKAGES=(
    "autoware_control_msgs"
    "autoware_planning_msgs"
    "autoware_vehicle_msgs"
)
for pkg in "${PACKAGES[@]}"; do
    echo -n "检查 $pkg: "
    if [ -d "$AUTOWARE_PATH/$pkg" ]; then
        echo -e "${GREEN}找到${NC}"
    else
        echo -e "${YELLOW}未找到${NC}"
        # 尝试查找实际路径
        echo "尝试查找 $pkg:"
        find "$AUTOWARE_PATH" -path "*/$pkg" -type d | head -n 5
    fi
done

# 列出所有可用的消息类型
echo -e "${GREEN}列出所有可用的Autoware消息类型:${NC}"
find "$AUTOWARE_PATH" -name "*.msg" | grep -E "control|planning|vehicle" | head -n 20

# 生成接口
source /opt/ros/humble/setup.bash
source /home/bupt/autoware/install/setup.bash
echo -e "${GREEN}正在使用rclgo-gen生成Go绑定: ${RCLGO_GEN}${NC}"

$RCLGO_GEN generate \
    --root-path="/opt/ros/humble" \
    --root-path="$AUTOWARE_PATH" \
    --include-package="autoware_vehicle_msgs" \
    --include-package="autoware_control_msgs" \
    --include-package="autoware_planning_msgs" \
    --dest-path="$OUTPUT_DIR" \
    --rclgo-import-path="github.com/tiiuae/rclgo" \
    --message-module-prefix="github.com/3DRX/webrtc-ros-bridge/rclgo_gen"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Go绑定生成成功！接口文件位于: $OUTPUT_DIR${NC}"
else
    echo -e "${RED}生成失败！${NC}"
    exit 1
fi 