#!/bin/bash

# 设置颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}开始安装Autoware相关软件包...${NC}"

# 检查是否已经source了ROS2环境
if [ -z "$ROS_DISTRO" ]; then
    echo -e "${YELLOW}ROS2环境未激活，正在source...${NC}"
    source /opt/ros/humble/setup.bash
fi

# 安装Autoware消息包
echo -e "${GREEN}安装Autoware消息包...${NC}"
sudo apt update
sudo apt install -y \
    ros-humble-autoware-auto-msgs \
    ros-humble-autoware-auto-control-msgs \
    ros-humble-autoware-auto-vehicle-msgs \
    ros-humble-autoware-auto-planning-msgs \
    ros-humble-autoware-auto-perception-msgs

# 确认安装成功
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Autoware消息包安装成功!${NC}"
else
    echo -e "${RED}Autoware消息包安装失败!${NC}"
    exit 1
fi

# 查看安装的包
echo -e "${GREEN}已安装的Autoware消息包:${NC}"
apt list --installed | grep autoware-auto

echo -e "${GREEN}现在您可以使用rclgo生成Go绑定了!${NC}"
echo -e "${YELLOW}请运行: ./scripts/generate_interfaces.sh <package_name>${NC}" 