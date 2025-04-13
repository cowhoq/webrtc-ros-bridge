cmake_minimum_required(VERSION 3.5)
project(rclgo_msg_gen)

find_package(ament_cmake REQUIRED)

foreach(dep autoware_auto_control_msgs)
  find_package(${dep} REQUIRED)
endforeach()

ament_package()
