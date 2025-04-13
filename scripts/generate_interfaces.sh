#!/bin/bash

# 检查参数
if [ "$#" -lt 1 ]; then
    echo "Usage: $0 <package_name> [more_packages...]"
    echo "Example: $0 sensor_msgs geometry_msgs"
    exit 1
fi

# 项目根目录
PROJECT_ROOT=$(pwd)
WORKSPACE_DIR="$PROJECT_ROOT/rclgo_ws"
OUTPUT_DIR="$PROJECT_ROOT/rclgo_gen"

# 创建工作空间
mkdir -p $WORKSPACE_DIR/src
mkdir -p $OUTPUT_DIR

# 复制package.xml模板
echo "<?xml version=\"1.0\"?>
<?xml-model href=\"http://download.ros.org/schema/package_format2.xsd\" schematypens=\"http://www.w3.org/2001/XMLSchema\"?>
<package format=\"2\">
  <name>rclgo_msg_gen</name>
  <version>0.0.1</version>
  <description>Package for generating ROS2 message bindings for Go</description>
  <maintainer email=\"user@example.com\">User</maintainer>
  <license>MIT</license>
  <buildtool_depend>ament_cmake</buildtool_depend>" > $WORKSPACE_DIR/src/package.xml

# 添加依赖项
for package in "$@"; do
    echo "  <depend>$package</depend>" >> $WORKSPACE_DIR/src/package.xml
done

# 关闭package.xml
echo "</package>" >> $WORKSPACE_DIR/src/package.xml

# 创建CMakeLists.txt
echo "cmake_minimum_required(VERSION 3.5)
project(rclgo_msg_gen)

find_package(ament_cmake REQUIRED)

foreach(dep $@)
  find_package(\${dep} REQUIRED)
endforeach()

ament_package()" > $WORKSPACE_DIR/src/CMakeLists.txt

# 生成接口
cd $WORKSPACE_DIR
source /opt/ros/humble/setup.bash
rclgo-gen -o $OUTPUT_DIR

echo "接口生成完成！接口文件位于: $OUTPUT_DIR" 